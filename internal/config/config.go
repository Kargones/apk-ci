// Package config содержит конфигурацию приложения.
package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"github.com/Kargones/apk-ci/internal/util/runner"

	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

// Все константы перенесены в пакет constants

// AppConfig представляет настройки приложения из файла app.yaml.
// Содержит конфигурацию уровня логирования, рабочих директорий, таймаутов,
// путей к исполняемым файлам и настроек подключения к различным сервисам.
type AppConfig struct {
	LogLevel string `yaml:"logLevel"`
	WorkDir  string `yaml:"workDir"`
	TmpDir   string `yaml:"tmpDir"`
	Timeout  int    `yaml:"timeout"`
	Paths    struct {
		Bin1cv8  string `yaml:"bin1cv8"`
		BinIbcmd string `yaml:"binIbcmd"`
		EdtCli   string `yaml:"edtCli"`
		Rac      string `yaml:"rac"`
	} `yaml:"paths"`
	Rac struct {
		Port    int `yaml:"port"`
		Timeout int `yaml:"timeout"`
		Retries int `yaml:"retries"`
	} `yaml:"rac"`
	Users struct {
		Rac        string `yaml:"rac"`
		Db         string `yaml:"db"`
		Mssql      string `yaml:"mssql"`
		StoreAdmin string `yaml:"storeAdmin"`
	} `yaml:"users"`
	Dbrestore struct {
		Database    string `yaml:"database"`
		Timeout     string `yaml:"timeout"`
		Autotimeout bool   `yaml:"autotimeout"`
	} `yaml:"dbrestore"`
	SonarQube  SonarQubeConfig `yaml:"sonarqube"`
	Scanner    ScannerConfig   `yaml:"scanner"`
	Git        GitConfig       `yaml:"git"`
	EdtTimeout time.Duration   `yaml:"edt_timeout" env:"EDT_TIMEOUT" env-default:"90m"`

	Logging LoggingConfig `yaml:"logging"`
}

// ProjectConfig представляет настройки проекта из файла project.yaml.
// Содержит конфигурацию режима отладки, базы данных хранилища и
// настройки продуктивных баз данных с их связанными компонентами.
type ProjectConfig struct {
	Debug   bool   `yaml:"debug"`
	StoreDb string `yaml:"store-db"`
	Prod    map[string]struct {
		DbName     string                 `yaml:"dbName"`
		AddDisable []string               `yaml:"add-disable"`
		Related    map[string]interface{} `yaml:"related"`
	} `yaml:"prod"`
}

// SecretConfig представляет секретные данные из файла secret.yaml.
// Содержит пароли для различных сервисов и токены доступа к внешним системам.
type SecretConfig struct {
	Passwords struct {
		Rac                string `yaml:"rac"`
		Db                 string `yaml:"db"`
		Mssql              string `yaml:"mssql"`
		StoreAdminPassword string `yaml:"storeAdminPassword"`
		Smb                string `yaml:"smb"`
	} `yaml:"passwords"`
	Gitea struct {
		AccessToken string `yaml:"accessToken"`
	} `yaml:"gitea"`
	SonarQube struct {
		Token string `yaml:"token"`
	} `yaml:"sonarqube"`
}

// DatabaseInfo представляет информацию о базе данных из файла dbconfig.yaml.
// Содержит настройки подключения к серверу 1С, признак продуктивности
// и информацию о сервере базы данных.
type DatabaseInfo struct {
	OneServer string `yaml:"one-server"`
	Prod      bool   `yaml:"prod"`
	DbServer  string `yaml:"dbserver"`
}

// InputParams представляет параметры, переданные через GitHub Actions.
// Содержит все входные параметры из action.yaml, включая настройки базы данных,
// конфигурационные файлы и параметры выполнения команд.
type InputParams struct {
	// GitHub Actions inputs
	GHADbName            string `env:"INPUT_DBNAME" env-default:""`
	GHAConfigSecret      string `env:"INPUT_CONFIGSECRET" env-default:""`
	GHATerminateSessions string `env:"INPUT_TERMINATESESSIONS" env-default:""`
	GHAActor             string `env:"INPUT_ACTOR" env-default:""`
	GHAConfigProject     string `env:"INPUT_CONFIGPROJECT" env-default:""`
	GHACommand           string `env:"INPUT_COMMAND" env-default:""`
	GHAIssueNumber       string `env:"INPUT_ISSUENUMBER" env-default:""`
	GHALogLevel          string `env:"INPUT_LOGLEVEL" env-default:""`
	GHAConfigSystem      string `env:"INPUT_CONFIGSYSTEM" env-default:""`
	GHAConfigDbData      string `env:"INPUT_CONFIGDBDATA" env-default:""`
	GHAAccessToken       string `env:"INPUT_ACCESSTOKEN" env-default:""`
	GHAGiteaURL          string `env:"INPUT_GITEAURL" env-default:""`
	GHARepository        string `env:"INPUT_REPOSITORY" env-default:""`
	GHAForceUpdate       string `env:"INPUT_FORCE_UPDATE" env-default:""`
	GHAMenuMain          string `env:"INPUT_MENUMAIN" env-default:""`
	GHAMenuDebug         string `env:"INPUT_MENUDEBUG" env-default:""`
	GHAStartEpf          string `env:"INPUT_STARTEPF" env-default:""`
	GHABranchForScan     string `env:"INPUT_BRANCHFORSCAN" env-default:""`
	GHACommitHash        string `env:"INPUT_COMMITHASH" env-default:""`
}

