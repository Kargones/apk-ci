// Package designer предоставляет функциональность для работы с 1С:Предприятие.
package designer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// Константы перенесены в internal/constants/constants.go

// OneDb представляет конфигурацию базы данных 1С:Предприятие.
// Содержит параметры подключения и настройки для работы с информационной базой.
type OneDb struct {
	DbConnectString   string `json:"Строка соединения,omitempty"`
	User              string `json:"Пользователь,omitempty"`
	Pass              string `json:"Пароль,omitempty"`
	FullConnectString string
	ServerDb          bool
	DbExist           bool
}

// Create создает новую информационную базу 1С:Предприятие.
// Выполняет создание базы данных с указанными параметрами подключения.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка создания базы данных, nil при успехе
func (odb *OneDb) Create(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	dbPath, err := GetDbName(ctx, l, cfg)
	if err != nil {
		return err
	}
	r := runner.Runner{}
	r.TmpDir = cfg.WorkDir
	r.RunString = cfg.AppConfig.Paths.BinIbcmd
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "infobase")
	r.Params = append(r.Params, "create")
	r.Params = append(r.Params, "--create-database")
	r.Params = append(r.Params, "--db-path="+dbPath)
	_, err = r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка создания базы данных",
			slog.String("Путь", dbPath),
			slog.String("Error", err.Error()),
		)
		return err
	}
	// Проверяем успешность создания базы данных (поддерживаем русский и английский варианты)
	if !strings.Contains(string(r.ConsoleOut), constants.SearchMsgBaseCreateOk) &&
		!strings.Contains(string(r.ConsoleOut), constants.SearchMsgBaseCreateOkEn) {
		badOutErr := fmt.Errorf("неопознанная ошибка создания базы данных: %s", string(r.ConsoleOut))
		l.Error("Неопознанная ошибка создания базы данных",
			slog.String("Путь", dbPath),
			slog.String("Error", badOutErr.Error()),
		)
		return badOutErr
	}
	odb.DbConnectString = dbPath
	odb.FullConnectString = "/F " + dbPath
	return err
}

// Add добавляет новое расширение в информационную базу.
// Создает расширение с указанным именем в существующей базе данных.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - nameAdd: имя создаваемого расширения
//
// Возвращает:
//   - error: ошибка создания расширения, nil при успехе
func (odb *OneDb) Add(ctx context.Context, l *slog.Logger, cfg *config.Config, nameAdd string) error {
	r := runner.Runner{}
	r.TmpDir = cfg.WorkDir
	r.RunString = cfg.AppConfig.Paths.BinIbcmd
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "extension")
	r.Params = append(r.Params, "create")
	r.Params = append(r.Params, "--db-path="+odb.DbConnectString)
	r.Params = append(r.Params, "--name="+nameAdd)
	//ToDo: заменить на приличное
	r.Params = append(r.Params, "--name-prefix="+"p")
	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка добавления расширения",
			slog.String("Путь", odb.DbConnectString),
			slog.String("Имя", nameAdd),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.ConsoleOut), constants.SearchMsgBaseAddOk) {
		l.Error("Неопознанная ошибка добавления расширения",
			slog.String("Путь", odb.DbConnectString),
			slog.String("Имя", nameAdd),
		)
		return err
	}
	return err
}

