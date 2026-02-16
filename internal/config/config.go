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
	"github.com/Kargones/apk-ci/internal/pkg/alerting"
	"github.com/Kargones/apk-ci/internal/pkg/urlutil"
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

// ImplementationsConfig содержит настройки выбора реализаций операций.
// Позволяет переключаться между различными инструментами (1cv8/ibcmd/native)
// без изменения кода приложения.
type ImplementationsConfig struct {
	// ConfigExport определяет инструмент для выгрузки конфигурации.
	// Допустимые значения: "1cv8" (default), "ibcmd", "native"
	ConfigExport string `yaml:"config_export" env:"BR_IMPL_CONFIG_EXPORT" env-default:"1cv8"`

	// DBCreate определяет инструмент для создания базы данных.
	// Допустимые значения: "1cv8" (default), "ibcmd"
	DBCreate string `yaml:"db_create" env:"BR_IMPL_DB_CREATE" env-default:"1cv8"`
}

// Validate проверяет корректность значений ImplementationsConfig.
// Возвращает ошибку если значения не соответствуют допустимым.
func (c *ImplementationsConfig) Validate() error {
	// Применяем defaults для пустых значений
	if c.ConfigExport == "" {
		c.ConfigExport = "1cv8"
	}
	if c.DBCreate == "" {
		c.DBCreate = "1cv8"
	}

	validConfigExport := map[string]bool{"1cv8": true, "ibcmd": true, "native": true}
	validDBCreate := map[string]bool{"1cv8": true, "ibcmd": true}

	if !validConfigExport[c.ConfigExport] {
		return fmt.Errorf("недопустимое значение ConfigExport: %q, допустимые: 1cv8, ibcmd, native", c.ConfigExport)
	}
	if !validDBCreate[c.DBCreate] {
		return fmt.Errorf("недопустимое значение DBCreate: %q, допустимые: 1cv8, ibcmd", c.DBCreate)
	}
	return nil
}

// LoggingConfig содержит настройки для логирования.
//
// TODO (M-10/Review #13): Dual source of truth — LoggingConfig здесь и logging.Config
// в internal/pkg/logging/config.go дублируют поля. При добавлении новых опций нужно
// менять оба места и синхронизировать defaults. Рефакторинг: использовать одну структуру.
type LoggingConfig struct {
	// Level - уровень логирования (debug, info, warn, error)
	Level string `yaml:"level" env:"BR_LOG_LEVEL" env-default:"info"`

	// Format - формат логов (json, text)
	Format string `yaml:"format" env:"BR_LOG_FORMAT" env-default:"text"`

	// Output - вывод логов (stdout, stderr, file)
	Output string `yaml:"output" env:"BR_LOG_OUTPUT" env-default:"stderr"`

	// FilePath - путь к файлу логов (если output=file)
	FilePath string `yaml:"filePath" env:"BR_LOG_FILE_PATH"`

	// MaxSize - максимальный размер файла лога в MB
	MaxSize int `yaml:"maxSize" env:"BR_LOG_MAX_SIZE" env-default:"100"`

	// MaxBackups - максимальное количество backup файлов
	MaxBackups int `yaml:"maxBackups" env:"BR_LOG_MAX_BACKUPS" env-default:"3"`

	// MaxAge - максимальный возраст backup файлов в днях
	MaxAge int `yaml:"maxAge" env:"BR_LOG_MAX_AGE" env-default:"7"`

	// Compress - сжимать ли backup файлы
	// TODO (M-4/Review #9): bool с env-default:"true" — в YAML yaml:"compress" false будет
	// перезаписан env-default при cleanenv.ReadEnv. Поведение корректно только при чтении
	// из env. Для YAML-source используется getDefaultLoggingConfig() где Compress=true.
	Compress bool `yaml:"compress" env:"BR_LOG_COMPRESS" env-default:"true"`
}

