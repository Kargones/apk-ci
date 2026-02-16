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
	"github.com/Kargones/apk-ci/internal/util/runner"
	"github.com/ilyakaznacheev/cleanenv"
	"gopkg.in/yaml.v3"
)

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
		WorkDir:  "/tmp/apk-ci",
		TmpDir:   "/tmp/apk-ci/temp",
		Timeout:  30,
		Paths: struct {
			Bin1cv8  string `yaml:"bin1cv8"`
			BinIbcmd string `yaml:"binIbcmd"`
			EdtCli   string `yaml:"edtCli"`
			Rac      string `yaml:"rac"`
		}{
			Bin1cv8:  constants.OneCBinPath(constants.Default1CVersion),
			BinIbcmd: constants.OneCIbcmdPath(constants.Default1CVersion),
			EdtCli:   constants.DefaultEdtCliPath,
			Rac:      constants.OneCRacPath(constants.Default1CVersion),
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