// Config - хранит настройки для работы приложения.
// Новая архитектура с разделением на логические группы конфигурации.
type Config struct {
	ProjectName string
	AddArray    []string
	// Системные настройки
	Actor   string `env:"BR_ACTOR" env-default:""`
	Env     string `env:"BR_ENV" env-default:"dev"`
	Command string `env:"BR_COMMAND" env-default:""`
	Logger  *slog.Logger

	// Пути к файлам конфигурации
	ConfigSystem    string `env:"BR_CONFIG_SYSTEM" env-default:""`
	ConfigProject   string `env:"BR_CONFIG_PROJECT" env-default:""`
	ConfigSecret    string `env:"BR_CONFIG_SECRET" env-default:""`
	ConfigDbData    string `env:"BR_CONFIG_DBDATA" env-default:""`
	ConfigMenuMain  string `env:"BR_CONFIG_MENU_MAIN" env-default:""`
	ConfigMenuDebug string `env:"BR_CONFIG_MENU_DEBUG" env-default:""`

	// Настройки приложения (из app.yaml)
	AppConfig *AppConfig

	// Настройки проекта (из project.yaml)
	ProjectConfig *ProjectConfig

	// Секреты (из secret.yaml)
	SecretConfig *SecretConfig

	// Конфигурация баз данных (из dbconfig.yaml)
	DbConfig map[string]*DatabaseInfo

	// SonarQube настройки
	SonarQubeConfig *SonarQubeConfig

	// Scanner настройки
	ScannerConfig *ScannerConfig

	// Git настройки
	GitConfig *GitConfig

	// Logging настройки
	LoggingConfig *LoggingConfig

	// RAC настройки
	RacConfig *RacConfig

	// MenuMain содержит шаблон главного меню как массив строк
	MenuMain []string

	// MenuDebug содержит шаблон меню отладки как массив строк
	MenuDebug []string

	// Дополнительные поля для обратной совместимости
	InfobaseName      string `env:"BR_INFOBASE_NAME" env-default:""`
	TerminateSessions bool   `env:"BR_TERMINATE_SESSIONS" env-default:"false"`
	ForceUpdate       bool   `env:"BR_FORCE_UPDATE" env-default:"false"`
	IssueNumber       int    `env:"BR_ISSUE_NUMBER" env-default:"1"`
	StartEpf          string `env:"BR_START_EPF" env-default:""`
	ExtDir            string `env:"BR_EXT_DIR" env-default:""`
	DryRun            bool   `env:"BR_DRY_RUN" env-default:"false"`
	ReleaseTag        string `env:"GITHUB_REF_NAME" env-default:"main"`

	// Gitea настройки (для обратной совместимости)
	GiteaURL   string
	Owner      string
	Repo       string
	BaseBranch string
	NewBranch  string
	// SonarQube настройки
	BranchForScan string
	CommitHash    string

	// Устаревшие поля (сохранены для обратной совместимости)
	// TODO: Удалить после полной миграции
	Connect     string `env:"Connect_String" env-default:""`
	RepPath     string `env:"RepPath" env-default:"/tmp/4del/rep"`
	WorkDir     string `env:"WorkDir" env-default:"/tmp/4del"`
	TmpDir      string `env:"TmpDir" env-default:"/tmp"`
	AccessToken string `env:"BR_ACCESS_TOKEN" env-default:"aa6a7d04119fc5dc23d8166a2fb4a5b8e967ce73"`
	PathOut     string `env:"RepPath" env-default:"/tmp/4del/xml/Tester"`
	WorkSpace   string `env:"RepPath" env-default:"/tmp/4del/ws01"`
}

// ConvertConfig представляет настройки для конвертации проекта.
// Содержит параметры ветки, коммита, настройки базы данных
// и учетные данные для процесса конвертации.
type ConvertConfig struct {
	Branch     string `json:"branch" yaml:"branch" env:""`
	CommitSHA1 string `json:"commitSHA1" yaml:"commitSHA1" env:""`
	OneDB      bool   `json:"oneDB" yaml:"oneDB" env:""`
	DbUser     string `json:"dbUser" yaml:"dbUser" env:""`
	DbPassword string `json:"dbPassword" yaml:"dbPassword" env:""`
	DbServer   string `json:"dbServer" yaml:"dbServer" env:""`
	DbName     string `json:"dbName" yaml:"dbName" env:""`
}

// DBRestoreConfig представляет настройки для восстановления базы данных.
// Содержит параметры подключения к серверам, пути к резервным копиям,
// таймауты и настройки автоматического определения времени ожидания.
type DBRestoreConfig struct {
	Server      string        `yaml:"server" env:"DBRESTORE_SERVER"`
	User        string        `yaml:"user" env:"DBRESTORE_USER"`
	Password    string        `yaml:"password" env:"DBRESTORE_PASSWORD"`
	Database    string        `yaml:"database" env:"DBRESTORE_DATABASE"`
	Backup      string        `yaml:"backup" env:"DBRESTORE_BACKUP"`
	Timeout     time.Duration `yaml:"timeout" env:"DBRESTORE_TIMEOUT"`
	SrcServer   string        `yaml:"srcServer" env:"DBRESTORE_SRC_SERVER"`
	SrcDB       string        `yaml:"srcDB" env:"DBRESTORE_SRC_DB"`
	DstServer   string        `yaml:"dstServer" env:"DBRESTORE_DST_SERVER"`
	DstDB       string        `yaml:"dstDB" env:"DBRESTORE_DST_DB"`
	Autotimeout bool          `yaml:"autotimeout" env:"DBRESTORE_AUTOTIMEOUT"`
}

// ServiceModeConfig представляет настройки для работы в сервисном режиме.
// Содержит параметры подключения к RAC серверу, учетные данные
// и настройки таймаутов и повторных попыток.
type ServiceModeConfig struct {
	RacPath     string        `yaml:"racPath" env:"SERVICE_RAC_PATH"`
	RacServer   string        `yaml:"racServer" env:"SERVICE_RAC_SERVER"`
	RacPort     int           `yaml:"racPort" env:"SERVICE_RAC_PORT"`
	RacUser     string        `yaml:"racUser" env:"SERVICE_RAC_USER"`
	RacPassword string        `yaml:"racPassword" env:"SERVICE_RAC_PASSWORD"`
	DbUser      string        `yaml:"dbUser" env:"SERVICE_DB_USER"`
	DbPassword  string        `yaml:"dbPassword" env:"SERVICE_DB_PASSWORD"`
	RacTimeout  time.Duration `yaml:"racTimeout" env:"SERVICE_RAC_TIMEOUT"`
	RacRetries  int           `yaml:"racRetries" env:"SERVICE_RAC_RETRIES"`
}

// EdtConfig представляет настройки для работы с EDT (Enterprise Development Tools).
// Содержит пути к исполняемому файлу EDT, рабочей области
// и директории проекта.
type EdtConfig struct {
	EdtCli     string `yaml:"edtCli" env:"EDT_CLI_PATH"`
	Workspace  string `yaml:"workspace" env:"EDT_WORKSPACE"`
	ProjectDir string `yaml:"projectDir" env:"EDT_PROJECT_DIR"`
}

// GitConfig содержит настройки для Git операций.
type GitConfig struct {
	// UserName - имя пользователя Git
	UserName string `yaml:"userName" env:"GIT_USER_NAME"`

	// UserEmail - email пользователя Git
	UserEmail string `yaml:"userEmail" env:"GIT_USER_EMAIL"`

	// DefaultBranch - ветка по умолчанию
	DefaultBranch string `yaml:"defaultBranch" env:"GIT_DEFAULT_BRANCH"`

	// Timeout - таймаут для Git операций
	Timeout time.Duration `yaml:"timeout" env:"GIT_TIMEOUT"`

	// CredentialHelper - настройка credential helper
	CredentialHelper string `yaml:"credentialHelper" env:"GIT_CREDENTIAL_HELPER"`

	// CredentialTimeout - таймаут для кэша credentials
	CredentialTimeout time.Duration `yaml:"credentialTimeout" env:"GIT_CREDENTIAL_TIMEOUT"`
}

// ResilienceConfig содержит настройки для resilience паттернов.

