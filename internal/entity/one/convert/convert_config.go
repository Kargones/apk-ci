package convert

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"unicode/utf8"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/one/designer"
	"github.com/Kargones/apk-ci/internal/entity/one/store"
)

// LoadFromConfig загружает конфигурацию конвертации из указанного файла.
// Читает JSON файл и десериализует его в структуру Config.
// Использует только разрешенные источники: AppConfig, ProjectConfig, SecretConfig, DbConfig и InfobaseName
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - *Config: загруженная конфигурация
//   - error: ошибка чтения или парсинга файла, nil при успехе
func LoadFromConfig(ctx context.Context, l *slog.Logger, cfg *config.Config) (*Config, error) {
	// Формирование StoreRoot из AppConfig
	storeRoot := constants.StoreRoot + cfg.Owner + "/" + cfg.Repo

	// Получение информации о базе данных из DbConfig
	dbInfo := cfg.GetDatabaseInfo(cfg.InfobaseName)
	if dbInfo == nil {
		return nil, fmt.Errorf("база данных %s не найдена в конфигурации", cfg.InfobaseName)
	}

	// Создание OneDB из DbConfig
	oneDB := designer.OneDb{
		DbConnectString:   "/S " + dbInfo.OneServer + "\\" + cfg.InfobaseName,
		User:              cfg.AppConfig.Users.Db,
		Pass:              cfg.SecretConfig.Passwords.Db,
		FullConnectString: "/S " + dbInfo.OneServer + "\\" + cfg.InfobaseName,
		ServerDb:          true,
		DbExist:           true,
	}

	// Добавление пользователя и пароля в полную строку подключения
	if oneDB.User != "" {
		oneDB.FullConnectString = oneDB.FullConnectString + " /N " + oneDB.User
		if oneDB.Pass != "" {
			oneDB.FullConnectString = oneDB.FullConnectString + " /P " + oneDB.Pass
		}
	}

	// Создание Pair из ProjectConfig
	convertPairs := []Pair{}

	// Добавляем основную конфигурацию
	mainPair := Pair{
		Source: Source{
			Name:    cfg.ProjectName,
			RelPath: "src/cfg",
			Main:    true,
		},
		Store: store.Store{
			Name: cfg.ProjectName,
			Path: "Main",
			User: cfg.AppConfig.Users.StoreAdmin,
			Pass: cfg.SecretConfig.Passwords.StoreAdminPassword,
		},
	}
	convertPairs = append(convertPairs, mainPair)

	// Добавляем расширения из ProjectConfig
	for _, addName := range cfg.AddArray {
		addPair := Pair{
			Source: Source{
				Name:    addName,
				RelPath: "src/cfe/" + addName,
				Main:    false,
			},
			Store: store.Store{
				Name: addName,
				Path: "add/" + addName,
				User: cfg.AppConfig.Users.StoreAdmin,
				Pass: cfg.SecretConfig.Passwords.StoreAdminPassword,
			},
		}
		convertPairs = append(convertPairs, addPair)
	}

	// Создание итоговой конфигурации
	cc := &Config{
		StoreRoot: storeRoot,
		OneDB:     oneDB,
		Pair:      convertPairs,
	}

	l.Debug("Загружена конфигурация конвертации",
		slog.String("StoreRoot", cc.StoreRoot),
		slog.String("DbConnectString", cc.OneDB.DbConnectString),
		slog.Int("Pairs", len(cc.Pair)),
	)

	return cc, nil
}

// LoadConfigFromData загружает конфигурацию конвертации из массива байт.
// Десериализует JSON данные в структуру Config.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - configData: JSON данные конфигурации в виде массива байт
// Возвращает:

// getDbUser возвращает пользователя базы данных с обработкой значений по умолчанию
func getDbUser(cfg *struct {
	DbUser     string
	DbPassword string
}) string {
	if cfg.DbUser == "" {
		return constants.DefaultUser
	}
	if cfg.DbUser == "-" {
		return ""
	}
	return cfg.DbUser
}

// getDbPassword возвращает пароль базы данных с обработкой значений по умолчанию
func getDbPassword(cfg *struct {
	DbUser     string
	DbPassword string
}) string {
	if cfg.DbPassword == "" {
		return constants.DefaultPass
	}
	if cfg.DbPassword == "-" {
		return ""
	}
	return cfg.DbPassword
}