// Load загружает конфигурацию из файлов в информационную базу.
// Может загружать как основную конфигурацию, так и расширения при указании имени расширения.
// Параметры:
//   - ctx: контекст выполнения операции (не используется)
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - sourcePath: путь к исходным файлам конфигурации
//   - extensionName: опциональное имя расширения для загрузки
//
// Возвращает:
//   - error: ошибка загрузки конфигурации, nil при успехе
func (odb *OneDb) Load(ctx context.Context, l *slog.Logger, cfg *config.Config, sourcePath string, extensionName ...string) error {
	r := runner.Runner{}
	r.TmpDir = cfg.WorkDir
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, odb.FullConnectString)
	r.Params = append(r.Params, "/LoadConfigFromFiles")
	r.Params = append(r.Params, sourcePath)

	// Если указано имя расширения, добавляем параметры для работы с расширением
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, extensionName[0])
	}

	// r.Params = append(r.Params, "/UpdateDBCfg")

	addDisableParam(&r)
	r.Params = append(r.Params, "/Out")

	// Устанавливаем соответствующее сообщение в зависимости от типа загрузки
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "/c Загрузка из файлов расширения")
	} else {
		r.Params = append(r.Params, "/c Загрузка из файлов основной конфигурации")
	}

	_, err := r.RunCommand(ctx, l)
	if err != nil {
		if len(extensionName) > 0 && extensionName[0] != "" {
			l.Error("Ошибка загрузки файлов расширения",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Источник файлов", sourcePath),
				slog.String("Error", err.Error()),
			)
		} else {
			l.Error("Ошибка загрузки файлов конфигурации",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Источник файлов", sourcePath),
				slog.String("Error", err.Error()),
			)
		}
		return err
	}

	// Проверяем результат выполнения
	if len(extensionName) > 0 && extensionName[0] != "" {
		if !strings.Contains(string(r.FileOut), constants.SearchMsgBaseLoadOk) &&
			!strings.Contains(string(r.FileOut), constants.SearchMsgEmptyFile) &&
			!strings.Contains(string(r.FileOut), constants.InvalidLink) {
			l.Error("Неопознанная ошибка загрузки файлов расширения",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Источник файлов", sourcePath),
			)
			// ToDo: дополнить игнорирование предупреждений
			// return err
		}
	} else {
		if !strings.Contains(string(r.FileOut), constants.SearchMsgBaseLoadOk) &&
			!strings.Contains(string(r.FileOut), constants.SearchMsgEmptyFile) &&
			!strings.Contains(string(r.FileOut), constants.InvalidLink) {
			l.Error("Неопознанная ошибка загрузки файлов конфигурации",
				slog.String("Путь", odb.DbConnectString),
				slog.String("FileOut", string(r.FileOut)),
			)
			// ToDo: дополнить игнорирование предупреждений
			// return err
		}
	}
	return err
}

// LoadAdd загружает расширение в информационную базу.
// Обертка для функции Load с указанием имени расширения для обратной совместимости.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - sourcePath: путь к исходным файлам расширения
//   - name: имя расширения
//
// Возвращает:
//   - error: ошибка загрузки расширения, nil при успехе
func (odb *OneDb) LoadAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, sourcePath string, name string) error {
	return odb.Load(ctx, l, cfg, sourcePath, name)
}

// UpdateCfg обновляет конфигурацию информационной базы.
// Применяет изменения конфигурации к базе данных, может работать с расширениями.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - sourcePath: путь к источнику обновлений
//   - extensionName: опциональное имя расширения для обновления
//
// Возвращает:
//   - error: ошибка обновления конфигурации, nil при успехе
func (odb *OneDb) UpdateCfg(ctx context.Context, l *slog.Logger, cfg *config.Config, _ string, extensionName ...string) error {
	r := runner.Runner{}
	r.TmpDir = cfg.WorkDir
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, odb.FullConnectString)
	r.Params = append(r.Params, "/UpdateDBCfg")

	// Если указано имя расширения, добавляем параметры для работы с расширением
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, extensionName[0])
	}

	addDisableParam(&r)
	r.Params = append(r.Params, "/Out")

	// Устанавливаем соответствующее сообщение в зависимости от типа обновления
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "/c Обновление конфигурации расширения")
	} else {
		r.Params = append(r.Params, "/c Обновление основной конфигурации")
	}

	_, err := r.RunCommand(ctx, l)
	if err != nil {
		l.Error("Ошибка обновления конфигурации расширения",
			slog.String("Путь", odb.DbConnectString),
			slog.String("Error", err.Error()),
		)
		return err
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgBaseLoadOk) && !strings.Contains(string(r.FileOut), constants.SearchMsgEmptyFile) {
		l.Error("Неопознанная ошибка обновления конфигурации расширения",
			slog.String("Путь", odb.DbConnectString),
			slog.String("Вывод файла", runner.TrimOut(r.FileOut)),
		)
		return err
	}
	return err
}

// UpdateAdd обертка для обратной совместимости.
// Вызывает метод UpdateCfg для обновления конфигурации расширения.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - sourcePath: путь к исходным файлам конфигурации
//   - name: имя расширения для обновления
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (odb *OneDb) UpdateAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, sourcePath string, name string) error {
	return odb.UpdateCfg(ctx, l, cfg, sourcePath, name)
}

