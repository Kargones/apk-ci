// Package app содержит основную бизнес-логику приложения benadis-runner.
// Включает функции для работы с базами данных, конвертации файлов,
// управления сервисным режимом и интеграции с системами контроля версий.
package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/dbrestore"
	"github.com/Kargones/apk-ci/internal/entity/one/convert"
	"github.com/Kargones/apk-ci/internal/entity/one/designer"
	"github.com/Kargones/apk-ci/internal/entity/one/edt"
	"github.com/Kargones/apk-ci/internal/entity/one/enterprise"
	"github.com/Kargones/apk-ci/internal/entity/one/store"
	sqEntity "github.com/Kargones/apk-ci/internal/entity/sonarqube"
	"github.com/Kargones/apk-ci/internal/git"
	"github.com/Kargones/apk-ci/internal/service"
	"github.com/Kargones/apk-ci/internal/servicemode"
)

// InitGit инициализирует и настраивает экземпляр Git для работы с репозиторием.
// Создает подключение к Gitea с использованием токена доступа и настраивает
// параметры репозитория для последующих операций клонирования и работы с ветками.
// Параметры:
//   - l: логгер для записи сообщений об ошибках
//   - cfg: конфигурация приложения с настройками Gitea и репозитория
//
// Возвращает:
//   - *git.Git: инициализированный экземпляр Git или nil при ошибке
//   - error: ошибка инициализации или nil при успехе
func InitGit(l *slog.Logger, cfg *config.Config) (*git.Git, error) {
	g := git.Git{}
	connectString := strings.Replace(cfg.GiteaURL, "https://", "https://"+cfg.AccessToken+":@", 1)
	g.RepURL = connectString + "/" + cfg.Owner + "/" + cfg.Repo + ".git"
	g.RepPath = cfg.RepPath
	g.Branch = cfg.BaseBranch
	g.Token = cfg.AccessToken
	g.WorkDir = cfg.WorkDir
	g.Timeout = cfg.GitConfig.Timeout
	if g.RepURL == "" {
		l.Error("Ошибка инициализации Git",
			slog.String("Описание ошибки", "Не задан URL репозитория"),
		)
		return nil, nil
	}
	return &g, nil
}