// LoggingConfig содержит настройки для логирования.
type LoggingConfig struct {
	// Level - уровень логирования (debug, info, warn, error)
	Level string `yaml:"level" env:"LOG_LEVEL"`

	// Format - формат логов (json, text)
	Format string `yaml:"format" env:"LOG_FORMAT"`

	// Output - вывод логов (stdout, stderr, file)
	Output string `yaml:"output" env:"LOG_OUTPUT"`

	// FilePath - путь к файлу логов (если output=file)
	FilePath string `yaml:"filePath" env:"LOG_FILE_PATH"`

	// MaxSize - максимальный размер файла лога в MB
	MaxSize int `yaml:"maxSize" env:"LOG_MAX_SIZE"`

	// MaxBackups - максимальное количество backup файлов
	MaxBackups int `yaml:"maxBackups" env:"LOG_MAX_BACKUPS"`

	// MaxAge - максимальный возраст backup файлов в днях
	MaxAge int `yaml:"maxAge" env:"LOG_MAX_AGE"`

	// Compress - сжимать ли backup файлы
	Compress bool `yaml:"compress" env:"LOG_COMPRESS"`
}

// RacConfig содержит настройки для RAC (Remote Administration Console).
type RacConfig struct {
	// RacPath - путь к исполняемому файлу RAC
	RacPath string `yaml:"racPath" env:"RAC_PATH"`

	// RacServer - адрес сервера RAC
	RacServer string `yaml:"racServer" env:"RAC_SERVER"`

	// RacPort - порт сервера RAC
	RacPort int `yaml:"racPort" env:"RAC_PORT"`

	// RacUser - пользователь RAC
	RacUser string `yaml:"racUser" env:"RAC_USER"`

	// RacPassword - пароль пользователя RAC
	RacPassword string `yaml:"racPassword" env:"RAC_PASSWORD"`

	// DbUser - пользователь базы данных
	DbUser string `yaml:"dbUser" env:"RAC_DB_USER"`

	// DbPassword - пароль пользователя базы данных
	DbPassword string `yaml:"dbPassword" env:"RAC_DB_PASSWORD"`

	// Timeout - таймаут для RAC операций
	Timeout time.Duration `yaml:"timeout" env:"RAC_TIMEOUT"`

	// Retries - количество попыток повтора RAC операций
	Retries int `yaml:"retries" env:"RAC_RETRIES"`
}

// Repo представляет информацию о репозитории.
// Содержит настройки репозитория, включая ветку по умолчанию.
type Repo struct {
	DefaultBranch string `json:"default_branch"`
}

// CreateGiteaAPI создает экземпляр Gitea API с конфигурацией из Config
func CreateGiteaAPI(cfg *Config) gitea.APIInterface {
	if cfg == nil {
		return nil
	}

	giteaConfig := gitea.Config{
		GiteaURL:    cfg.GiteaURL,
		Owner:       cfg.Owner,
		Repo:        cfg.Repo,
		AccessToken: cfg.AccessToken,
		BaseBranch:  cfg.BaseBranch,
		NewBranch:   cfg.NewBranch,
		Command:     cfg.Command,
	}
	return gitea.NewGiteaAPI(giteaConfig)
}

// GetInputParams получает параметры ввода из переменных окружения и аргументов командной строки.
// Анализирует переменные окружения и флаги командной строки для формирования
// структуры InputParams с настройками выполнения приложения.
// Возвращает:
//   - *InputParams: указатель на структуру с параметрами ввода
func GetInputParams() *InputParams {
	inputParams := &InputParams{}
	// Загружаем переменные среды в структуру
	if err := cleanenv.ReadEnv(inputParams); err != nil {
		return nil
	}

	return inputParams
}

// validateRequiredParams проверяет наличие обязательных параметров конфигурации
// Возвращает ошибку с описанием отсутствующих параметров
func validateRequiredParams(inputParams *InputParams, l *slog.Logger) error {
	var missingParams []string

	// Проверяем обязательные параметры
	if inputParams.GHAActor == "" {
		missingParams = append(missingParams, "ACTOR")
	}
	if inputParams.GHAGiteaURL == "" {
		missingParams = append(missingParams, "GITEAURL")
	}
	if inputParams.GHARepository == "" {
		missingParams = append(missingParams, "REPOSITORY")
	}
	if inputParams.GHAAccessToken == "" {
		missingParams = append(missingParams, "ACCESSTOKEN")
	}
	if inputParams.GHACommand == "" {
		missingParams = append(missingParams, "COMMAND")
	}

	// Если есть отсутствующие параметры, возвращаем ошибку
	if len(missingParams) > 0 {
		missingParamsStr := strings.Join(missingParams, ", ")
		errorMsg := fmt.Sprintf("Отсутствуют обязательные параметры конфигурации: %s", missingParamsStr)
		l.Error(errorMsg)
		return errors.New(errorMsg)
	}

	return nil
}

