// Package convert предоставляет функциональность для конвертации конфигураций 1С
package convert

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path"
	"unicode/utf8"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/one/designer"
	"github.com/Kargones/apk-ci/internal/entity/one/store"
)

// MergeSettingsString - строка настроек слияния конфигураций.
const MergeSettingsString string = `<?xml version="1.0" encoding="UTF-8"?>
<Settings xmlns="http://v8.1c.ru/8.3/config/merge/settings" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.2" platformVersion="8.3.11">
	<Parameters>
		<AllowMainConfigurationObjectDeletion>true</AllowMainConfigurationObjectDeletion>
	</Parameters>
</Settings>`

// Константы перенесены в internal/constants/constants.go

// Config представляет конфигурацию для процесса конвертации данных.
// Содержит параметры источника, назначения и правила преобразования.
// Config представляет конфигурацию для процесса конвертации данных.
// Содержит параметры источника, назначения и правила преобразования.
type Config struct {
	StoreRoot string         `json:"Корень хранилища"`
	OneDB     designer.OneDb `json:"Параметры подключения"`
	Pair      []Pair         `json:"Сопоставления"`
}

// Pair представляет пару "источник-назначение" для операции конвертации.
// Определяет связь между исходными и целевыми данными.
// Pair представляет пару "источник-назначение" для операции конвертации.
// Определяет связь между исходными и целевыми данными.
type Pair struct {
	Source Source      `json:"Источник"`
	Store  store.Store `json:"Хранилище"`
}