// Convert выполняет конвертацию проекта 1C:Enterprise из репозитория.
// Клонирует репозиторий, переключается на нужную ветку, загружает конфигурацию
// конвертера и выполняет процесс конвертации файлов проекта.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - error: ошибка выполнения конвертации или nil при успехе
func Convert(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	// Создаем временную директорию
	cfg.RepPath, err = os.MkdirTemp(cfg.WorkDir, "s")
	if err != nil {
		l.Error("Ошибка создания временной директории",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	g, err := InitGit(l, cfg)
	if err != nil {
		return err
	}
	// Клонируем репозиторий полностью, включая все ветки
	g.Branch = ""
	err = g.Clone(ctx, l)
	if err != nil {
		l.Error("Ошибка клонирования репозитория",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	g.Branch = "main"
	if err := g.Switch(*ctx, l); err != nil {
		l.Error("Ошибка переключения на ветку main",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	c := &edt.Convert{}

	err = c.MustLoad(l, cfg)
	if err != nil {
		l.Error("Ошибка инициализации конвертора",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}
	g.Branch = c.Source.Branch
	if err := g.Switch(*ctx, l); err != nil {
		l.Error("Ошибка переключения на исходную ветку",
			slog.String("Описание ошибки", err.Error()),
			slog.String("Ветка", c.Source.Branch),
		)
		return err
	}
	// Создаем контекст с таймаутом для EDT операций
	edtCtx, cancel := context.WithTimeout(*ctx, cfg.AppConfig.EdtTimeout)
	defer cancel()

	err = c.Convert(&edtCtx, l, cfg)
	if err != nil {
		l.Error("Ошибка конвертации",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	return nil
}

// SQProjectUpdate выполняет обновление проекта SonarQube.
// Функция инициализирует обработчик команд SonarQube и выполняет обновление
// метаданных проекта с использованием настроек из конфигурации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения с настройками репозитория
//
// Возвращает:
//   - error: ошибка выполнения или nil при успехе
func SQProjectUpdate(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	// Проверяем конфигурацию на nil
	if cfg == nil {
		return errors.New("configuration cannot be nil")
	}

	l.Info("Starting SonarQube project update",
		slog.String("owner", cfg.Owner),
		slog.String("repo", cfg.Repo),
	)

	// Initialize SonarQube services with structured logging
	// Инициализируем Gitea API клиент с конфигурацией
	giteaAPI := config.CreateGiteaAPI(cfg)
	handler, err := InitSonarQubeServices(l, cfg, giteaAPI)
	if err != nil {
		l.Error("Failed to initialize SonarQube services",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Создаем параметры для обновления проекта
	params := &sqEntity.ProjectUpdateParams{
		Owner: cfg.Owner,
		Repo:  cfg.Repo,
	}

	// Вызываем метод обновления проекта
	err = handler.HandleSQProjectUpdate(*ctx, params)
	if err != nil {
		l.Error("Failed to update SonarQube project",
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

// SQReportBranch выполняет генерацию отчета по ветке SonarQube.
// Функция инициализирует обработчик команд SonarQube и выполняет генерацию
// отчета по указанной ветке с использованием настроек из конфигурации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения с настройками репозитория
//
// Возвращает:
//   - error: ошибка выполнения или nil при успехе
func SQReportBranch(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	// Проверяем конфигурацию на nil
	if cfg == nil {
		return errors.New("configuration cannot be nil")
	}

	l.Info("Starting SonarQube branch report generation",
		slog.String("owner", cfg.Owner),
		slog.String("repo", cfg.Repo),
		slog.String("branch", cfg.BaseBranch),
	)

	// Initialize SonarQube services with structured logging
	// Инициализируем Gitea API клиент с конфигурацией
	giteaAPI := config.CreateGiteaAPI(cfg)
	handler, err := InitSonarQubeServices(l, cfg, giteaAPI)
	if err != nil {
		l.Error("Failed to initialize SonarQube services",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Создаем параметры для генерации отчета по ветке
	params := &sqEntity.ReportBranchParams{
		Owner:           cfg.Owner,
		Repo:            cfg.Repo,
		Branch:          cfg.BaseBranch,
		FirstCommitHash: "", // TODO: Получить первый коммит из конфигурации
		LastCommitHash:  "", // TODO: Получить последний коммит из конфигурации
	}

	// Вызываем метод генерации отчета по ветке
	err = handler.HandleSQReportBranch(*ctx, params)
	if err != nil {
		l.Error("Failed to generate SonarQube branch report",
			slog.String("branch", cfg.BaseBranch),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

// ServiceModeEnable включает сервисный режим для указанной информационной базы.
// Блокирует доступ пользователей к базе данных для выполнения административных операций.
// При необходимости принудительно завершает активные пользовательские сессии.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками подключения
//   - infobaseName: имя информационной базы для включения сервисного режима
//   - terminateSessions: флаг принудительного завершения пользовательских сессий
//
// Возвращает:
//   - error: ошибка включения сервисного режима или nil при успехе
func ServiceModeEnable(ctx *context.Context, l *slog.Logger, cfg *config.Config, infobaseName string, terminateSessions bool) error {
	// Создаем логгер для servicemode
	logger := &servicemode.SlogLogger{Logger: l}

	// Выполняем операцию включения сервисного режима
	err := servicemode.ManageServiceMode(*ctx, "enable", infobaseName, terminateSessions, cfg, logger)
	if err != nil {
		l.Error("Failed to enable service mode", "error", err, "infobase", infobaseName)
		return err
	}

	l.Info("Service mode enabled successfully", "infobase", infobaseName)
	return nil
}

// ServiceModeDisable отключает сервисный режим для указанной информационной базы.
// Восстанавливает нормальный доступ пользователей к базе данных после завершения
// административных операций.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками подключения
//   - infobaseName: имя информационной базы для отключения сервисного режима
//
// Возвращает:
//   - error: ошибка отключения сервисного режима или nil при успехе
func ServiceModeDisable(ctx *context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error {
	// Создаем логгер для servicemode
	logger := &servicemode.SlogLogger{Logger: l}

	// Выполняем операцию отключения сервисного режима
	err := servicemode.ManageServiceMode(*ctx, "disable", infobaseName, false, cfg, logger)
	if err != nil {
		l.Error("Failed to disable service mode", "error", err, "infobase", infobaseName)
		return err
	}

	l.Info("Service mode disabled successfully", "infobase", infobaseName)
	return nil
}

// ServiceModeStatus проверяет текущий статус сервисного режима для указанной информационной базы.
// Возвращает информацию о том, включен ли сервисный режим и какие ограничения действуют.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками подключения
//   - infobaseName: имя информационной базы для проверки статуса
//
// Возвращает:
//   - error: ошибка получения статуса или nil при успехе
func ServiceModeStatus(ctx *context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error {
	// Создаем логгер для servicemode
	logger := &servicemode.SlogLogger{Logger: l}

	// Выполняем операцию получения статуса сервисного режима
	err := servicemode.ManageServiceMode(*ctx, "status", infobaseName, false, cfg, logger)
	if err != nil {
		l.Error("Failed to get service mode status", "error", err, "infobase", infobaseName)
		return err
	}

	return nil
}

// ActServiceModeEnable выполняет активацию сервисного режима с дополнительными проверками.
// Расширенная версия ServiceModeEnable с проверкой состояния базы данных перед включением
// сервисного режима и дополнительным логированием операций.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения
//   - dbName: имя базы данных
//   - terminateSessions: флаг принудительного завершения сессий
//
// Возвращает:
//   - error: ошибка активации сервисного режима или nil при успехе
func ActServiceModeEnable(ctx *context.Context, l *slog.Logger, cfg *config.Config, dbName string, terminateSessions bool) error {
	logger := &servicemode.SlogLogger{Logger: l}

	// Выполняем операцию включения сервисного режима
	l.Debug("Executing service mode enable operation",
		"action", "enable",
		"dbName", dbName,
		"terminateSessions", terminateSessions)
	err := servicemode.ManageServiceMode(*ctx, "enable", dbName, terminateSessions, cfg, logger)
	if err != nil {
		l.Error("Failed to enable service mode",
			"error", err,
			"dbName", dbName,
			"terminateSessions", terminateSessions)
		return err
	}

	l.Info("Service mode enabled successfully",
		"dbName", dbName,
		"terminateSessions", terminateSessions)
	return nil
}

// ActServiceModeDisable выполняет деактивацию сервисного режима с дополнительными проверками.
// Расширенная версия ServiceModeDisable с проверкой состояния базы данных перед отключением
// сервисного режима и дополнительным логированием операций.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения
//   - dbName: имя базы данных
//
// Возвращает:
//   - error: ошибка деактивации сервисного режима или nil при успехе
func ActServiceModeDisable(ctx *context.Context, l *slog.Logger, cfg *config.Config, dbName string) error {
	// Создаем логгер для servicemode
	l.Debug("Creating servicemode logger")
	logger := &servicemode.SlogLogger{Logger: l}

	// Выполняем операцию отключения сервисного режима
	l.Debug("Executing service mode disable operation",
		"action", "disable",
		"dbName", dbName,
		"terminateSessions", false)
	err := servicemode.ManageServiceMode(*ctx, "disable", dbName, false, cfg, logger)
	if err != nil {
		l.Error("Failed to disable service mode",
			"error", err,
			"dbName", dbName)
		return err
	}

	l.Info("Service mode disabled successfully", "dbName", dbName)
	return nil
}

// ActServiceModeStatus получает расширенный статус сервисного режима с дополнительной информацией.
// Расширенная версия ServiceModeStatus с детальной информацией о состоянии базы данных
// и активных ограничениях сервисного режима.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения
//   - dbName: имя базы данных
//   - _: неиспользуемый параметр (для совместимости)
//
// Возвращает:
//   - error: ошибка получения статуса или nil при успехе
func ActServiceModeStatus(ctx *context.Context, l *slog.Logger, cfg *config.Config, dbName string, _ string) error {
	logger := &servicemode.SlogLogger{Logger: l}

	// Выполняем операцию получения статуса сервисного режима
	l.Debug("Executing service mode status operation",
		"action", "status",
		"dbName", dbName)
	err := servicemode.ManageServiceMode(*ctx, "status", dbName, false, cfg, logger)
	if err != nil {
		l.Error("Failed to get service mode status",
			"error", err,
			"dbName", dbName)
		return err
	}

	l.Info("Service mode status retrieved successfully", "dbName", dbName)
	return nil
}

// Git2Store выполняет синхронизацию данных из Git репозитория в хранилище конфигурации 1C.
// Клонирует репозиторий, анализирует изменения и применяет их к хранилищу конфигурации,
// обеспечивая актуальность данных в системе управления версиями 1C:Enterprise.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками репозитория и хранилища
//
// Возвращает:
//   - error: ошибка синхронизации или nil при успехе
func Git2Store(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	NetHaspInit(ctx, l)
	cc := convert.Config{}

	// defer func() {
	// 	_ = cc.StoreUnBind(ctx, l, cfg)
	// }()

	cfg.RepPath, err = os.MkdirTemp(cfg.WorkDir, "s")
	if err != nil {
		l.Error("Ошибка создания временной директории",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	g, err := InitGit(l, cfg)
	if err != nil {
		return err
	}
	err = g.Clone(ctx, l)
	if err != nil {
		l.Error("Ошибка клонирования репозитория",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}
	g.Branch = constants.EdtBranch
	if err := g.Switch(*ctx, l); err != nil {
		l.Error("Ошибка переключения на EDT ветку",
			slog.String("Описание ошибки", err.Error()),
			slog.String("Ветка", constants.EdtBranch),
		)
		return err
	}

	if cfg.ProjectConfig.StoreDb == constants.LocalBase {
		// Генерируем путь для временной базы данных
		dbPath := filepath.Join(cfg.TmpDir, "temp_db_"+time.Now().Format("20060102_150405"))

		// Вызов функции CreateTempDb с правильными параметрами
		oneDb, errTempDbCreate := designer.CreateTempDb(*ctx, l, cfg, dbPath, cfg.AddArray)
		if errTempDbCreate != nil {
			l.Error("Ошибка создания временной базы данных",
				slog.String("Описание ошибки", errTempDbCreate.Error()),
			)
			return errTempDbCreate
		}

		l.Debug("Временная база данных создана успешно",
			slog.String("Строка соединения", oneDb.DbConnectString),
			slog.String("Полная строка соединения", oneDb.FullConnectString),
			slog.Bool("Существует", oneDb.DbExist),
		)
		cc.OneDB = oneDb
	}

	if err = cc.Load(ctx, l, cfg, cfg.InfobaseName); err != nil {
		return err
	}
	// _ = cc.SourceData(ctx, l, cfg)

	// _ = cc.InitDb(ctx, l, cfg, "db1")
	g.Branch = constants.OneCBranch
	if err := g.Switch(*ctx, l); err != nil {
		l.Error("Ошибка переключения на 1C ветку",
			slog.String("Описание ошибки", err.Error()),
			slog.String("Ветка", constants.OneCBranch),
		)
		return err
	}
	if err = cc.InitDb(ctx, l, cfg); err != nil {
		return err
	}
	//ToDo: Добавить отключение базы от хранилища для решения проблемы с блокировкой базы после ошибки в процессе сборки
	err = cc.StoreUnBind(ctx, l, cfg)
	if err != nil {
		l.Error("Ошибка отключения базы от хранилища",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	if err = cc.LoadDb(ctx, l, cfg); err != nil {
		return err
	}
	if err = cc.DbUpdate(ctx, l, cfg); err != nil {
		return err
	}
	if err = cc.DumpDb(ctx, l, cfg); err != nil {
		return err
	}
	if err = cc.StoreBind(ctx, l, cfg); err != nil {
		return err
	}
	if err = cc.DbUpdate(ctx, l, cfg); err != nil {
		return err
	}
	if err = cc.StoreLock(ctx, l, cfg); err != nil {
		return err
	}
	if err = cc.Merge(ctx, l, cfg); err != nil {
		return err
	}
	// Обновление необходимо для расширений
	// ToDo:	Разобраться почему не работает UpdateDBCfg. Возможно надо добавлять еще раз [-Extension <имя расширения>]
	if err = cc.DbUpdate(ctx, l, cfg); err != nil {
		return err
	}
	if err = cc.StoreCommit(ctx, l, cfg); err != nil {
		return err
	}
	return nil
}

// Store2Db выполняет выгрузку конфигурации из хранилища 1C в базу данных.
// Извлекает последнюю версию конфигурации из хранилища и применяет её к указанной
// базе данных, обновляя структуру и данные согласно изменениям в конфигурации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками хранилища и базы данных
//
// Возвращает:
//   - error: ошибка выгрузки конфигурации или nil при успехе
func Store2Db(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	NetHaspInit(ctx, l)
	cc := convert.Config{}
	// defer func() {
	// 	_ = cc.StoreUnBind(ctx, l, cfg)
	// }()
	err = cc.Load(ctx, l, cfg, cfg.InfobaseName)
	if err != nil {
		return err
	}
	err = cc.StoreBind(ctx, l, cfg)
	if err != nil {
		return err
	}
	return err
}

// Store2DbWithConfig выполняет выгрузку конфигурации из хранилища 1C в базу данных с указанным файлом конфигурации.
// Расширенная версия Store2Db, позволяющая использовать пользовательский файл конфигурации
// вместо стандартного, что обеспечивает гибкость в настройке процесса выгрузки.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками хранилища и базы данных
//
// Возвращает:
//   - error: ошибка выгрузки конфигурации или nil при успехе
func Store2DbWithConfig(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	NetHaspInit(ctx, l)

	// Загружаем конфигурацию из переданных данных
	cc, err := convert.LoadFromConfig(ctx, l, cfg)
	if err != nil {
		return err
	}

	err = cc.StoreBind(ctx, l, cfg)
	if err != nil {
		return err
	}
	return err
}

// StoreBind выполняет привязку хранилища конфигурации к базе данных 1C.
// Устанавливает связь между хранилищем конфигурации и базой данных, позволяя
// синхронизировать изменения конфигурации между этими компонентами системы.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками хранилища и базы данных
//
// Возвращает:
//   - error: ошибка привязки хранилища или nil при успехе
func StoreBind(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	NetHaspInit(ctx, l)

	// Загружаем конфигурацию из переданных данных
	cc, err := convert.LoadFromConfig(ctx, l, cfg)
	if err != nil {
		return err
	}

	err = cc.StoreBind(ctx, l, cfg)
	if err != nil {
		return err
	}
	return err
}

// DbUpdate выполняет обновление структуры базы данных 1C до актуальной версии конфигурации.
// Анализирует текущую структуру базы данных и применяет необходимые изменения
// для приведения её в соответствие с загруженной конфигурацией.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками базы данных
//
// Возвращает:
//   - error: ошибка обновления базы данных или nil при успехе
func DbUpdate(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	NetHaspInit(ctx, l)
	cc := convert.Config{}
	// defer func() {
	// 	_ = cc.StoreUnBind(ctx, l, cfg)
	// }()
	err = cc.Load(ctx, l, cfg, cfg.InfobaseName)
	if err != nil {
		return err
	}
	err = cc.DbUpdate(ctx, l, cfg)
	if err != nil {
		return err
	}
	// Второй раз запускаем чтобы обновить расширения
	err = cc.DbUpdate(ctx, l, cfg)
	if err != nil {
		return err
	}
	return err
}

// DbUpdateWithConfig выполняет обновление структуры базы данных 1C с использованием указанного файла конфигурации.
// Расширенная версия DbUpdate, позволяющая использовать пользовательский файл конфигурации
// для более гибкого управления процессом обновления структуры базы данных.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками базы данных
//
// Возвращает:
//   - error: ошибка обновления базы данных или nil при успехе
func DbUpdateWithConfig(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	NetHaspInit(ctx, l)

	// Загружаем конфигурацию из переданных данных
	cc, err := convert.LoadFromConfig(ctx, l, cfg)
	if err != nil {
		return err
	}

	err = cc.DbUpdate(ctx, l, cfg)
	if err != nil {
		return err
	}
	// Второй раз запускаем чтобы обновить расширения
	err = cc.DbUpdate(ctx, l, cfg)
	if err != nil {
		return err
	}
	return err
}

// ExecuteEpf выполняет внешнюю обработку (.epf файл) в 1С:Предприятие.
// Функция использует модуль enterprise для выполнения всех операций.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками подключения
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func ExecuteEpf(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	if cfg == nil {
		return errors.New("конфигурация не может быть nil")
	}

	NetHaspInit(ctx, l)

	// Создаем исполнитель внешних обработок
	executor := enterprise.NewEpfExecutor(l, cfg.WorkDir)

	// Выполняем внешнюю обработку
	return executor.Execute(*ctx, cfg)
}

// NetHaspInit выполняет инициализацию сетевого ключа защиты HASP для 1C:Enterprise.
// Настраивает подключение к серверу лицензий HASP и проверяет доступность
// необходимых лицензий для работы с базой данных 1C.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
func NetHaspInit(_ *context.Context, l *slog.Logger) {
	// Содержимое файла
	content := `[NH_COMMON]
NH_IPX=Disabled
NH_NETBIOS=Disabled        
NH_TCPIP=Enabled
[NH_TCPIP]
NH_SERVER_ADDR = 10.50.69.155
NH_PORT_NUMBER = 475
NH_USE_BROADCAST=Disabled
NH_TCPIP_METHOD = TCP`

	// Путь к файлу
	dirPath := "/opt/1cv8/conf"
	filePath := filepath.Join(dirPath, "nethasp.ini")

	// Валидация пути файла
	if !strings.HasPrefix(filepath.Clean(filePath), "/opt/1cv8/conf/") {
		l.Error("Небезопасный путь к файлу", slog.String("path", filePath))
		return
	}

	// Создаем директорию (если не существует)
	if err := os.MkdirAll(dirPath, 0750); err != nil {
		l.Error("Ошибка создания директории",
			slog.String("Описание ошибки", err.Error()),
		)
		return
	}

	// Создаем файл
	file, err := os.Create(filePath)
	if err != nil {
		l.Error("Ошибка создания файла",
			slog.String("Описание ошибки", err.Error()),
		)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			l.Warn("Failed to close file", slog.String("error", err.Error()))
		}
	}()

	// Записываем содержимое
	if _, err := file.WriteString(content); err != nil {
		l.Error("Ошибка записи в файл",
			slog.String("Описание ошибки", err.Error()),
		)
		return
	}

	l.Debug("Файл успешно создан:",
		slog.String("Путь к файлу", filePath),
	)
}

// DbRestoreWithConfig выполняет восстановление базы данных 1C из резервной копии с использованием указанной конфигурации.
// Создает подключение к базе данных, анализирует параметры резервной копии и выполняет
// процесс восстановления с автоматическим расчетом таймаута операции.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками подключения
//   - dbname: имя базы данных для восстановления
//
// Возвращает:
//   - error: ошибка восстановления базы данных или nil при успехе
func DbRestoreWithConfig(ctx *context.Context, l *slog.Logger, cfg *config.Config, dbname string) error {
	var err error
	NetHaspInit(ctx, l)

	// Создание экземпляра DBRestore
	dbr, err := dbrestore.NewFromConfig(l, cfg, dbname)
	if err != nil {
		l.Error("Ошибка инициализации",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	// Создание контекста с таймаутом
	timeoutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	*ctx = timeoutCtx
	defer cancel()

	// Подключение к базе данных
	err = dbr.Connect(*ctx)
	if err != nil {
		l.Error("Ошибка подключения",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}
	defer func() {
		if closeErr := dbr.Close(); closeErr != nil {
			l.Warn("Failed to close database connection", slog.String("error", closeErr.Error()))
		}
	}()

	l.Info("Successfully connected to database")

	// Получение статистики о бэкапе для автоматического расчета таймаута
	_, err = dbr.GetRestoreStats(*ctx)
	if err != nil {
		l.Warn("Failed to get restore statistics",
			slog.String("error", err.Error()),
		)
	}

	// Создание контекста для восстановления с учетом таймаута
	restoreCtx := *ctx
	if dbr.Timeout > 0 {
		restoreCtx, cancel = context.WithTimeout(context.Background(), dbr.Timeout)
		defer cancel()
	}

	// Выполнение восстановления
	err = dbr.Restore(restoreCtx)
	if err != nil {
		l.Error("Ошибка восстановления базы данных",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	l.Info("Восстановление базы данных завершено успешно")
	return nil
}

// ActionMenuBuildWrapper выполняет построение меню действий для интерфейса пользователя.
// Обертка для функции ActionMenuBuild, обеспечивающая интеграцию с основным модулем app
// и доступ к Gitea API для динамического формирования меню доступных операций.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками API
//
// Возвращает:
//   - error: ошибка построения меню или nil при успехе
func ActionMenuBuildWrapper(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	l.Info("Запуск построения меню действий")

	// Вызов основной функции ActionMenuBuild
	err := ActionMenuBuild(*ctx, l, cfg)
	if err != nil {
		l.Error("Ошибка построения меню действий",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}

	l.Info("Построение меню действий завершено успешно")
	return nil
}

// CreateTempDbWrapper создает временную базу данных 1C с указанными расширениями.
// Обертка для функции CreateTempDb, обеспечивающая интеграцию с основным модулем app
// и автоматическое управление жизненным циклом временных баз данных.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками базы данных
//
// Возвращает:
//   - string: строка подключения к созданной временной базе данных
//   - error: ошибка создания базы данных или nil при успехе
func CreateTempDbWrapper(ctx *context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	l.Debug("Запуск создания временной базы данных")

	// Генерируем путь для временной базы данных
	dbPath := filepath.Join(cfg.TmpDir, "temp_db_"+time.Now().Format("20060102_150405"))

	// Вызов функции CreateTempDb с правильными параметрами
	oneDb, err := designer.CreateTempDb(*ctx, l, cfg, dbPath, cfg.AddArray)
	if err != nil {
		l.Error("Ошибка создания временной базы данных",
			slog.String("Описание ошибки", err.Error()),
		)
		return "", err
	}

	l.Debug("Временная база данных создана успешно",
		slog.String("Строка соединения", oneDb.DbConnectString),
		slog.String("Полная строка соединения", oneDb.FullConnectString),
		slog.Bool("Существует", oneDb.DbExist),
	)
	return oneDb.FullConnectString, nil
}

// CreateStoresWrapper создает хранилища конфигурации 1C для управления версиями.
// Обертка для функции CreateStores, автоматически создающая временную базу данных
// и инициализирующая структуру хранилищ конфигурации для проекта.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения с настройками хранилища
//
// Возвращает:
//   - error: ошибка создания хранилищ или nil при успехе
func CreateStoresWrapper(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	dbConnectString, err := CreateTempDbWrapper(ctx, l, cfg)
	if err != nil {
		l.Error("Ошибка создания временной базы данных",
			slog.String("Описание ошибки", err.Error()),
		)
		return err
	}
	l.Debug("Запуск создания хранилищ конфигурации")

	// Генерируем корневой путь для хранилищ
	// storeRoot := constants.StoreRoot + cfg.Owner + "/" + cfg.Repo + "/"
	storeRoot := filepath.Join(cfg.TmpDir, "store_"+time.Now().Format("20060102_150405"), cfg.Owner, cfg.Repo)

	// Вызов функции CreateStores
	err = store.CreateStores(l, cfg, storeRoot, dbConnectString, cfg.AddArray)
	if err != nil {
		l.Error("Ошибка создания хранилищ конфигурации",
			slog.String("Описание ошибки", err.Error()),
			slog.String("Корневой путь", storeRoot),
		)
		return err
	}

	l.Debug("Хранилища конфигурации созданы успешно",
		slog.String("Корневой путь", storeRoot),
		slog.String("Строка подключения к БД", dbConnectString),
		slog.Any("Массив расширений", cfg.AddArray),
	)
	return nil
}

// shouldRunScanBranch проверяет необходимость запуска сканирования ветки.
// Возвращает true только если:
// 1. Ветка "main" или начинается с "t" и содержит от 6 до 7 цифр
// 2. Коммит содержит изменения в каталогах основной конфигурации или расширений
//
// Параметры:
//   - l: логгер для записи отладочной информации
//   - cfg: конфигурация приложения с параметрами Gitea
//   - branch: имя ветки для проверки
//   - commitHash: SHA коммита для анализа изменений
//
// Возвращает:
//   - bool: true если нужно запускать сканирование, false иначе
//   - error: ошибка проверки или nil при успехе
func shouldRunScanBranch(l *slog.Logger, cfg *config.Config, branch, commitHash string) (bool, error) {
	l.Debug("Проверка необходимости сканирования",
		slog.String("branch", branch),
		slog.String("commitHash", commitHash),
	)

	// Проверка 1: Валидация имени ветки
	if !isValidBranchForScanning(branch) {
		l.Info("Ветка не соответствует критериям для сканирования",
			slog.String("branch", branch),
		)
		return false, nil
	}

	// Если коммит не указан, считаем что сканирование нужно
	if commitHash == "" {
		l.Debug("Коммит не указан, пропускаем проверку изменений")
		return true, nil
	}

	// Проверка 2: Анализ изменений в коммите
	hasRelevantChanges, err := hasRelevantChangesInCommit(l, cfg, branch, commitHash)
	if err != nil {
		l.Error("Ошибка проверки изменений в коммите",
			slog.String("error", err.Error()),
		)
		return false, err
	}

	if !hasRelevantChanges {
		l.Info("Коммит не содержит изменений в каталогах конфигурации",
			slog.String("commitHash", commitHash),
		)
		return false, nil
	}

	l.Debug("Все проверки пройдены, сканирование необходимо")
	return true, nil
}

// isValidBranchForScanning проверяет, соответствует ли имя ветки критериям для сканирования.
// Возвращает true для ветки "main" или веток, начинающихся с "t" и содержащих 6-7 цифр.
func isValidBranchForScanning(branch string) bool {
	// Ветка main всегда сканируется
	if branch == "main" {
		return true
	}

	// Проверяем ветки вида t123456 или t1234567
	if !strings.HasPrefix(branch, "t") {
		return false
	}

	// Извлекаем часть после 't'
	digits := strings.TrimPrefix(branch, "t")

	// Проверяем, что это только цифры и их количество 6-7
	if len(digits) < 6 || len(digits) > 7 {
		return false
	}

	for _, char := range digits {
		if char < '0' || char > '9' {
			return false
		}
	}

	return true
}

// hasRelevantChangesInCommit проверяет, содержит ли коммит изменения в каталогах конфигурации.
// Анализирует структуру проекта и проверяет изменения в основной конфигурации и расширениях.
func hasRelevantChangesInCommit(l *slog.Logger, cfg *config.Config, branch, commitHash string) (bool, error) {
	// Создаем API клиент Gitea
	giteaAPI := config.CreateGiteaAPI(cfg)

	// Анализируем структуру проекта для получения списка каталогов конфигурации
	projectStructure, err := giteaAPI.AnalyzeProject(branch)
	if err != nil {
		return false, fmt.Errorf("ошибка анализа структуры проекта: %v", err)
	}

	if len(projectStructure) == 0 {
		l.Debug("Структура проекта не определена")
		return true, nil // На всякий случай разрешаем сканирование
	}

	// Первый элемент - основная конфигурация, остальные - расширения
	mainConfig := projectStructure[0]
	var configDirs []string
	configDirs = append(configDirs, mainConfig)

	// Добавляем каталоги расширений в формате <основная>.<расширение>
	for i := 1; i < len(projectStructure); i++ {
		extensionDir := mainConfig + "." + projectStructure[i]
		configDirs = append(configDirs, extensionDir)
	}

	l.Debug("Каталоги конфигурации для проверки",
		slog.Any("configDirs", configDirs),
	)

	// Получаем список измененных файлов в коммите
	changedFiles, err := giteaAPI.GetCommitFiles(commitHash)
	if err != nil {
		return false, fmt.Errorf("ошибка получения измененных файлов: %v", err)
	}

	// Проверяем, есть ли изменения в каталогах конфигурации
	for _, file := range changedFiles {
		for _, configDir := range configDirs {
			if strings.HasPrefix(file.Filename, configDir+"/") {
				l.Debug("Найдены изменения в каталоге конфигурации",
					slog.String("file", file.Filename),
					slog.String("configDir", configDir),
				)
				return true, nil
			}
		}
	}

	l.Debug("Изменения в каталогах конфигурации не найдены",
		slog.Int("changedFiles", len(changedFiles)),
	)
	return false, nil
}

// SQScanBranch выполняет сканирование ветки с помощью SonarQube.
// Инициализирует необходимые сервисы и выполняет сканирование указанной ветки.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений и ошибок
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - error: ошибка выполнения сканирования или nil при успехе
func SQScanBranch(ctx *context.Context, l *slog.Logger, cfg *config.Config, commitHash string) error {
	l.Info("Starting SonarQube branch scanning",
		slog.String("owner", cfg.Owner),
		slog.String("repo", cfg.Repo),
		slog.String("branch", cfg.BranchForScan),
	)

	// Проверяем необходимость запуска сканирования
	shouldScan, err := shouldRunScanBranch(l, cfg, cfg.BranchForScan, commitHash)
	if err != nil {
		l.Error("Ошибка проверки необходимости сканирования",
			slog.String("error", err.Error()),
		)
		return err
	}

	if !shouldScan {
		l.Info("Сканирование не требуется, пропускаем",
			slog.String("branch", cfg.BranchForScan),
			slog.String("commitHash", commitHash),
		)
		return nil
	}

	// Initialize SonarQube services with structured logging
	// Инициализируем Gitea API клиент с конфигурацией
	giteaAPI := config.CreateGiteaAPI(cfg)
	handler, err := InitSonarQubeServices(l, cfg, giteaAPI)
	if err != nil {
		l.Error("Failed to initialize SonarQube services",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Создаем параметры для проверки коммитов
	params := &sqEntity.ScanBranchParams{
		Owner:      cfg.Owner,
		Repo:       cfg.Repo,
		Branch:     cfg.BranchForScan,
		CommitHash: commitHash, // Переданный хеш коммита или пустой для последнего коммита
		SourceDir:  "",         // Пока не устанавливаем, так как репозиторий еще не клонирован
	}

	// Проверяем, какие коммиты нужно сканировать
	commitsToScan, err := handler.CheckScanBranch(*ctx, params)
	if err != nil {
		l.Error("Failed to check commits for scanning",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Если нет коммитов для сканирования, возвращаемся
	if len(commitsToScan) == 0 {
		l.Info("No commits to scan, all commits already analyzed")
		return nil
	}

	l.Info("Found commits to scan", slog.Int("count", len(commitsToScan)))

	// Теперь клонируем репозиторий и выполняем сканирование
	// Создаем временный каталог
	tempDir, err := os.MkdirTemp(cfg.WorkDir, "sonar-scan-*")
	if err != nil {
		l.Error("Failed to create temp directory",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Очищаем временный каталог после завершения
	defer func() {
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			l.Warn("Failed to remove temp directory",
				slog.String("path", tempDir),
				slog.String("error", removeErr.Error()),
			)
		}
	}()

	// Клонируем репозиторий во временный каталог
	repoURL := cfg.GiteaURL + "/" + cfg.Owner + "/" + cfg.Repo
	// Клонируем весь репозиторий без фильтрации по веткам чтобы получать первый коммит ветки
	tempRepoPath, err := git.CloneToTempDir(*ctx, l, tempDir, repoURL, "", cfg.AccessToken, 60*time.Minute)
	if err != nil {
		l.Error("Failed to clone repository to temp directory",
			slog.String("repo_url", repoURL),
			slog.String("branch", cfg.BranchForScan),
			slog.String("error", err.Error()),
		)
		return err
	}

	l.Info("Repository cloned successfully",
		slog.String("temp_path", tempRepoPath),
	)

	// Обновляем конфигурацию для использования временного каталога
	originalWorkDir := cfg.WorkDir
	cfg.WorkDir = tempRepoPath
	defer func() {
		cfg.WorkDir = originalWorkDir
	}()

	params.SourceDir = cfg.WorkDir
	err = handler.HandleSQScanBranchWithCommits(*ctx, params, commitsToScan)
	if err != nil {
		l.Error("Failed to scan branch with SonarQube",
			slog.String("branch", cfg.BranchForScan),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

// SQScanPR выполняет сканирование pull request с помощью SonarQube.
// Функция инициализирует обработчик команд SonarQube и выполняет сканирование
// указанного pull request с использованием настроек из конфигурации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения с настройками репозитория
//
// Возвращает:
//   - error: ошибка выполнения или nil при успехе
func SQScanPR(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	// Проверяем конфигурацию на nil
	if cfg == nil {
		return errors.New("configuration cannot be nil")
	}

	l.Info("Starting SonarQube pull request scanning",
		slog.String("owner", cfg.Owner),
		slog.String("repo", cfg.Repo),
		slog.String("branch", cfg.BaseBranch),
	)

	// TODO: Реализовать получение номера PR из конфигурации или параметров
	// Initialize SonarQube services with structured logging
	// Инициализируем Gitea API клиент с конфигурацией
	giteaAPI := config.CreateGiteaAPI(cfg)
	handler, err := InitSonarQubeServices(l, cfg, giteaAPI)
	if err != nil {
		l.Error("Failed to initialize SonarQube services",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Создаем параметры для сканирования pull request
	params := &sqEntity.ScanPRParams{
		Owner: cfg.Owner,
		Repo:  cfg.Repo,
		PR:    0, // TODO: Получить номер PR из конфигурации
	}

	// Вызываем метод сканирования pull request
	err = handler.HandleSQScanPR(*ctx, params)
	if err != nil {
		l.Error("Failed to scan pull request with SonarQube",
			slog.String("branch", cfg.BaseBranch),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

// TestMerge выполняет проверку конфликтов слияния для активных pull request.
// Создает тестовую ветку и проверяет возможность слияния всех активных PR.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: конфигурация приложения с параметрами Gitea
//
// Возвращает:
//   - error: ошибка выполнения или nil при успехе
func TestMerge(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	l.Debug("Начинаем проверку конфликтов слияния",
		slog.String("owner", cfg.Owner),
		slog.String("repo", cfg.Repo),
	)

	// Создание фабрики и сервиса Gitea
	factory := service.NewGiteaFactory()
	giteaService, err := factory.CreateGiteaService(cfg)
	if err != nil {
		l.Error("Ошибка создания Gitea сервиса",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Выполнение проверки конфликтов слияния
	err = giteaService.TestMerge(*ctx, l)
	if err != nil {
		l.Error("Ошибка проверки конфликтов слияния",
			slog.String("error", err.Error()),
		)
		return err
	}

	l.Info("Проверка конфликтов слияния успешно завершена")
	return nil
}