// loadGitConfig загружает конфигурацию Git из AppConfig или устанавливает значения по умолчанию
func loadGitConfig(l *slog.Logger, cfg *Config) (*GitConfig, error) {
	// Сначала пытаемся загрузить из AppConfig
	if cfg.AppConfig != nil {
		gitConfig := &cfg.AppConfig.Git
		// Если конфигурация не пустая, используем её
		if gitConfig.UserName != "" || gitConfig.UserEmail != "" {
			l.Info("Git конфигурация загружена из AppConfig")
			return gitConfig, nil
		}
	}

	// Если не удалось загрузить из AppConfig, используем значения по умолчанию
	gitConfig := getDefaultGitConfig()

	// Пытаемся загрузить из переменных окружения
	if err := cleanenv.ReadEnv(gitConfig); err != nil {
		l.Warn("Ошибка загрузки Git конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	return gitConfig, nil
}

// loadLoggingConfig загружает конфигурацию логирования из AppConfig, переменных окружения или устанавливает значения по умолчанию
func loadLoggingConfig(l *slog.Logger, cfg *Config) (*LoggingConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && (cfg.AppConfig.Logging != LoggingConfig{}) {
		return &cfg.AppConfig.Logging, nil
	}

	loggingConfig := getDefaultLoggingConfig()

	if err := cleanenv.ReadEnv(loggingConfig); err != nil {
		l.Warn("Ошибка загрузки Logging конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	return loggingConfig, nil
}

// loadRacConfig загружает конфигурацию RAC из AppConfig, SecretConfig, переменных окружения или устанавливает значения по умолчанию
func loadRacConfig(l *slog.Logger, cfg *Config) (*RacConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil {
		racConfig := &RacConfig{
			RacPath:   cfg.AppConfig.Paths.Rac,
			RacServer: "localhost", // значение по умолчанию
			RacPort:   cfg.AppConfig.Rac.Port,
			RacUser:   cfg.AppConfig.Users.Rac,
			DbUser:    cfg.AppConfig.Users.Db,
			Timeout:   time.Duration(cfg.AppConfig.Rac.Timeout) * time.Second,
			Retries:   cfg.AppConfig.Rac.Retries,
		}

		// Дополняем паролями из SecretConfig, если они есть
		if cfg.SecretConfig != nil {
			if cfg.SecretConfig.Passwords.Rac != "" {
				racConfig.RacPassword = cfg.SecretConfig.Passwords.Rac
			}
			if cfg.SecretConfig.Passwords.Db != "" {
				racConfig.DbPassword = cfg.SecretConfig.Passwords.Db
			}
		}

		// Если основные поля заполнены, возвращаем конфигурацию
		if racConfig.RacPath != "" || racConfig.RacPort != 0 {
			return racConfig, nil
		}
	}

	racConfig := getDefaultRacConfig()

	if err := cleanenv.ReadEnv(racConfig); err != nil {
		l.Warn("Ошибка загрузки RAC конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	return racConfig, nil
}

// getDefaultGitConfig возвращает конфигурацию Git по умолчанию
func getDefaultGitConfig() *GitConfig {
	return &GitConfig{
		UserName:          "benadis-runner",
		UserEmail:         "runner@benadis.ru",
		DefaultBranch:     "main",
		Timeout:           60 * time.Minute,
		CredentialHelper:  "store",
		CredentialTimeout: 24 * time.Hour,
	}
}

// getDefaultLoggingConfig возвращает конфигурацию логирования по умолчанию
func getDefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		FilePath:   "/var/log/benadis-runner.log",
		MaxSize:    100,
		MaxBackups: 3,
		MaxAge:     7,
		Compress:   true,
	}
}

// getDefaultRacConfig возвращает конфигурацию RAC по умолчанию
func getDefaultRacConfig() *RacConfig {
	return &RacConfig{
		RacPath:     "/opt/1cv8/x86_64/8.3.25.1257/rac",
		RacServer:   "localhost",
		RacPort:     1545,
		RacUser:     "",
		RacPassword: "",
		DbUser:      "",
		DbPassword:  "",
		Timeout:     30 * time.Second,
		Retries:     3,
	}
}

// MustLoad загружает конфигурацию приложения из файла или завершает выполнение при ошибке.
// Читает конфигурационный файл, парсит его содержимое и возвращает структуру Config.
// В случае ошибки загрузки приложение завершается с фатальной ошибкой.
// Возвращает:
//   - *Config: указатель на загруженную конфигурацию приложения
//   - error: ошибка загрузки конфигурации или nil при успехе
func MustLoad() (*Config, error) {
	var cfg Config
	var err error

	// Читаем переменные окружения в структуру Config
	if err = cleanenv.ReadEnv(&cfg); err != nil {
		return nil, fmt.Errorf("не удалось прочитать переменные окружения в Config: %w", err)
	}

	// Читаем переданные параметры
	inputParams := GetInputParams()
	if inputParams == nil {
		return nil, errors.New("не удалось получить входные параметры")
	}

	// Получаем пользователя который запустил процесс
	cfg.Actor = inputParams.GHAActor

	// Инициализируем логгер
	l := getSlog(cfg.Actor, inputParams.GHALogLevel)
	cfg.Logger = l
	l.Debug("inputParams", "inputParams", inputParams)
	// Проверяем обязательные параметры перед загрузкой конфигурации
	if err = validateRequiredParams(inputParams, l); err != nil {
		return nil, err
	}

	cfg.GiteaURL = inputParams.GHAGiteaURL
	cfg.Command = inputParams.GHACommand
	issueNumStr := inputParams.GHAIssueNumber
	if issueNumStr != "" {
		if num, numErr := strconv.Atoi(issueNumStr); numErr == nil {
			cfg.IssueNumber = num
		}
	}

	cfg.ConfigSystem = inputParams.GHAConfigSystem
	cfg.ConfigProject = inputParams.GHAConfigProject
	cfg.ConfigSecret = inputParams.GHAConfigSecret
	cfg.ConfigDbData = inputParams.GHAConfigDbData
	cfg.ConfigMenuMain = inputParams.GHAMenuMain
	cfg.ConfigMenuDebug = inputParams.GHAMenuDebug

	cfg.AccessToken = inputParams.GHAAccessToken
	cfg.GiteaURL = inputParams.GHAGiteaURL
	cfg.InfobaseName = inputParams.GHADbName
	cfg.TerminateSessions = inputParams.GHATerminateSessions == "true"
	cfg.ForceUpdate = inputParams.GHAForceUpdate == "true"
	cfg.StartEpf = inputParams.GHAStartEpf

	repositoryInput := inputParams.GHARepository
	cfg.Owner = getOwner(repositoryInput)

	cfg.Repo = getRepo(repositoryInput)
	cfg.BaseBranch = constants.BaseBranch

	cfg.NewBranch = constants.TestBranch

	cfg.BranchForScan = inputParams.GHABranchForScan
	cfg.CommitHash = inputParams.GHACommitHash

	// Загрузка конфигурации приложения
	if cfg.AppConfig, err = loadAppConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации приложения", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.AppConfig = getDefaultAppConfig()
	}

	// Загрузка конфигурации проекта
	if cfg.ProjectConfig, err = loadProjectConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации проекта", slog.String("error", err.Error()))
	}

	// Загрузка секретов
	if cfg.SecretConfig, err = loadSecretConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки секретов", slog.String("error", err.Error()))
	}

	// Загрузка конфигурации баз данных
	if cfg.DbConfig, err = loadDbConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации БД", slog.String("error", err.Error()))
	}

	// Загрузка конфигурации главного меню
	if cfg.MenuMain, err = loadMenuMainConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации главного меню", slog.String("error", err.Error()))
	}

	// Загрузка конфигурации меню отладки
	if cfg.MenuDebug, err = loadMenuDebugConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации меню отладки", slog.String("error", err.Error()))
	}

	// Загрузка конфигурации SonarQube из переменных окружения
	if cfg.SonarQubeConfig, err = GetSonarQubeConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации SonarQube", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.SonarQubeConfig = GetDefaultSonarQubeConfig()
	}

	// Загрузка конфигурации сканера из переменных окружения
	if cfg.ScannerConfig, err = GetScannerConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации сканера", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.ScannerConfig = GetDefaultScannerConfig()
	}

	// Загрузка конфигурации Git
	if cfg.GitConfig, err = loadGitConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации Git", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.GitConfig = getDefaultGitConfig()
	}

	// Загрузка конфигурации логирования
	if cfg.LoggingConfig, err = loadLoggingConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации логирования", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.LoggingConfig = getDefaultLoggingConfig()
	}

	// Загрузка конфигурации RAC
	if cfg.RacConfig, err = loadRacConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации RAC", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.RacConfig = getDefaultRacConfig()
	}

	// l.Debug("Config", "Параметры конфигурации", cfg)
	err = runner.DisplayConfig(l)
	if err != nil {
		l.Error("Ошибка настройки виртуального дисплея",
			slog.String("Описание ошибки", err.Error()),
		)
	}
	cfg.WorkDir = constants.WorkDir
	cfg.TmpDir = constants.TempDir

	err = createWorkDirectories(&cfg)
	if err != nil {
		l.Error("Ошибка создания рабочих директорий",
			slog.String("Описание ошибки", err.Error()),
		)
	}

	// Заполнение данных проекта
	err = cfg.AnalyzeProject(l, constants.BaseBranch)
	if err != nil {
		l.Error("Ошибка анализа проекта",
			slog.String("Описание ошибки", err.Error()),
		)
		return nil, err
	}
	return &cfg, nil
}

