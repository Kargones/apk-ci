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
		err := os.Mkdir(dbPath, constants.DirPermStandard)
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