// Dump выгружает конфигурацию базы данных в файл.
// Если указано имя расширения, выгружает конфигурацию расширения в файл .cfe,
// иначе выгружает основную конфигурацию в файл main.cf.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - extensionName: опциональное имя расширения для выгрузки
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (odb *OneDb) Dump(ctx context.Context, l *slog.Logger, cfg *config.Config, extensionName ...string) error {
	var fileName string
	r := runner.Runner{}
	r.TmpDir = cfg.WorkDir
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, odb.FullConnectString)
	r.Params = append(r.Params, "/DumpDBCfg")

	// Если указано имя расширения, выгружаем расширение
	if len(extensionName) > 0 && extensionName[0] != "" {
		fileName = path.Join(cfg.WorkDir, extensionName[0]+".cfe")
		r.Params = append(r.Params, fileName)
		r.Params = append(r.Params, "-Extension")
		r.Params = append(r.Params, extensionName[0])
	} else {
		fileName = path.Join(cfg.WorkDir, "main.cf")
		r.Params = append(r.Params, fileName)
	}

	addDisableParam(&r)
	r.Params = append(r.Params, "/Out")

	// Устанавливаем соответствующее сообщение в зависимости от типа выгрузки
	if len(extensionName) > 0 && extensionName[0] != "" {
		r.Params = append(r.Params, "/c Выгрузка расширения")
	} else {
		r.Params = append(r.Params, "/c Выгрузка основной конфигурации")
	}

	_, err := r.RunCommand(ctx, l)
	if err != nil {
		if len(extensionName) > 0 && extensionName[0] != "" {
			l.Error("Ошибка выгрузки расширения",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Имя расширения", extensionName[0]),
				slog.String("Имя файла", fileName),
				slog.String("Error", err.Error()),
			)
		} else {
			l.Error("Ошибка выгрузки основной конфигурации",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Имя файла", fileName),
				slog.String("Error", err.Error()),
			)
		}
		return err
	}

	if !strings.Contains(string(r.FileOut), constants.SearchMsgBaseDumpOk) {
		if len(extensionName) > 0 && extensionName[0] != "" {
			l.Error("Неопознанная ошибка выгрузки расширения",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Имя расширения", extensionName[0]),
				slog.String("Имя файла", fileName),
			)
		} else {
			l.Error("Неопознанная ошибка выгрузки основной конфигурации",
				slog.String("Путь", odb.DbConnectString),
				slog.String("Имя файла", fileName),
			)
		}
		return err
	}
	return err
}

// DumpAdd обертка для обратной совместимости.
// Вызывает метод Dump для выгрузки конфигурации расширения.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - name: имя расширения для выгрузки
//
// Возвращает:
//   - error: ошибка выполнения операции или nil при успехе
func (odb *OneDb) DumpAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, name string) error {
	return odb.Dump(ctx, l, cfg, name)
}

// GetDbName создает временную базу данных и возвращает путь к ней.
// Создает временный каталог для базы данных на основе конфигурации подключения.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения с параметрами подключения
//
// Возвращает:
//   - string: путь к созданной временной базе данных
//   - error: ошибка создания каталога или nil при успехе
func GetDbName(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	if cfg.Connect != "" {
		dbPath := path.Join(cfg.TmpDir, cfg.Connect)
		err := os.Mkdir(dbPath, 0750)
		if err != nil {
			l.Error("Не удалось создать временный каталог",
				slog.String("Корень временных каталогов", cfg.TmpDir),
				slog.String("Текст ошибки", err.Error()),
			)
			return "", err
		}
		return dbPath, err
	}

	dbPath, err := os.MkdirTemp(cfg.WorkDir, "db")
	if err != nil {
		l.Error("Не удалось создать временный каталог",
			slog.String("Корень временных каталогов", cfg.WorkDir),
			slog.String("Текст ошибки", err.Error()),
		)
		return "", err
	}
	return dbPath, err
}