func getSlog(actor string, logLevel string) *slog.Logger {
	var programLevel = new(slog.LevelVar)
	boLogLevel := constants.LogLevelDefault

	if actor == constants.DebugUser {
		boLogLevel = logLevel
		if boLogLevel == "" {
			boLogLevel = constants.LogLevelDefault
		}
	}
	switch boLogLevel {
	default:
		programLevel.Set(slog.LevelInfo)
	case constants.LogLevelDebug:
		programLevel.Set(slog.LevelDebug)
	case constants.LogLevelInfo:
		programLevel.Set(slog.LevelInfo)
	case constants.LogLevelWarn:
		programLevel.Set(slog.LevelWarn)
	case constants.LogLevelError:
		programLevel.Set(slog.LevelError)
	}

	l := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
		Level:     programLevel,
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.SourceKey {
				s := a.Value.Any().(*slog.Source)
				s.File = path.Base(s.File)
			}
			return a
		},
	}))
	l = l.With(slog.Group("App info",
		slog.String("version", constants.Version),
	))
	return l
}

func getOwner(repository string) string {
	substrings := strings.Split(repository, "/")
	return substrings[0]
}
func getRepo(repository string) string {
	substrings := strings.Split(repository, "/")
	if len(substrings) == 2 {
		return substrings[1]
	}
	return ""
}

// LoadDBRestoreConfig загружает конфигурацию для восстановления базы данных из указанного файла.
// Читает и парсит YAML файл с настройками восстановления базы данных,
// включая параметры подключения и настройки процесса восстановления.
// Параметры:
//   - cfg: указатель на основную конфигурацию приложения
//   - dbName: имя базы данных для восстановления
//
// Возвращает:
//   - *DBRestoreConfig: указатель на конфигурацию восстановления базы данных
//   - error: ошибка загрузки конфигурации или nil при успехе
func LoadDBRestoreConfig(cfg *Config, dbName string) (*DBRestoreConfig, error) {
	dbCfg := &DBRestoreConfig{
		// Значения по умолчанию
		Server:  "localhost",
		Timeout: 30 * time.Second,
	}

	// Загрузка настроек из app.yaml
	if cfg.AppConfig != nil {
		if cfg.AppConfig.Dbrestore.Database != "" {
			dbCfg.Database = cfg.AppConfig.Dbrestore.Database
		}
		if cfg.AppConfig.Dbrestore.Timeout != "" {
			if timeout, err := time.ParseDuration(cfg.AppConfig.Dbrestore.Timeout); err == nil {
				dbCfg.Timeout = timeout
			}
		}
		dbCfg.Autotimeout = cfg.AppConfig.Dbrestore.Autotimeout
	}

	// Загрузка пароля из secret.yaml
	if cfg.SecretConfig != nil && cfg.SecretConfig.Passwords.Mssql != "" {
		dbCfg.Password = cfg.SecretConfig.Passwords.Mssql
	}

	// Определение srcServer и dstServer на основе алгоритма
	if dbName != "" {
		srcServer, dstServer, srcDB, dstDB, err := cfg.DetermineSrcAndDstServers(dbName)
		if err != nil {
			return nil, fmt.Errorf("failed to determine src and dst servers: %w", err)
		}
		dbCfg.SrcServer = srcServer
		dbCfg.DstServer = dstServer
		dbCfg.SrcDB = srcDB
		dbCfg.DstDB = dstDB
	}

	// Загрузка из переменных окружения (перезаписывает значения из конфигурации)
	if err := cleanenv.ReadEnv(dbCfg); err != nil {
		return nil, fmt.Errorf("failed to read dbrestore config from env: %w", err)
	}

	// Обработка пароля из переменной окружения MSSQL_PASSWORD
	if mssqlPassword := os.Getenv("MSSQL_PASSWORD"); mssqlPassword != "" {
		dbCfg.Password = mssqlPassword
	}

	return dbCfg, nil
}

// LoadServiceModeConfig загружает конфигурацию сервисного режима для указанной базы данных.
// Создает конфигурацию для управления сервисным режимом базы данных 1C,
// включая настройки подключения и параметры блокировки пользователей.
// Параметры:
//   - dbName: имя базы данных для настройки сервисного режима
//
// Возвращает:
//   - *ServiceModeConfig: указатель на конфигурацию сервисного режима
//   - error: ошибка создания конфигурации или nil при успехе
func (cfg *Config) LoadServiceModeConfig(dbName string) (*ServiceModeConfig, error) {
	serviceCfg := &ServiceModeConfig{
		// Значения по умолчанию из AppConfig
		RacTimeout: time.Duration(30) * time.Second,
		RacRetries: 3,
	}

	// Загружаем настройки из AppConfig если доступны
	if cfg.AppConfig != nil {
		serviceCfg.RacPath = cfg.AppConfig.Paths.Rac
		serviceCfg.RacPort = cfg.AppConfig.Rac.Port
		serviceCfg.RacTimeout = time.Duration(cfg.AppConfig.Rac.Timeout) * time.Second
		serviceCfg.RacRetries = cfg.AppConfig.Rac.Retries
		serviceCfg.RacUser = cfg.AppConfig.Users.Rac
		serviceCfg.DbUser = cfg.AppConfig.Users.Db
	} else {
		return nil, fmt.Errorf("app config is not loaded")
	}

	// Получаем RAC сервер для конкретной базы данных
	if dbName != "" {
		if racServer := cfg.GetRacServerForDb(dbName); racServer != "" {
			serviceCfg.RacServer = racServer
		}
	}
	// Загружаем пароли из SecretConfig если доступны
	if cfg.SecretConfig != nil {
		serviceCfg.RacPassword = cfg.SecretConfig.Passwords.Rac
		serviceCfg.DbPassword = cfg.SecretConfig.Passwords.Db
	}

	return serviceCfg, nil
}

// loadAppConfig загружает конфигурацию приложения из app.yaml
func loadAppConfig(l *slog.Logger, cfg *Config) (*AppConfig, error) {
	giteaAPI := CreateGiteaAPI(cfg)
	data, err := giteaAPI.GetConfigData(l, cfg.ConfigSystem)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных app.yaml: %w", err)
	}

	var appConfig AppConfig
	if err = yaml.Unmarshal(data, &appConfig); err != nil {
		return nil, fmt.Errorf("ошибка парсинга app.yaml: %w", err)
	}

	// Применяем значения по умолчанию для нулевых значений
	if appConfig.EdtTimeout <= 0 {
		appConfig.EdtTimeout = 90 * time.Minute
	}

	return &appConfig, nil
}