// LoadConfigFromData загружает конфигурацию конвертации из массива байт.
// Десериализует JSON данные в структуру Config.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - configData: JSON данные конфигурации в виде массива байт
//
// Возвращает:
//   - *Config: загруженная конфигурация
//   - error: ошибка парсинга JSON, nil при успехе
func LoadConfigFromData(ctx context.Context, l *slog.Logger, _ *config.Config, configData []byte) (*Config, error) {
	// Сначала парсим JSON во временную структуру для обработки пустых значений
	var tempConfig struct {
		Branch     string `json:"Имя ветки"`
		CommitSHA1 string `json:"Хеш коммита"`
		StoreRoot  string `json:"Корень хранилища"`
		OneDB      struct {
			DbConnectString string `json:"Строка соединения"`
			User            string `json:"Пользователь"`
			Pass            string `json:"Пароль"`
		} `json:"Параметры подключения"`
		Pair []Pair `json:"Сопоставления"`
	}

	err := json.Unmarshal(configData, &tempConfig)
	if err != nil {
		l.Error("Не удалось прочитать конфигурацию из JSON-данных",
			slog.String("Описание ошибки", err.Error()),
			slog.String("configData", string(configData)),
		)
		return nil, err
	}

	// Создаем временную структуру для обработки пустых значений
	tempCfg := &struct {
		DbUser     string
		DbPassword string
	}{
		DbUser:     tempConfig.OneDB.User,
		DbPassword: tempConfig.OneDB.Pass,
	}

	// Создаем Config с правильной обработкой пустых значений
	cc := &Config{
		StoreRoot: tempConfig.StoreRoot,
		OneDB: designer.OneDb{
			DbConnectString: tempConfig.OneDB.DbConnectString,
			User:            getDbUser(tempCfg),
			Pass:            getDbPassword(tempCfg),
		},
		Pair: tempConfig.Pair,
	}

	setupDbParams(cc)
	return cc, nil
}

// setupDbParams настраивает параметры подключения к базе данных
func setupDbParams(cc *Config) {
	if utf8.RuneCountInString(cc.OneDB.DbConnectString) > 6 && cc.OneDB.DbConnectString[0:3] == "/S " {
		cc.OneDB.ServerDb = true
		cc.OneDB.DbExist = true
	} else {
		cc.OneDB.ServerDb = false
		if utf8.RuneCountInString(cc.OneDB.DbConnectString) > 8 && cc.OneDB.DbConnectString[0:3] == "/F " && exists(cc.OneDB.DbConnectString[4:]) {
			cc.OneDB.DbExist = true
		} else {
			cc.OneDB.DbExist = false
		}
	}

	// Формирование полной строки подключения
	cc.OneDB.FullConnectString = cc.OneDB.DbConnectString
	if cc.OneDB.User != "" {
		cc.OneDB.FullConnectString = cc.OneDB.FullConnectString + " /N " + cc.OneDB.User
		if cc.OneDB.Pass != "" {
			cc.OneDB.FullConnectString = cc.OneDB.FullConnectString + " /P " + cc.OneDB.Pass
		}
	}
}

// Load загружает данные из всех источников, указанных в конфигурации.
// Выполняет последовательную загрузку данных из каждого источника в соответствующее хранилище.
// Параметры:
//   - ctx: контекст выполнения
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - dbName: имя базы данных
//
// Возвращает:
//   - error: ошибка загрузки данных, nil при успехе
func (cc *Config) Load(ctx context.Context, _ *slog.Logger, cfg *config.Config, _ string) error {
	// Формирование StoreRoot
	cc.StoreRoot = constants.StoreRoot + cfg.Owner + "/" + cfg.Repo

	// Формирование OneDB
	if cfg.ProjectConfig.StoreDb != constants.LocalBase {
		// ToDo: перенести логику в app.go
		if dbInfo := cfg.GetDatabaseInfo(cfg.ProjectConfig.StoreDb); dbInfo != nil {
			cc.OneDB.DbConnectString = "/S " + dbInfo.OneServer + "\\" + cfg.ProjectConfig.StoreDb
			cc.OneDB.FullConnectString = cc.OneDB.DbConnectString
			cc.OneDB.ServerDb = true
			cc.OneDB.DbExist = true
		} else {
			return fmt.Errorf("база данных %s не найдена в конфигурации", cfg.ProjectConfig.StoreDb)
		}
	}
	// Формирование Pair
	cc.Pair = []Pair{}

	// Добавляем основную конфигурацию
	mainPair := Pair{
		Source: Source{
			Name:    cfg.ProjectName,
			RelPath: "src/cfg",
			Main:    true,
		},
		Store: store.Store{
			Name: cfg.ProjectName,
			Path: "Main",
			User: cfg.AppConfig.Users.StoreAdmin,
			Pass: cfg.SecretConfig.Passwords.StoreAdminPassword,
		},
	}
	cc.Pair = append(cc.Pair, mainPair)

	// Добавляем расширения
	for _, addName := range cfg.AddArray {
		addPair := Pair{
			Source: Source{
				Name:    addName,
				RelPath: "src/cfe/" + addName,
				Main:    false,
			},
			Store: store.Store{
				Name: addName,
				Path: "add/" + addName,
				User: cfg.AppConfig.Users.StoreAdmin,
				Pass: cfg.SecretConfig.Passwords.StoreAdminPassword,
			},
		}
		cc.Pair = append(cc.Pair, addPair)
	}
	return nil
}