// CreateTempDb создает новую временную базу данных 1С в указанном каталоге.
// Создает базу данных по указанному пути и добавляет в неё расширения из массива arrayAdd.
// Использует утилиту ibcmd для создания базы данных и расширений.
// Параметры:
//   - ctx: контекст выполнения операции
//   - logger: логгер для записи сообщений и отладочной информации
//   - cfg: конфигурация приложения с путями к исполняемым файлам
//   - dbPath: путь к каталогу для создания базы данных
//   - arrayAdd: массив имен расширений для добавления в базу данных
//
// Возвращает:
//   - OneDb: заполненную структуру с параметрами созданной базы данных
//   - error: ошибка создания базы данных или расширений, или nil при успехе
func CreateTempDb(ctx context.Context, logger *slog.Logger, cfg *config.Config, dbPath string, arrayAdd []string) (OneDb, error) {
	var odb OneDb
	var err error
	// Создаем базу данных
	r := runner.Runner{}
	r.TmpDir = cfg.TmpDir
	r.RunString = cfg.AppConfig.Paths.BinIbcmd
	r.WorkDir = cfg.WorkDir
	r.Params = append(r.Params, "infobase")
	r.Params = append(r.Params, "create")
	r.Params = append(r.Params, "--create-database")
	r.Params = append(r.Params, "--db-path="+dbPath)

	logger.Debug("Запуск создания базы данных",
		slog.String("command", r.RunString),
		slog.Any("params", r.Params))

	_, err = r.RunCommand(ctx, logger)
	if err != nil {
		logger.Error("Ошибка создания временной базы данных",
			slog.String("error", err.Error()),
			slog.String("db_path", dbPath))
		return odb, fmt.Errorf("ошибка создания временной базы данных: %w", err)
	}

	// Проверяем успешность создания базы данных (поддерживаем русский и английский варианты)
	if !strings.Contains(string(r.ConsoleOut), constants.SearchMsgBaseCreateOk) &&
		!strings.Contains(string(r.ConsoleOut), constants.SearchMsgBaseCreateOkEn) {
		logger.Error("Неопознанная ошибка создания временной базы данных",
			slog.String("output", string(r.ConsoleOut)))
		return odb, fmt.Errorf("неопознанная ошибка создания временной базы данных: %s", string(r.ConsoleOut))
	}

	logger.Debug("База данных успешно создана", slog.String("db_path", dbPath))

	// Заполняем структуру OneDb
	odb.DbConnectString = dbPath
	odb.FullConnectString = "/F " + dbPath
	odb.ServerDb = false
	odb.DbExist = true

	// Добавляем расширения из массива arrayAdd
	for _, extensionName := range arrayAdd {
		if extensionName == "" {
			continue // Пропускаем пустые имена расширений
		}

		logger.Debug("Добавление расширения", slog.String("extension", extensionName))

		// Создаем новый runner для каждого расширения
		extRunner := runner.Runner{}
		extRunner.TmpDir = cfg.TmpDir
		extRunner.RunString = cfg.AppConfig.Paths.BinIbcmd
		extRunner.WorkDir = cfg.WorkDir
		extRunner.Params = append(extRunner.Params, "extension")
		extRunner.Params = append(extRunner.Params, "create")
		extRunner.Params = append(extRunner.Params, "--db-path="+odb.DbConnectString)
		extRunner.Params = append(extRunner.Params, "--name="+extensionName)
		extRunner.Params = append(extRunner.Params, "--name-prefix=p") // Используем стандартный префикс

		logger.Debug("Запуск создания расширения",
			slog.String("extension", extensionName),
			slog.String("command", extRunner.RunString),
			slog.Any("params", extRunner.Params))

		_, err = extRunner.RunCommand(ctx, logger)
		if err != nil {
			logger.Error("Ошибка добавления расширения",
				slog.String("extension", extensionName),
				slog.String("error", err.Error()))
			return odb, fmt.Errorf("ошибка добавления расширения '%s': %w", extensionName, err)
		}

		// Проверяем успешность добавления расширения
		if !strings.Contains(string(extRunner.ConsoleOut), constants.SearchMsgBaseAddOk) {
			logger.Error("Неопознанная ошибка добавления расширения",
				slog.String("extension", extensionName),
				slog.String("output", string(extRunner.ConsoleOut)))
			return odb, fmt.Errorf("неопознанная ошибка добавления расширения '%s': %s", extensionName, string(extRunner.ConsoleOut))
		}

		logger.Debug("Расширение успешно добавлено",
			slog.String("extension", extensionName),
			slog.String("db_path", odb.DbConnectString))
	}

	logger.Debug("Временная база данных успешно создана",
		slog.String("db_path", odb.DbConnectString),
		slog.Int("extensions_count", len(arrayAdd)),
		slog.String("temp_dir", dbPath))

	return odb, nil
}

func addDisableParam(r *runner.Runner) {
	r.Params = append(r.Params, "/DisableStartupDialogs")
	r.Params = append(r.Params, "/DisableStartupMessages")
	r.Params = append(r.Params, "/DisableUnrecoverableErrorMessage")
	r.Params = append(r.Params, "/UC ServiceMode")
}