// loadProjectConfig загружает конфигурацию проекта из project.yaml
func loadProjectConfig(l *slog.Logger, cfg *Config) (*ProjectConfig, error) {
	giteaAPI := CreateGiteaAPI(cfg)
	data, err := giteaAPI.GetConfigData(l, cfg.ConfigProject)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных project.yaml: %w", err)
	}

	var projectConfig ProjectConfig
	if err = yaml.Unmarshal(data, &projectConfig); err != nil {
		return nil, fmt.Errorf("ошибка парсинга project.yaml: %w", err)
	}

	return &projectConfig, nil
}

// loadSecretConfig загружает секреты из secret.yaml
func loadSecretConfig(l *slog.Logger, cfg *Config) (*SecretConfig, error) {
	giteaAPI := CreateGiteaAPI(cfg)
	data, err := giteaAPI.GetConfigData(l, cfg.ConfigSecret)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных secret.yaml: %w", err)
	}

	var secretConfig SecretConfig
	if err = yaml.Unmarshal(data, &secretConfig); err != nil {
		return nil, fmt.Errorf("ошибка парсинга secret.yaml: %w", err)
	}

	return &secretConfig, nil
}

// loadDbConfig загружает конфигурацию баз данных из файла, указанного в cfg.ConfigDbData
func loadDbConfig(l *slog.Logger, cfg *Config) (map[string]*DatabaseInfo, error) {
	giteaAPI := CreateGiteaAPI(cfg)
	data, err := giteaAPI.GetConfigData(l, cfg.ConfigDbData)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных %s: %w", cfg.ConfigDbData, err)
	}

	var dbConfig map[string]*DatabaseInfo
	if err = yaml.Unmarshal(data, &dbConfig); err != nil {
		return nil, fmt.Errorf("ошибка парсинга %s: %w", cfg.ConfigDbData, err)
	}

	return dbConfig, nil
}

// loadMenuMainConfig загружает конфигурацию главного меню как массив строк
func loadMenuMainConfig(l *slog.Logger, cfg *Config) ([]string, error) {
	if cfg.ConfigMenuMain == "" {
		return nil, nil
	}

	giteaAPI := CreateGiteaAPI(cfg)
	data, err := giteaAPI.GetConfigData(l, cfg.ConfigMenuMain)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных menu-main: %w", err)
	}

	// Разбиваем содержимое файла на строки
	lines := strings.Split(string(data), "\n")
	return lines, nil
}

// loadMenuDebugConfig загружает конфигурацию меню отладки как массив строк
func loadMenuDebugConfig(l *slog.Logger, cfg *Config) ([]string, error) {
	if cfg.ConfigMenuDebug == "" {
		return nil, nil
	}

	giteaAPI := CreateGiteaAPI(cfg)
	data, err := giteaAPI.GetConfigData(l, cfg.ConfigMenuDebug)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных menu-debug: %w", err)
	}

	// Разбиваем содержимое файла на строки
	lines := strings.Split(string(data), "\n")
	return lines, nil
}

// getDefaultAppConfig возвращает конфигурацию приложения по умолчанию
func getDefaultAppConfig() *AppConfig {
	return &AppConfig{
		LogLevel: "Debug",
		WorkDir:  "/tmp/benadis",
		TmpDir:   "/tmp/benadis/temp",
		Timeout:  30,
		Paths: struct {
			Bin1cv8  string `yaml:"bin1cv8"`
			BinIbcmd string `yaml:"binIbcmd"`
			EdtCli   string `yaml:"edtCli"`
			Rac      string `yaml:"rac"`
		}{
			Bin1cv8:  "/opt/1cv8/x86_64/8.3.27.1606/1cv8",
			BinIbcmd: "/opt/1cv8/x86_64/8.3.27.1606/ibcmd",
			EdtCli:   "/opt/1C/1CE/components/1c-edt-2024.2.6+7-x86_64/1cedtcli",
			Rac:      "/opt/1cv8/x86_64/8.3.27.1606/rac",
		},
		Rac: struct {
			Port    int `yaml:"port"`
			Timeout int `yaml:"timeout"`
			Retries int `yaml:"retries"`
		}{
			Port:    1545,
			Timeout: 30,
			Retries: 3,
		},
		Users: struct {
			Rac        string `yaml:"rac"`
			Db         string `yaml:"db"`
			Mssql      string `yaml:"mssql"`
			StoreAdmin string `yaml:"storeAdmin"`
		}{
			Rac:        "admin",
			Db:         "db_user",
			Mssql:      "mssql_user",
			StoreAdmin: "store_admin",
		},
		SonarQube:  *GetDefaultSonarQubeConfig(),
		Scanner:    *GetDefaultScannerConfig(),
		EdtTimeout: 90 * time.Minute,
	}
}

// createWorkDirectories создает необходимые рабочие директории
func createWorkDirectories(cfg *Config) error {
	workDir := cfg.WorkDir
	tmpDir := cfg.TmpDir

	// Используем настройки из AppConfig если они доступны
	if cfg.AppConfig != nil {
		if cfg.AppConfig.WorkDir != "" {
			workDir = cfg.AppConfig.WorkDir
		}
		if cfg.AppConfig.TmpDir != "" {
			tmpDir = cfg.AppConfig.TmpDir
		}
	}

	// Создаем рабочую директорию
	if err := os.MkdirAll(workDir, 0750); err != nil {
		return fmt.Errorf("ошибка создания рабочей директории %s: %w", workDir, err)
	}

	// Создаем временную директорию
	if tmpDir == "" {
		tmpDir = workDir + "/temp"
	}
	if err := os.MkdirAll(tmpDir, 0750); err != nil {
		return fmt.Errorf("ошибка создания временной директории %s: %w", tmpDir, err)
	}

	return nil
}

// GetOneServer возвращает адрес сервера 1C:Enterprise для указанной базы данных.
// Извлекает настройки подключения к серверу 1C из конфигурации приложения
// для использования в операциях с базами данных и хранилищами конфигурации.
// Параметры:
//   - dbName: имя базы данных
//
// Возвращает:
//   - string: адрес сервера 1C:Enterprise для указанной базы данных
func (cfg *Config) GetOneServer(dbName string) string {
	if cfg.DbConfig == nil {
		return ""
	}
	if dbInfo, exists := cfg.DbConfig[dbName]; exists {
		return dbInfo.OneServer
	}
	return ""
}

// GetRacServerForDb возвращает адрес RAC сервера для указанной базы данных.
// Определяет адрес сервера администрирования кластера (RAC) для конкретной
// базы данных на основе её имени и настроек конфигурации.
// Параметры:
//   - dbName: имя базы данных
//
// Возвращает:
//   - string: адрес RAC сервера для указанной базы данных
func (cfg *Config) GetRacServerForDb(dbName string) string {
	if cfg.DbConfig == nil {
		return ""
	}
	if dbInfo, exists := cfg.DbConfig[dbName]; exists {
		// RacServer отсутствует в DatabaseInfo, используем OneServer
		return dbInfo.OneServer
	}
	return ""
}

