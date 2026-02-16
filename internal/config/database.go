package config

import (
	"fmt"
	"os"
	"time"

	"github.com/ilyakaznacheev/cleanenv"
)

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