// AlertingConfig содержит настройки для алертинга.
type AlertingConfig struct {
	// Enabled — включён ли алертинг (по умолчанию false).
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_ENABLED" env-default:"false"`

	// RateLimitWindow — минимальный интервал между алертами одного типа.
	RateLimitWindow time.Duration `yaml:"rateLimitWindow" env:"BR_ALERTING_RATE_LIMIT_WINDOW" env-default:"5m"`

	// Email — конфигурация email канала.
	Email EmailChannelConfig `yaml:"email"`

	// Telegram — конфигурация telegram канала.
	Telegram TelegramChannelConfig `yaml:"telegram"`

	// Webhook — конфигурация webhook канала.
	Webhook WebhookChannelConfig `yaml:"webhook"`

	// Rules — правила фильтрации алертов.
	Rules AlertRulesConfig `yaml:"rules"`
}

// AlertRulesConfig содержит настройки правил фильтрации алертов.
type AlertRulesConfig struct {
	// MinSeverity — минимальный уровень severity для отправки алерта.
	// Значения: "INFO", "WARNING", "CRITICAL". По умолчанию: "INFO" (все алерты).
	MinSeverity string `yaml:"minSeverity" env:"BR_ALERTING_RULES_MIN_SEVERITY" env-default:"INFO"`

	// ExcludeErrorCodes — коды ошибок, для которых НЕ отправляются алерты.
	ExcludeErrorCodes []string `yaml:"excludeErrorCodes" env:"BR_ALERTING_RULES_EXCLUDE_ERRORS" env-separator:","`

	// IncludeErrorCodes — если задан, алерты отправляются ТОЛЬКО для этих кодов.
	// Имеет приоритет над ExcludeErrorCodes.
	IncludeErrorCodes []string `yaml:"includeErrorCodes" env:"BR_ALERTING_RULES_INCLUDE_ERRORS" env-separator:","`

	// ExcludeCommands — команды, для которых НЕ отправляются алерты.
	ExcludeCommands []string `yaml:"excludeCommands" env:"BR_ALERTING_RULES_EXCLUDE_COMMANDS" env-separator:","`

	// IncludeCommands — если задан, алерты отправляются ТОЛЬКО для этих команд.
	// Имеет приоритет над ExcludeCommands.
	IncludeCommands []string `yaml:"includeCommands" env:"BR_ALERTING_RULES_INCLUDE_COMMANDS" env-separator:","`

	// ChannelOverrides — правила для конкретных каналов.
	// ВНИМАНИЕ: channel override ПОЛНОСТЬЮ ЗАМЕНЯЕТ глобальные правила для канала,
	// а НЕ мержит с ними. Если указан override с excludeErrorCodes но без minSeverity,
	// будет использован minSeverity=INFO (default), а не глобальный minSeverity.
	//
	// Пример НЕПРАВИЛЬНОЙ конфигурации (email получит ВСЕ severity, не только CRITICAL):
	//   rules:
	//     minSeverity: "CRITICAL"
	//     channels:
	//       email:
	//         excludeErrorCodes: ["ERR_SPAM"]
	//
	// Пример ПРАВИЛЬНОЙ конфигурации (повторяем minSeverity в override):
	//   rules:
	//     minSeverity: "CRITICAL"
	//     channels:
	//       email:
	//         minSeverity: "CRITICAL"
	//         excludeErrorCodes: ["ERR_SPAM"]
	ChannelOverrides map[string]ChannelRuleConfig `yaml:"channels"`
}

// ChannelRuleConfig — правила для конкретного канала алертинга.
type ChannelRuleConfig struct {
	MinSeverity       string   `yaml:"minSeverity"`
	ExcludeErrorCodes []string `yaml:"excludeErrorCodes"`
	IncludeErrorCodes []string `yaml:"includeErrorCodes"`
	ExcludeCommands   []string `yaml:"excludeCommands"`
	IncludeCommands   []string `yaml:"includeCommands"`
}