// IsProductionDb проверяет, является ли указанная база данных продуктивной.
// Анализирует конфигурацию базы данных для определения её типа (продуктивная/тестовая),
// что влияет на применяемые политики безопасности и процедуры обновления.
// Параметры:
//   - dbName: имя базы данных для проверки
//
// Возвращает:
//   - bool: true если база данных продуктивная, false если тестовая
func (cfg *Config) IsProductionDb(dbName string) bool {
	if cfg.DbConfig == nil {
		return false
	}
	if dbInfo, exists := cfg.DbConfig[dbName]; exists {
		return dbInfo.Prod
	}
	return false
}

// GetDbServer возвращает адрес MS SQL сервера для указанной базы данных.
// Извлекает настройки подключения к серверу базы данных Microsoft SQL Server
// из конфигурации для выполнения операций с базой данных.
// Параметры:
//   - dbName: имя базы данных
//
// Возвращает:
//   - string: адрес MS SQL сервера для указанной базы данных
func (cfg *Config) GetDbServer(dbName string) string {
	if cfg.DbConfig == nil {
		return ""
	}
	if dbInfo, exists := cfg.DbConfig[dbName]; exists {
		return dbInfo.DbServer
	}
	return ""
}

// GetDatabaseInfo возвращает полную информацию о конфигурации базы данных.
// Предоставляет детальные настройки базы данных включая серверы, тип (продуктивная/тестовая)
// и другие параметры конфигурации для указанной базы данных.
// Параметры:
//   - dbName: имя базы данных
//
// Возвращает:
//   - *DatabaseInfo: указатель на структуру с информацией о базе данных или nil если не найдена
func (cfg *Config) GetDatabaseInfo(dbName string) *DatabaseInfo {
	if cfg.DbConfig == nil {
		return nil
	}
	if dbInfo, exists := cfg.DbConfig[dbName]; exists {
		return dbInfo
	}
	return nil
}

// GetSonarQubeConfig возвращает конфигурацию SonarQube.
// Если конфигурация не загружена, возвращает nil.
// Возвращает:
//   - *SonarQubeConfig: указатель на конфигурацию SonarQube
func (cfg *Config) GetSonarQubeConfig() *SonarQubeConfig {
	return cfg.SonarQubeConfig
}

// GetScannerConfig возвращает конфигурацию сканера.
// Если конфигурация не загружена, возвращает nil.
// Возвращает:
//   - *ScannerConfig: указатель на конфигурацию сканера
func (cfg *Config) GetScannerConfig() *ScannerConfig {
	return cfg.ScannerConfig
}

// SetSonarQubeConfig устанавливает конфигурацию SonarQube.
// Параметры:
//   - config: указатель на конфигурацию SonarQube
func (cfg *Config) SetSonarQubeConfig(config *SonarQubeConfig) {
	cfg.SonarQubeConfig = config
}

// SetScannerConfig устанавливает конфигурацию сканера.
// Параметры:
//   - config: указатель на конфигурацию сканера
func (cfg *Config) SetScannerConfig(config *ScannerConfig) {
	cfg.ScannerConfig = config
}

// LoadSonarQubeConfigFromEnv загружает конфигурацию SonarQube из переменных окружения.
// Возвращает:
//   - *SonarQubeConfig: указатель на загруженную конфигурацию SonarQube
//   - error: ошибка загрузки конфигурации или nil при успехе
func (cfg *Config) LoadSonarQubeConfigFromEnv() (*SonarQubeConfig, error) {
	sonarQubeConfig, err := GetSonarQubeConfig(cfg.Logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load SonarQube config: %w", err)
	}

	cfg.SetSonarQubeConfig(sonarQubeConfig)
	return sonarQubeConfig, nil
}

// LoadScannerConfigFromEnv загружает конфигурацию сканера из переменных окружения.
// Возвращает:
//   - *ScannerConfig: указатель на загруженную конфигурацию сканера
//   - error: ошибка загрузки конфигурации или nil при успехе
func (cfg *Config) LoadScannerConfigFromEnv() (*ScannerConfig, error) {
	scannerConfig, err := GetScannerConfig(cfg.Logger, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load Scanner config: %w", err)
	}

	cfg.SetScannerConfig(scannerConfig)
	return scannerConfig, nil
}

