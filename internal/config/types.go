package config

import (
	"log/slog"
	"time"
)

// Все константы перенесены в пакет constants
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

	Logging         LoggingConfig         `yaml:"logging"`
	Implementations ImplementationsConfig `yaml:"implementations"`
	Alerting        AlertingConfig        `yaml:"alerting"`
	Metrics         MetricsConfig         `yaml:"metrics"`
	Tracing         TracingConfig         `yaml:"tracing"`
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
// GetServer возвращает сервер для подключения к базе данных.
// LOW-2 fix (Review #4): централизованная логика fallback OneServer → DbServer.
// Приоритет: OneServer (если указан), иначе DbServer.
func (d *DatabaseInfo) GetServer() string {
	if d.OneServer != "" {
		return d.OneServer
	}
	return d.DbServer
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

	// Implementations настройки (выбор реализаций операций)
	ImplementationsConfig *ImplementationsConfig

	// RAC настройки
	RacConfig *RacConfig

	// Alerting настройки
	AlertingConfig *AlertingConfig

	// Metrics настройки
	MetricsConfig *MetricsConfig

	// Tracing настройки (OpenTelemetry)
	TracingConfig *TracingConfig

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
	PRNumber      int64 `env:"BR_PR_NUMBER" env-default:"0"`

	// Устаревшие поля (сохранены для обратной совместимости)
	// TODO: Удалить после полной миграции
	Connect     string `env:"Connect_String" env-default:""`
	RepPath     string `env:"RepPath" env-default:"/tmp/4del/rep"`
	WorkDir     string `env:"WorkDir" env-default:"/tmp/4del"`
	TmpDir      string `env:"TmpDir" env-default:"/tmp"`
	AccessToken string `env:"BR_ACCESS_TOKEN" env-default:""`
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
// Repo представляет информацию о репозитории.
// Содержит настройки репозитория, включая ветку по умолчанию.
type Repo struct {
	DefaultBranch string `json:"default_branch"`
}