// EmailChannelConfig содержит настройки email канала.
type EmailChannelConfig struct {
	// Enabled — включён ли email канал.
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_EMAIL_ENABLED" env-default:"false"`

	// SMTPHost — адрес SMTP сервера.
	SMTPHost string `yaml:"smtpHost" env:"BR_ALERTING_SMTP_HOST"`

	// SMTPPort — порт SMTP сервера (25, 465, 587).
	SMTPPort int `yaml:"smtpPort" env:"BR_ALERTING_SMTP_PORT" env-default:"587"`

	// SMTPUser — пользователь для SMTP авторизации.
	SMTPUser string `yaml:"smtpUser" env:"BR_ALERTING_SMTP_USER"`

	// SMTPPassword — пароль для SMTP авторизации.
	SMTPPassword string `yaml:"smtpPassword" env:"BR_ALERTING_SMTP_PASSWORD"`

	// UseTLS — использовать TLS (StartTLS для 587, implicit для 465).
	// TODO (M-4/Review #9): bool с env-default:"true" — аналогично Compress,
	// YAML false может быть перезаписан cleanenv.ReadEnv. Для YAML-source
	// используется getDefaultAlertingConfig() где UseTLS=true.
	UseTLS bool `yaml:"useTLS" env:"BR_ALERTING_SMTP_TLS" env-default:"true"`

	// From — адрес отправителя.
	From string `yaml:"from" env:"BR_ALERTING_EMAIL_FROM"`

	// To — список получателей (comma-separated в env).
	To []string `yaml:"to" env:"BR_ALERTING_EMAIL_TO" env-separator:","`

	// SubjectTemplate — шаблон темы письма.
	// Placeholders: {{.ErrorCode}}, {{.Command}}, {{.Infobase}}
	SubjectTemplate string `yaml:"subjectTemplate" env:"BR_ALERTING_EMAIL_SUBJECT" env-default:"[apk-ci] {{.ErrorCode}}: {{.Command}}"`

	// Timeout — таймаут SMTP операций.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_SMTP_TIMEOUT" env-default:"30s"`
}

// TelegramChannelConfig содержит настройки telegram канала.
type TelegramChannelConfig struct {
	// Enabled — включён ли telegram канал.
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_TELEGRAM_ENABLED" env-default:"false"`

	// BotToken — токен Telegram бота (получить у @BotFather).
	BotToken string `yaml:"botToken" env:"BR_ALERTING_TELEGRAM_BOT_TOKEN"`

	// ChatIDs — список идентификаторов чатов/групп для отправки.
	// Может быть числовой ID или @username для публичных каналов.
	ChatIDs []string `yaml:"chatIds" env:"BR_ALERTING_TELEGRAM_CHAT_IDS" env-separator:","`

	// Timeout — таймаут HTTP запросов к Telegram API.
	// По умолчанию: 10 секунд.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_TELEGRAM_TIMEOUT" env-default:"10s"`
}

// WebhookChannelConfig содержит настройки webhook канала.
type WebhookChannelConfig struct {
	// Enabled — включён ли webhook канал.
	Enabled bool `yaml:"enabled" env:"BR_ALERTING_WEBHOOK_ENABLED" env-default:"false"`

	// URLs — список URL для отправки webhook.
	// Алерт отправляется на все указанные URL.
	URLs []string `yaml:"urls" env:"BR_ALERTING_WEBHOOK_URLS" env-separator:","`

	// Headers — дополнительные HTTP заголовки.
	// Используется для Authorization, X-Api-Key и т.д.
	// TODO (M-3/Review Epic-6): Headers доступны только через YAML, не через env.
	// cleanenv не поддерживает map[string]string из env переменных.
	// Для CI/CD использовать YAML или добавить парсинг "Key=Val,Key2=Val2" из env.
	Headers map[string]string `yaml:"headers"`

	// Timeout — таймаут HTTP запросов.
	// По умолчанию: 10 секунд.
	Timeout time.Duration `yaml:"timeout" env:"BR_ALERTING_WEBHOOK_TIMEOUT" env-default:"10s"`

	// MaxRetries — максимальное количество повторных попыток.
	// По умолчанию: 3.
	MaxRetries int `yaml:"maxRetries" env:"BR_ALERTING_WEBHOOK_MAX_RETRIES" env-default:"3"`
}