// Source представляет источник данных для операции конвертации.
// Содержит информацию о расположении и параметрах доступа к исходным данным.
type Source struct {
	Name    string `json:"Имя"`
	RelPath string `json:"Относительный путь"`
	Main    bool   `json:"Основная конфигурация"`
}

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
func LoadFromConfig(_ context.Context, l *slog.Logger, cfg *config.Config) (*Config, error) {
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
func LoadConfigFromData(_ context.Context, l *slog.Logger, _ *config.Config, configData []byte) (*Config, error) {
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

// ToDo: Обобщить функцию загрузки конфигурации конвертации для любого типа базы данных

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
func (cc *Config) Load(_ context.Context, _ *slog.Logger, cfg *config.Config, _ string) error {
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
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// InitDb инициализирует базу данных для процесса конвертации.
// Создает новую базу данных если она не существует и добавляет в неё все источники конфигурации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка инициализации базы данных, nil при успехе
func (cc *Config) InitDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if cc.OneDB.DbExist {
		return nil
	}
	if cc.OneDB.ServerDb {
		return nil
	}
	err := cc.OneDB.Create(ctx, l, cfg)
	if err != nil {
		return err
	}
	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cc.OneDB.Add(ctx, l, cfg, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// LoadDb загружает конфигурацию из файлов в базу данных.
// Последовательно загружает основные конфигурации и дополнительные расширения.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка загрузки конфигурации, nil при успехе
func (cc *Config) LoadDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cc.OneDB.Load(ctx, l, cfg, path.Join(cfg.RepPath, cp.Source.RelPath))
		if err != nil {
			return err
		}
	}
	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cc.OneDB.LoadAdd(ctx, l, cfg, path.Join(cfg.RepPath, cp.Source.RelPath), cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// DumpDb выгружает конфигурацию из базы данных в файлы.
// Выполняет выгрузку основной конфигурации и всех дополнительных расширений.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка выгрузки конфигурации, nil при успехе
func (cc *Config) DumpDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	err := cc.OneDB.Dump(ctx, l, cfg)
	if err != nil {
		return err
	}
	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cc.OneDB.DumpAdd(ctx, l, cfg, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// StoreLock блокирует объекты в хранилище конфигурации для редактирования.
// Выполняет блокировку основных конфигураций и дополнительных расширений.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка блокировки объектов, nil при успехе
func (cc *Config) StoreLock(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cp.Store.Lock(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cp.Store.LockAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// StoreBind привязывает хранилища конфигурации к базе данных.
// Устанавливает связь между файловыми хранилищами и базой данных для синхронизации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка привязки хранилищ, nil при успехе
func (cc *Config) StoreBind(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		l.Debug("Привязка хранилища к базе данных",
			slog.String("Имя хранилища", cp.Source.Name),
		)
		bindErr := cp.Store.Bind(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, true)
		if bindErr != nil {
			return bindErr
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		l.Debug("Привязка хранилища расширения к базе данных",
			slog.String("Имя хранилища", cp.Source.Name),
		)
		bindAddErr := cp.Store.BindAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, cp.Source.Name)
		if bindAddErr != nil {
			return bindAddErr
		}
	}
	return nil
}

// DbUpdate обновляет конфигурацию базы данных из привязанных хранилищ.
// Синхронизирует изменения из файловых хранилищ в базу данных.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка обновления конфигурации, nil при успехе
func (cc *Config) DbUpdate(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cc.OneDB.UpdateCfg(ctx, l, cfg, cc.OneDB.FullConnectString)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cc.OneDB.UpdateAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// StoreUnBind отвязывает хранилища конфигурации от базы данных.
// Разрывает связь между файловыми хранилищами и базой данных.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка отвязки хранилищ, nil при успехе
func (cc *Config) StoreUnBind(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cp.Store.UnBind(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cp.Store.UnBindAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// StoreCommit фиксирует изменения в хранилище конфигурации.
// Создает новую версию в хранилище с автоматически сгенерированным комментарием.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка фиксации изменений, nil при успехе
func (cc *Config) StoreCommit(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	var err error
	comment := "Версия создана автоматически\nИсходный репозиторий ххх\nДата создания ххх"
	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cp.Store.StoreCommit(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, comment)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cp.Store.StoreCommitAdd(ctx, l, cfg, cc.OneDB.FullConnectString, cc.StoreRoot, comment, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}

// func (s *Store) StoreCommitAdd(ctx context.Context, l *slog.Logger, cfg *config.Config, dbConnectString string, storeRoot string, comment string, addName string) error {

// Merge выполняет слияние конфигурации с хранилищем.
// Объединяет изменения из рабочей директории с версией в хранилище конфигурации.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//
// Возвращает:
//   - error: ошибка слияния конфигурации, nil при успехе
func (cc *Config) Merge(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	mergeSettingFileName, err := mergeSetting(cfg)
	if err != nil {
		l.Error("ошибка создания файла настроек слияния",
			slog.String("Error", err.Error()),
		)
		return fmt.Errorf("ошибка создания файла настроек слияния: %w", err)
	}
	defer func() {
		if removeErr := os.Remove(mergeSettingFileName); removeErr != nil {
			l.Warn("Failed to remove merge settings file", slog.String("file", mergeSettingFileName), slog.String("error", removeErr.Error()))
		}
	}()

	for _, cp := range cc.Pair {
		if !cp.Source.Main {
			continue
		}
		err = cp.Store.Merge(ctx, l, cfg, cc.OneDB.FullConnectString, path.Join(cfg.WorkDir, "main.cf"), mergeSettingFileName, cc.StoreRoot)
		if err != nil {
			return err
		}
	}

	for _, cp := range cc.Pair {
		if cp.Source.Main {
			continue
		}
		err = cp.Store.MergeAdd(ctx, l, cfg, cc.OneDB.FullConnectString, path.Join(cfg.WorkDir, cp.Source.Name+".cfe"), mergeSettingFileName, cc.StoreRoot, cp.Source.Name)
		if err != nil {
			return err
		}
	}
	return err
}
func mergeSetting(cfg *config.Config) (string, error) {
	tFile, err := os.CreateTemp(cfg.WorkDir, "*.xml")
	if err != nil {
		return "", err
	}
	if _, err := tFile.Write([]byte(MergeSettingsString)); err != nil {
		return "", err
	}
	if err := tFile.Close(); err != nil {
		return "", err
	}
	return tFile.Name(), nil
}

// Save сохраняет текущую конфигурацию конвертации в указанный файл.
// Сериализует структуру Config в JSON формат и записывает в файл.
// Параметры:
//   - ctx: контекст выполнения операции (не используется)
//   - l: логгер для записи отладочной информации
//   - cfg: основная конфигурация приложения
//   - configPath: путь к файлу для сохранения конфигурации
//
// Возвращает:
//   - error: ошибка сохранения файла, nil при успехе
func (cc *Config) Save(_ context.Context, _ *slog.Logger, _ *config.Config, configPath string) error {
	ocJSON, err := json.MarshalIndent(cc, "", "\t")
	if err != nil {
		return fmt.Errorf("ошибка сериализации конфигурации: %w", err)
	}
	err = os.WriteFile(configPath, ocJSON, 0600)
	return err
}

/*
	func (cc *Config) SourceData(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
		if len(cc.RepURL) > 10 && cc.RepURL[:4] == "http" {
			g := git.Git{}
			g.RepURL = cc.RepURL
			g.RepPath = path.Join(cfg.TmpDir, "r")
			g.Branch = cc.Branch
			g.CommitSHA1 = cc.CommitSHA1
			err := g.Clone(ctx, l)
			if err != nil {
				l.Error("Ошибка клонирования репозитория",
					slog.String("URL репозитория", g.RepURL),
					slog.String("Каталог приемник", g.RepPath),
					slog.String("Ветка", g.Branch),
					slog.String("Коммит", g.CommitSHA1),
					slog.String("Текст ошибки", err.Error()),
				)
				return err
			}
			cc.SourceRoot = g.RepPath
		} else if ok, _ := exists(cc.RepURL); ok {
			cc.SourceRoot = cc.RepURL
		} else {
			l.Error("Отсутствует источник данных",
				slog.String("URL репозитория", cc.RepURL),
			)
			return fmt.Errorf("отсутствует источник данных %v", cc.RepURL)
		}
		return nil
	}
*/
/*
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}
*/