// ReloadConfig перезагружает конфигурацию из файлов и переменных окружения.
// Этот метод позволяет обновить конфигурацию во время выполнения приложения.
// Возвращает:
//   - error: ошибка перезагрузки конфигурации или nil при успехе
func (cfg *Config) ReloadConfig() error {
	// Перезагрузка конфигурации приложения
	appConfig, err := loadAppConfig(cfg.Logger, cfg)
	if err != nil {
		return fmt.Errorf("failed to reload app config: %w", err)
	}
	cfg.AppConfig = appConfig

	// Перезагрузка конфигурации проекта
	projectConfig, err := loadProjectConfig(cfg.Logger, cfg)
	if err != nil {
		cfg.Logger.Warn("ошибка перезагрузки конфигурации проекта", slog.String("error", err.Error()))
	} else {
		cfg.ProjectConfig = projectConfig
	}

	// Перезагрузка секретов
	secretConfig, err := loadSecretConfig(cfg.Logger, cfg)
	if err != nil {
		cfg.Logger.Warn("ошибка перезагрузки секретов", slog.String("error", err.Error()))
	} else {
		cfg.SecretConfig = secretConfig
	}

	// Перезагрузка конфигурации баз данных
	dbConfig, err := loadDbConfig(cfg.Logger, cfg)
	if err != nil {
		cfg.Logger.Warn("ошибка перезагрузки конфигурации БД", slog.String("error", err.Error()))
	} else {
		cfg.DbConfig = dbConfig
	}

	// Перезагрузка конфигурации главного меню
	menuMain, err := loadMenuMainConfig(cfg.Logger, cfg)
	if err != nil {
		cfg.Logger.Warn("ошибка перезагрузки конфигурации главного меню", slog.String("error", err.Error()))
	} else {
		cfg.MenuMain = menuMain
	}

	// Перезагрузка конфигурации меню отладки
	menuDebug, err := loadMenuDebugConfig(cfg.Logger, cfg)
	if err != nil {
		cfg.Logger.Warn("ошибка перезагрузки конфигурации меню отладки", slog.String("error", err.Error()))
	} else {
		cfg.MenuDebug = menuDebug
	}

	// Перезагрузка конфигурации SonarQube из переменных окружения
	sonarQubeConfig, err := GetSonarQubeConfig(cfg.Logger, cfg)
	if err != nil {
		cfg.Logger.Warn("ошибка перезагрузки конфигурации SonarQube", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.SonarQubeConfig = GetDefaultSonarQubeConfig()
	} else {
		cfg.SonarQubeConfig = sonarQubeConfig
	}

	// Перезагрузка конфигурации сканера из переменных окружения
	scannerConfig, err := GetScannerConfig(cfg.Logger, cfg)
	if err != nil {
		cfg.Logger.Warn("ошибка перезагрузки конфигурации сканера", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.ScannerConfig = GetDefaultScannerConfig()
	} else {
		cfg.ScannerConfig = scannerConfig
	}

	return nil
}

// GetAllDatabases возвращает список имен всех доступных баз данных.
// Извлекает полный перечень баз данных из конфигурации для использования
// в операциях массового управления и мониторинга.
// Возвращает:
//   - []string: срез с именами всех настроенных баз данных
func (cfg *Config) GetAllDatabases() []string {
	if cfg.DbConfig == nil {
		return nil
	}
	databases := make([]string, 0, len(cfg.DbConfig))
	for dbName := range cfg.DbConfig {
		databases = append(databases, dbName)
	}
	return databases
}

// GetProductionDatabases возвращает список имен всех продуктивных баз данных.
// Фильтрует базы данных по типу, возвращая только продуктивные базы
// для применения специальных политик безопасности и процедур обновления.
// Возвращает:
//   - []string: срез с именами продуктивных баз данных
func (cfg *Config) GetProductionDatabases() []string {
	if cfg.DbConfig == nil {
		return nil
	}
	var prodDatabases []string
	for dbName, dbInfo := range cfg.DbConfig {
		if dbInfo.Prod {
			prodDatabases = append(prodDatabases, dbName)
		}
	}
	return prodDatabases
}

// GetTestDatabases возвращает список имен всех тестовых баз данных.
// Фильтрует базы данных по типу, возвращая только тестовые базы
// для выполнения операций разработки и тестирования.
// Возвращает:
//   - []string: срез с именами тестовых баз данных
func (cfg *Config) GetTestDatabases() []string {
	if cfg.DbConfig == nil {
		return nil
	}
	var testDatabases []string
	for dbName, dbInfo := range cfg.DbConfig {
		if !dbInfo.Prod {
			testDatabases = append(testDatabases, dbName)
		}
	}
	return testDatabases
}

// FindRelatedDatabase находит связанную базу данных для указанной базы данных.
// Анализирует конфигурацию для поиска парной базы данных (продуктивная/тестовая)
// или связанной базы данных в рамках одного проекта или кластера.
// Параметры:
//   - dbName: имя базы данных для поиска связанной
//
// Возвращает:
//   - string: имя связанной базы данных
//   - error: ошибка поиска или nil при успехе
func (cfg *Config) FindRelatedDatabase(dbName string) (string, error) {
	if cfg.ProjectConfig == nil {
		return "", fmt.Errorf("project config is not loaded")
	}

	// Проверяем, является ли dbName продуктивной базой
	for prodDB, prodInfo := range cfg.ProjectConfig.Prod {
		if prodDB == dbName {
			// Ищем первую связанную базу
			for relatedDB := range prodInfo.Related {
				return relatedDB, nil
			}
			return "", fmt.Errorf("no related database found for production database %s", dbName)
		}
	}

	// Проверяем, является ли dbName связанной базой
	for prodDB, prodInfo := range cfg.ProjectConfig.Prod {
		for relatedDB := range prodInfo.Related {
			if relatedDB == dbName {
				return prodDB, nil
			}
		}
	}

	return "", fmt.Errorf("database %s is not found in project configuration", dbName)
}

// GetDatabaseServer возвращает адрес сервера базы данных для указанной БД.
// Извлекает настройки сервера из конфигурации и возвращает полный адрес
// сервера базы данных для установления подключения.
// Параметры:
//   - dbName: имя базы данных
//
// Возвращает:
//   - string: адрес сервера базы данных
//   - error: ошибка получения адреса сервера или nil при успехе
func (cfg *Config) GetDatabaseServer(dbName string) (string, error) {
	if cfg.DbConfig == nil {
		return "", fmt.Errorf("database config is not loaded")
	}

	dbInfo, exists := cfg.DbConfig[dbName]
	if !exists {
		return "", fmt.Errorf("database %s not found in database configuration", dbName)
	}

	if dbInfo.DbServer == "" {
		return "", fmt.Errorf("database server not specified for database %s", dbName)
	}

	return dbInfo.DbServer, nil
}

// DetermineSrcAndDstServers определяет исходный и целевой серверы для операций с базой данных.
// Анализирует конфигурацию и определяет серверы-источник и назначение для операций
// копирования, синхронизации или миграции данных между базами данных.
// Параметры:
//   - dbName: имя базы данных
//
// Возвращает:
//   - srcServer: адрес исходного сервера
//   - dstServer: адрес целевого сервера
//   - srcDB: имя исходной базы данных
//   - dstDB: имя целевой базы данных
//   - err: ошибка определения серверов или nil при успехе
func (cfg *Config) DetermineSrcAndDstServers(dbName string) (srcServer, dstServer, srcDB, dstDB string, err error) {
	// Находим связанную базу
	relatedDB, err := cfg.FindRelatedDatabase(dbName)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to find related database: %w", err)
	}

	// Определяем, какая база продуктивная, а какая тестовая
	var prodDB, testDB string
	if cfg.IsProductionDb(dbName) {
		prodDB = dbName
		testDB = relatedDB
	} else {
		prodDB = relatedDB
		testDB = dbName
	}

	// Получаем серверы для обеих баз
	prodServer, err := cfg.GetDatabaseServer(prodDB)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get server for production database %s: %w", prodDB, err)
	}

	testServer, err := cfg.GetDatabaseServer(testDB)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to get server for test database %s: %w", testDB, err)
	}

	// srcServer - продуктивный сервер, dstServer - тестовый сервер
	return prodServer, testServer, prodDB, testDB, nil
}

// FileInfo теперь используется из модуля gitea

// AnalyzeProject анализирует проект и заполняет поля ProjectName и AddArray в конфигурации.
// Выполняет анализ структуры проекта и его конфигурации, проверяет целостность настроек проекта,
// валидирует конфигурацию баз данных и анализирует зависимости между компонентами системы.
// Параметры:
//   - l: логгер для записи информации о процессе анализа
//   - branch: имя ветки для анализа проекта
//
// Возвращает:
//   - error: ошибка анализа проекта или nil при успехе
func (cfg *Config) AnalyzeProject(l *slog.Logger, branch string) error {
	// Инициализируем Gitea API
	g := CreateGiteaAPI(cfg)

	// Анализируем проект
	analysis, err := g.AnalyzeProject(branch)
	if err != nil {
		l.Error("Ошибка анализа проекта",
			slog.String("error", err.Error()),
		)
		return err
	}

	// Заполняем поля конфигурации
	if len(analysis) == 0 {
		l.Info("Проект не найден или не соответствует критериям")
		cfg.ProjectName = ""
		cfg.AddArray = []string{}
	} else {
		cfg.ProjectName = analysis[0]
		cfg.AddArray = analysis[1:]
		l.Info("Результат анализа проекта",
			slog.String("project_name", cfg.ProjectName),
			slog.Any("extensions", cfg.AddArray),
		)
	}

	return nil
}