// MetricsConfig содержит настройки для Prometheus метрик.
type MetricsConfig struct {
	// Enabled — включены ли метрики (по умолчанию false).
	Enabled bool `yaml:"enabled" env:"BR_METRICS_ENABLED" env-default:"false"`

	// PushgatewayURL — URL Prometheus Pushgateway.
	// Пример: "http://pushgateway:9091"
	PushgatewayURL string `yaml:"pushgatewayUrl" env:"BR_METRICS_PUSHGATEWAY_URL"`

	// JobName — имя job для группировки метрик.
	// По умолчанию: "apk-ci"
	JobName string `yaml:"jobName" env:"BR_METRICS_JOB_NAME" env-default:"apk-ci"`

	// Timeout — таймаут HTTP запросов к Pushgateway.
	// По умолчанию: 10 секунд.
	Timeout time.Duration `yaml:"timeout" env:"BR_METRICS_TIMEOUT" env-default:"10s"`

	// InstanceLabel — переопределение instance label.
	// Если пусто — используется hostname.
	InstanceLabel string `yaml:"instanceLabel" env:"BR_METRICS_INSTANCE"`
}

// TracingConfig содержит настройки OpenTelemetry трейсинга.
type TracingConfig struct {
	// Enabled включает отправку трейсов в OTLP бэкенд.
	Enabled bool `yaml:"enabled" env:"BR_TRACING_ENABLED" env-default:"false"`

	// Endpoint — URL OTLP HTTP endpoint (например, http://jaeger:4318).
	Endpoint string `yaml:"endpoint" env:"BR_TRACING_ENDPOINT"`

	// ServiceName — имя сервиса для resource attributes.
	ServiceName string `yaml:"serviceName" env:"BR_TRACING_SERVICE_NAME" env-default:"apk-ci"`

	// Environment — окружение (production, staging, development).
	Environment string `yaml:"environment" env:"BR_TRACING_ENVIRONMENT" env-default:"production"`

	// Insecure — использовать HTTP вместо HTTPS для OTLP endpoint.
	// L-11/Review #15: По умолчанию true (HTTP) для совместимости с внутренними сетями.
	// Для production deployment через публичные сети установить false (HTTPS).
	Insecure bool `yaml:"insecure" env:"BR_TRACING_INSECURE" env-default:"true"`

	// Timeout — таймаут для экспорта трейсов.
	Timeout time.Duration `yaml:"timeout" env:"BR_TRACING_TIMEOUT" env-default:"5s"`

	// SamplingRate — доля сэмплируемых трейсов (0.0 — ни один, 1.0 — все).
	SamplingRate float64 `yaml:"samplingRate" env:"BR_TRACING_SAMPLING_RATE" env-default:"1.0"`
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

// loadImplementationsConfig загружает конфигурацию реализаций из AppConfig, переменных окружения или устанавливает значения по умолчанию
func loadImplementationsConfig(l *slog.Logger, cfg *Config) (*ImplementationsConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && (cfg.AppConfig.Implementations != ImplementationsConfig{}) {
		implConfig := &cfg.AppConfig.Implementations
		// Применяем переопределения из переменных окружения
		if err := cleanenv.ReadEnv(implConfig); err != nil {
			l.Warn("Ошибка загрузки Implementations конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Implementations конфигурация загружена из AppConfig",
			slog.String("config_export", implConfig.ConfigExport),
			slog.String("db_create", implConfig.DBCreate),
		)
		return implConfig, nil
	}

	// Если конфигурация не найдена, используем значения по умолчанию
	implConfig := getDefaultImplementationsConfig()

	// Применяем переопределения из переменных окружения
	if err := cleanenv.ReadEnv(implConfig); err != nil {
		l.Warn("Ошибка загрузки Implementations конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Implementations конфигурация: используются значения по умолчанию",
		slog.String("config_export", implConfig.ConfigExport),
		slog.String("db_create", implConfig.DBCreate),
	)

	return implConfig, nil
}

// loadLoggingConfig загружает конфигурацию логирования из AppConfig, переменных окружения или устанавливает значения по умолчанию.
// Переменные окружения BR_LOG_* переопределяют значения из AppConfig (AC4).
func loadLoggingConfig(l *slog.Logger, cfg *Config) (*LoggingConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && (cfg.AppConfig.Logging != LoggingConfig{}) {
		loggingConfig := &cfg.AppConfig.Logging
		// Применяем env override для AppConfig (симметрично с loadImplementationsConfig)
		if err := cleanenv.ReadEnv(loggingConfig); err != nil {
			l.Warn("Ошибка загрузки Logging конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Logging конфигурация загружена из AppConfig",
			slog.String("level", loggingConfig.Level),
			slog.String("format", loggingConfig.Format),
		)
		return loggingConfig, nil
	}

	loggingConfig := getDefaultLoggingConfig()

	if err := cleanenv.ReadEnv(loggingConfig); err != nil {
		l.Warn("Ошибка загрузки Logging конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Logging конфигурация: используются значения по умолчанию",
		slog.String("level", loggingConfig.Level),
		slog.String("format", loggingConfig.Format),
	)

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
		UserName:          "apk-ci",
		UserEmail:         "runner@benadis.ru",
		DefaultBranch:     "main",
		Timeout:           60 * time.Minute,
		CredentialHelper:  "store",
		CredentialTimeout: 24 * time.Hour,
	}
}

// getDefaultImplementationsConfig возвращает конфигурацию реализаций по умолчанию
func getDefaultImplementationsConfig() *ImplementationsConfig {
	return &ImplementationsConfig{
		ConfigExport: "1cv8",
		DBCreate:     "1cv8",
	}
}

// getDefaultLoggingConfig возвращает конфигурацию логирования по умолчанию.
// ВАЖНО: Значения ДОЛЖНЫ совпадать с константами logging.DefaultXxx из
// internal/pkg/logging/config.go — единственный источник истины для defaults.
func getDefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		Level:      "info",                         // logging.DefaultLevel
		Format:     "text",                         // logging.DefaultFormat
		Output:     "stderr",                       // logging.DefaultOutput
		FilePath:   "/var/log/apk-ci.log",  // logging.DefaultFilePath
		MaxSize:    100,                            // logging.DefaultMaxSize
		MaxBackups: 3,                              // logging.DefaultMaxBackups
		MaxAge:     7,                              // logging.DefaultMaxAge
		Compress:   true,                           // logging.DefaultCompress
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

// isTracingConfigPresent проверяет, задана ли конфигурация трейсинга.
// Возвращает true если хотя бы одно значимое поле отличается от zero value.
func isTracingConfigPresent(cfg *TracingConfig) bool {
	if cfg == nil {
		return false
	}
	return cfg.Enabled || cfg.Endpoint != ""
}

// getDefaultTracingConfig возвращает конфигурацию трейсинга по умолчанию.
// Трейсинг отключён по умолчанию (AC5).
func getDefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:      false,
		Endpoint:     "",
		ServiceName:  "apk-ci",
		Environment:  "production",
		Insecure:     true,
		Timeout:      5 * time.Second,
		SamplingRate: 1.0,
	}
}

// validateTracingConfig проверяет корректность конфигурации трейсинга при загрузке.
// Проверяет обязательные поля при включённом трейсинге.
func validateTracingConfig(tc *TracingConfig) error {
	if !tc.Enabled {
		return nil
	}
	if tc.Endpoint == "" {
		return fmt.Errorf("tracing: endpoint обязателен при enabled=true")
	}
	if tc.ServiceName == "" {
		return fmt.Errorf("tracing: service name обязателен при enabled=true")
	}
	if tc.Timeout <= 0 {
		return fmt.Errorf("tracing: timeout должен быть положительным")
	}
	if tc.SamplingRate < 0.0 || tc.SamplingRate > 1.0 {
		// L-12/Review #15: %g вместо %f для читаемого вывода.
		return fmt.Errorf("tracing: sampling rate должен быть от 0.0 до 1.0, получено: %g", tc.SamplingRate)
	}
	return nil
}

// loadTracingConfig загружает конфигурацию трейсинга из AppConfig, переменных окружения или устанавливает значения по умолчанию.
// Переменные окружения BR_TRACING_* переопределяют значения из AppConfig.
func loadTracingConfig(l *slog.Logger, cfg *Config) (*TracingConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && isTracingConfigPresent(&cfg.AppConfig.Tracing) {
		tracingConfig := &cfg.AppConfig.Tracing
		// Применяем env override для AppConfig
		if err := cleanenv.ReadEnv(tracingConfig); err != nil {
			l.Warn("Ошибка загрузки Tracing конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Debug("Tracing конфигурация загружена из AppConfig",
			slog.Bool("enabled", tracingConfig.Enabled),
			slog.String("endpoint", tracingConfig.Endpoint),
			slog.String("service_name", tracingConfig.ServiceName),
		)
		return tracingConfig, nil
	}

	tracingConfig := getDefaultTracingConfig()

	if err := cleanenv.ReadEnv(tracingConfig); err != nil {
		l.Warn("Ошибка загрузки Tracing конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Tracing конфигурация: используются значения по умолчанию",
		slog.Bool("enabled", tracingConfig.Enabled),
	)

	return tracingConfig, nil
}

// isAlertingConfigPresent проверяет, задана ли конфигурация алертинга.
// Возвращает true если хотя бы одно значимое поле отличается от zero value.
func isAlertingConfigPresent(cfg *AlertingConfig) bool {
	if cfg == nil {
		return false
	}
	// Проверяем любое значимое поле (enabled, или настройки email/telegram/webhook)
	return cfg.Enabled ||
		cfg.Email.Enabled || cfg.Email.SMTPHost != "" ||
		cfg.Telegram.Enabled || cfg.Telegram.BotToken != "" ||
		cfg.Webhook.Enabled || len(cfg.Webhook.URLs) > 0
}

// getDefaultAlertingConfig возвращает конфигурацию алертинга по умолчанию.
// Алертинг отключён по умолчанию (AC6).
// M-1/Review #9: используем константы из пакета alerting вместо magic numbers.
func getDefaultAlertingConfig() *AlertingConfig {
	return &AlertingConfig{
		Enabled:         false,
		RateLimitWindow: alerting.DefaultRateLimitWindow,
		Email: EmailChannelConfig{
			Enabled:         false,
			SMTPPort:        alerting.DefaultSMTPPort,
			UseTLS:          true,
			SubjectTemplate: alerting.DefaultSubjectTemplate,
			Timeout:         alerting.DefaultSMTPTimeout,
		},
		Telegram: TelegramChannelConfig{
			Enabled: false,
			Timeout: alerting.DefaultTelegramTimeout,
		},
		Webhook: WebhookChannelConfig{
			Enabled:    false,
			Timeout:    alerting.DefaultWebhookTimeout,
			MaxRetries: alerting.DefaultMaxRetries,
		},
		Rules: AlertRulesConfig{
			MinSeverity: "INFO",
		},
	}
}

// loadAlertingConfig загружает конфигурацию алертинга из AppConfig, переменных окружения или устанавливает значения по умолчанию.
// Переменные окружения BR_ALERTING_* переопределяют значения из AppConfig (AC5).
func loadAlertingConfig(l *slog.Logger, cfg *Config) (*AlertingConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	// Примечание: нельзя сравнивать struct напрямую из-за slice в EmailChannelConfig
	if cfg.AppConfig != nil && isAlertingConfigPresent(&cfg.AppConfig.Alerting) {
		alertingConfig := &cfg.AppConfig.Alerting
		// Применяем env override для AppConfig (симметрично с loadLoggingConfig)
		if err := cleanenv.ReadEnv(alertingConfig); err != nil {
			l.Warn("Ошибка загрузки Alerting конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Alerting конфигурация загружена из AppConfig",
			slog.Bool("enabled", alertingConfig.Enabled),
			slog.Bool("email_enabled", alertingConfig.Email.Enabled),
			slog.Bool("telegram_enabled", alertingConfig.Telegram.Enabled), // L1 fix
		)
		return alertingConfig, nil
	}

	alertingConfig := getDefaultAlertingConfig()

	if err := cleanenv.ReadEnv(alertingConfig); err != nil {
		l.Warn("Ошибка загрузки Alerting конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Alerting конфигурация: используются значения по умолчанию",
		slog.Bool("enabled", alertingConfig.Enabled),
	)

	return alertingConfig, nil
}

// isMetricsConfigPresent проверяет, задана ли конфигурация метрик.
// Возвращает true если хотя бы одно значимое поле отличается от zero value.
func isMetricsConfigPresent(cfg *MetricsConfig) bool {
	if cfg == nil {
		return false
	}
	// Проверяем любое значимое поле (enabled, или pushgateway URL)
	return cfg.Enabled || cfg.PushgatewayURL != ""
}

// getDefaultMetricsConfig возвращает конфигурацию метрик по умолчанию.
// Метрики отключены по умолчанию (AC6).
func getDefaultMetricsConfig() *MetricsConfig {
	return &MetricsConfig{
		Enabled:        false,
		PushgatewayURL: "",
		JobName:        "apk-ci",
		Timeout:        10 * time.Second,
		InstanceLabel:  "",
	}
}

// loadMetricsConfig загружает конфигурацию метрик из AppConfig, переменных окружения или устанавливает значения по умолчанию.
// Переменные окружения BR_METRICS_* переопределяют значения из AppConfig.
func loadMetricsConfig(l *slog.Logger, cfg *Config) (*MetricsConfig, error) {
	// Проверяем, есть ли конфигурация в AppConfig
	if cfg.AppConfig != nil && isMetricsConfigPresent(&cfg.AppConfig.Metrics) {
		metricsConfig := &cfg.AppConfig.Metrics
		// Применяем env override для AppConfig (симметрично с loadAlertingConfig)
		if err := cleanenv.ReadEnv(metricsConfig); err != nil {
			l.Warn("Ошибка загрузки Metrics конфигурации из переменных окружения",
				slog.String("error", err.Error()),
			)
		}
		l.Info("Metrics конфигурация загружена из AppConfig",
			slog.Bool("enabled", metricsConfig.Enabled),
			slog.String("pushgateway_url", urlutil.MaskURL(metricsConfig.PushgatewayURL)),
			slog.String("job_name", metricsConfig.JobName),
		)
		return metricsConfig, nil
	}

	metricsConfig := getDefaultMetricsConfig()

	if err := cleanenv.ReadEnv(metricsConfig); err != nil {
		l.Warn("Ошибка загрузки Metrics конфигурации из переменных окружения",
			slog.String("error", err.Error()),
		)
	}

	l.Debug("Metrics конфигурация: используются значения по умолчанию",
		slog.Bool("enabled", metricsConfig.Enabled),
	)

	return metricsConfig, nil
}

// validateAlertingConfig проверяет корректность конфигурации алертинга при загрузке.
// Проверяет обязательные поля для каждого включённого канала.
//
// M-2/Review #9: Это предварительная (config-level) валидация — проверяет только наличие
// обязательных полей при загрузке конфигурации. Полная валидация (формат URL, CRLF injection
// в email адресах, Header Injection в webhook headers) выполняется в alerting.Config.Validate()
// при создании Alerter через providers.go. Defense-in-depth: fail-fast при явно невалидной конфигурации.
func validateAlertingConfig(ac *AlertingConfig) error {
	if !ac.Enabled {
		return nil
	}
	if ac.Email.Enabled {
		if ac.Email.SMTPHost == "" {
			return fmt.Errorf("alerting.email: SMTP host обязателен")
		}
		if ac.Email.From == "" {
			return fmt.Errorf("alerting.email: адрес отправителя (from) обязателен")
		}
		if len(ac.Email.To) == 0 {
			return fmt.Errorf("alerting.email: хотя бы один получатель (to) обязателен")
		}
	}
	if ac.Telegram.Enabled {
		if ac.Telegram.BotToken == "" {
			return fmt.Errorf("alerting.telegram: bot_token обязателен")
		}
		if len(ac.Telegram.ChatIDs) == 0 {
			return fmt.Errorf("alerting.telegram: хотя бы один chat_id обязателен")
		}
	}
	if ac.Webhook.Enabled && len(ac.Webhook.URLs) == 0 {
		return fmt.Errorf("alerting.webhook: хотя бы один URL обязателен")
	}
	return nil
}

// validateMetricsConfig проверяет корректность конфигурации метрик при загрузке.
// Проверяет обязательные поля при включённых метриках.
func validateMetricsConfig(mc *MetricsConfig) error {
	if !mc.Enabled {
		return nil
	}
	if mc.PushgatewayURL == "" {
		return fmt.Errorf("metrics: pushgateway_url обязателен при enabled=true")
	}
	if mc.Timeout <= 0 {
		return fmt.Errorf("metrics: timeout должен быть положительным")
	}
	return nil
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
	// TODO (M-2/Review #17): SonarQubeConfig.Validate() и ScannerConfig.Validate()
	// существуют, но не вызываются в MustLoad(). Добавить fail-fast валидацию
	// по аналогии с AlertingConfig/MetricsConfig/TracingConfig.
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

	// Загрузка конфигурации реализаций
	if cfg.ImplementationsConfig, err = loadImplementationsConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации реализаций", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.ImplementationsConfig = getDefaultImplementationsConfig()
	}

	// Валидация конфигурации реализаций (AC6: невалидные значения обнаруживаются early)
	if cfg.ImplementationsConfig != nil {
		if err = cfg.ImplementationsConfig.Validate(); err != nil {
			l.Warn("невалидная конфигурация реализаций, используются значения по умолчанию",
				slog.String("error", err.Error()),
			)
			cfg.ImplementationsConfig = getDefaultImplementationsConfig()
		}
	}

	// Загрузка конфигурации RAC
	if cfg.RacConfig, err = loadRacConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации RAC", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.RacConfig = getDefaultRacConfig()
	}

	// Загрузка конфигурации алертинга
	if cfg.AlertingConfig, err = loadAlertingConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации алертинга", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.AlertingConfig = getDefaultAlertingConfig()
	}
	// Fail-fast валидация: обнаруживаем невалидную конфигурацию при загрузке,
	// а не при первом использовании Alerter в runtime.
	// TODO (M-1/Review #17): При ошибке валидации код выставляет Enabled=false,
	// но невалидные поля (пустой SMTPHost и т.д.) остаются в структуре.
	// Рекомендуется заменять невалидную конфигурацию на getDefault*Config().
	if cfg.AlertingConfig != nil && cfg.AlertingConfig.Enabled {
		if valErr := validateAlertingConfig(cfg.AlertingConfig); valErr != nil {
			l.Warn("невалидная конфигурация алертинга, алертинг отключён",
				slog.String("error", valErr.Error()),
				slog.String("reason", "validation_failed"),
			)
			cfg.AlertingConfig.Enabled = false
		}
	}

	// Загрузка конфигурации метрик
	if cfg.MetricsConfig, err = loadMetricsConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации метрик", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.MetricsConfig = getDefaultMetricsConfig()
	}
	// Fail-fast валидация: обнаруживаем невалидную конфигурацию при загрузке.
	if cfg.MetricsConfig != nil && cfg.MetricsConfig.Enabled {
		if valErr := validateMetricsConfig(cfg.MetricsConfig); valErr != nil {
			l.Warn("невалидная конфигурация метрик, метрики отключены",
				slog.String("error", valErr.Error()),
				slog.String("reason", "validation_failed"),
			)
			cfg.MetricsConfig.Enabled = false
		}
	}

	// Загрузка конфигурации трейсинга
	if cfg.TracingConfig, err = loadTracingConfig(l, &cfg); err != nil {
		l.Warn("ошибка загрузки конфигурации трейсинга", slog.String("error", err.Error()))
		// Используем значения по умолчанию
		cfg.TracingConfig = getDefaultTracingConfig()
	}
	// Fail-fast валидация: обнаруживаем невалидную конфигурацию при загрузке.
	if cfg.TracingConfig != nil && cfg.TracingConfig.Enabled {
		if valErr := validateTracingConfig(cfg.TracingConfig); valErr != nil {
			l.Warn("невалидная конфигурация трейсинга, трейсинг отключён",
				slog.String("error", valErr.Error()),
				slog.String("reason", "validation_failed"),
			)
			cfg.TracingConfig.Enabled = false
		}
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
