package dbrestore

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/Kargones/apk-ci/internal/config"

	"gopkg.in/yaml.v3"
)

// LoadDBRestoreConfig загружает конфигурацию восстановления базы данных из файла проекта.
// Ищет конфигурацию базы данных в файле проекта и создает объект DBRestore с соответствующими настройками.
// Параметры:
//   - l: логгер для записи сообщений
//   - cfg: общая конфигурация приложения
//   - dbName: имя базы данных для поиска в конфигурации
//
// Возвращает:
//   - *DBRestore: настроенный объект для восстановления базы данных
//   - error: ошибка загрузки конфигурации или nil при успехе
func LoadDBRestoreConfig(l *slog.Logger, cfg *config.Config, dbName string) (*DBRestore, error) {
	// Используем централизованную функцию загрузки
	dbCfg, err := config.LoadDBRestoreConfig(cfg, dbName)
	if err != nil {
		l.Error("Не удалось загрузить конфигурацию восстановления БД",
			slog.String("dbName", dbName),
			slog.String("error", err.Error()),
		)
		return nil, err
	}

	// Создаем экземпляр DBRestore с данными из централизованной конфигурации
	dbR := &DBRestore{
		Server:      dbCfg.Server,
		User:        dbCfg.User,
		Password:    dbCfg.Password,
		Database:    dbCfg.Database,
		Timeout:     dbCfg.Timeout,
		AutoTimeOut: dbCfg.Autotimeout,
		SrcServer:   dbCfg.SrcServer,
		SrcDB:       dbCfg.SrcDB,
		DstServer:   dbCfg.DstServer,
		DstDB:       dbCfg.DstDB,
		Port:        1433, // Значение по умолчанию
	}

	// Установка дополнительных значений по умолчанию
	setupDBRestoreDefaults(dbR)

	l.Debug("Конфигурация восстановления БД успешно загружена",
		slog.String("server", dbR.Server),
		slog.String("user", dbR.User),
		slog.String("database", dbR.Database),
		slog.Duration("timeout", dbR.Timeout),
	)

	return dbR, nil
}

// NewFromConfig создает новый экземпляр DBRestore на основе конфигурации проекта.
// Загружает настройки из файла проекта, устанавливает значения по умолчанию и инициализирует объект.
// Параметры:
//   - logger: логгер для записи сообщений и отладочной информации
//   - cfg: общая конфигурация приложения с путями к файлам
//   - dbName: имя базы данных для поиска в конфигурации проекта
//
// Возвращает:
//   - *DBRestore: полностью настроенный объект для восстановления базы данных
//   - error: ошибка создания или инициализации объекта, или nil при успехе
func NewFromConfig(logger *slog.Logger, cfg *config.Config, dbName string) (*DBRestore, error) {
	logger.Debug("Начало создания DBRestore из конфигурации", "dbName", dbName)

	if cfg == nil {
		logger.Debug("Ошибка: конфигурация равна nil")
		return nil, fmt.Errorf("config не может быть nil")
	}

	// Загружаем временную зону Europe/Moscow
	moscowTZ, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		logger.Warn("Не удалось загрузить временную зону Europe/Moscow, используется UTC", "error", err)
		moscowTZ = time.UTC
	}

	// Создаем новый экземпляр DBRestore
	nowInMoscow := time.Now().In(moscowTZ)
	dbR := &DBRestore{
		Port:            DefaultPort, // Значение по умолчанию
		Server:          DefaultServer,
		TimeToRestore:   nowInMoscow.Format("2006-01-02T15:04:05"),
		TimeToStatistic: nowInMoscow.AddDate(0, 0, -120).Format("2006-01-02T15:04:05"),
		Description:     "gitops db restore task",
	}
	// Сервер для подключения (обычно тот же, что и источник)
	logger.Debug("Установлен сервер подключения", "server", DefaultServer)

	logger.Debug("Создан новый экземпляр DBRestore с портом по умолчанию", "port", dbR.Port)

	// Заполняем данные из AppConfig
	if cfg.AppConfig != nil {
		logger.Debug("Обработка AppConfig")

		// Основные настройки восстановления БД
		if cfg.AppConfig.Dbrestore.Database != "" {
			dbR.Database = cfg.AppConfig.Dbrestore.Database
			logger.Debug("Установлена база данных из AppConfig", "database", dbR.Database)
		}
		dbR.AutoTimeOut = cfg.AppConfig.Dbrestore.Autotimeout
		logger.Debug("Установлен AutoTimeOut", "autoTimeout", dbR.AutoTimeOut)

		// Парсим таймаут из строки
		if cfg.AppConfig.Dbrestore.Timeout != "" {
			logger.Debug("Парсинг таймаута из строки", "timeoutString", cfg.AppConfig.Dbrestore.Timeout)
			if timeout, err := time.ParseDuration(cfg.AppConfig.Dbrestore.Timeout); err == nil {
				dbR.Timeout = timeout
				logger.Debug("Таймаут успешно распарсен", "timeout", dbR.Timeout)
			} else {
				logger.Debug("Ошибка парсинга таймаута", "error", err, "timeoutString", cfg.AppConfig.Dbrestore.Timeout)
			}
		}

		// Пользователи
		if cfg.AppConfig.Users.Mssql != "" {
			dbR.User = cfg.AppConfig.Users.Mssql
			logger.Debug("Установлен пользователь MSSQL", "user", dbR.User)
		}
	} else {
		logger.Debug("AppConfig отсутствует")
	}

	// Заполняем пароли из SecretConfig
	if cfg.SecretConfig != nil && cfg.SecretConfig.Passwords.Mssql != "" {
		dbR.Password = cfg.SecretConfig.Passwords.Mssql
		logger.Debug("Установлен пароль MSSQL из SecretConfig")
	} else {
		logger.Debug("SecretConfig отсутствует или пароль MSSQL не задан")
	}
	prodDbName := FindProductionDatabase(cfg.ProjectConfig, dbName)
	// Определяем серверы источника и назначения из DbConfig
	if cfg.DbConfig != nil && dbName != "" || prodDbName != "" {
		logger.Debug("Обработка DbConfig", "dbName", dbName, "dbConfigCount", len(cfg.DbConfig))
		if dbInfo, exists := cfg.DbConfig[prodDbName]; exists {
			logger.Debug("Найдена информация о базе данных в DbConfig", "dbServer", dbInfo.DbServer)

			dbR.SrcServer = dbInfo.DbServer
			dbR.SrcDB = prodDbName
			logger.Debug("Установлен сервер-источник", "srcServer", dbR.SrcServer, "srcDB", dbR.SrcDB)

			if dbInfo, exists := cfg.DbConfig[dbName]; exists {
				logger.Debug("Найдена информация о базе данных в DbConfig", "dbServer", dbInfo.DbServer)
				dbR.DstServer = dbInfo.DbServer
				dbR.DstDB = dbName
				logger.Debug("Установлен сервер назначения", "dstServer", dbR.DstServer, "dstDB", dbR.DstDB)
			} else {
				logger.Debug("База данных не найдена в DbConfig", "dbName", dbName)
				return nil, fmt.Errorf("база данных %s не найдена в DbConfig", dbName)
			}
		} else {
			logger.Debug("База данных не найдена в DbConfig", "dbName", prodDbName)
			return nil, fmt.Errorf("база данных %s не найдена в DbConfig", prodDbName)
		}
	} else {
		if cfg.DbConfig == nil {
			logger.Debug("DbConfig отсутствует")
		} else {
			logger.Debug("Имя базы данных пустое")
		}
	}

	// Устанавливаем значения по умолчанию только для таймаута
	if dbR.Timeout == 0 {
		dbR.Timeout = 30 * time.Second
		logger.Debug("Установлен таймаут по умолчанию", "timeout", dbR.Timeout)
	}

	logger.Debug("DBRestore успешно создан",
		"server", dbR.Server,
		"user", dbR.User,
		// ToDo: Обязательно удалить!
		"password", dbR.Password,
		"port", dbR.Port,
		"database", dbR.Database,
		"timeout", dbR.Timeout,
		"autoTimeout", dbR.AutoTimeOut,
		"srcServer", dbR.SrcServer,
		"srcDB", dbR.SrcDB,
		"dstServer", dbR.DstServer,
		"dstDB", dbR.DstDB)

	return dbR, nil
}

// setupDBRestoreDefaults устанавливает значения по умолчанию для DBRestore
func setupDBRestoreDefaults(dbR *DBRestore) {
	if dbR.User == "" {
		dbR.User = "gitops"
	}
	if dbR.Server == "" {
		dbR.Server = "MSK-SQL-SVC-01"
	}
	if dbR.Port == 0 {
		dbR.Port = 1433
	}
	if dbR.Database == "" {
		dbR.Database = "master"
	}
	if dbR.Description == "" {
		dbR.Description = "gitops db restore task"
	}
	if dbR.TimeToRestore == "" {
		dbR.TimeToRestore = time.Now().Format("2006-01-02T15:04:05")
	}
	if dbR.TimeToStatistic == "" {
		dbR.TimeToStatistic = time.Now().AddDate(0, 0, -120).Format("2006-01-02T15:04:05")
	}
	if dbR.Timeout == 0 {
		dbR.AutoTimeOut = true
	}
	if dbR.Password == "" {
		// Проверяем переменную окружения MSSQL_PASSWORD
		if mssqlPassword := os.Getenv("MSSQL_PASSWORD"); mssqlPassword != "" {
			dbR.Password = mssqlPassword
		} else {
			//ToDo: Заменить на шифрование yaml (sops)
			dbR.Password = "8kX8h!NNIbEbdu8o0"
		}
	}
}

// Init - устаревшая функция, оставлена для обратной совместимости
// Рекомендуется использовать LoadDBRestoreConfig
func (dbR *DBRestore) Init(yamlFile []byte) error {
	var configData map[string]DBRestore
	err := yaml.Unmarshal(yamlFile, &configData) // Need to import "gopkg.in/yaml.v2"
	if err != nil {
		return fmt.Errorf("ошибка разбора config.yaml: %w", err)
	}

	restoreConfig, ok := configData["tempdbrestore"]
	if !ok {
		return fmt.Errorf("в config.yaml отсутствует раздел 'tempdbrestore'")
	}
	*dbR = restoreConfig

	// Установка значений по умолчанию
	setupDBRestoreDefaults(dbR)
	return nil
}

// FindProductionDatabase анализирует ProjectConfig на основе предоставленного dbname,
// определяет, к какой производственной базе данных принадлежит указанная база,
// и возвращает имя этой производственной базы данных.
//
// Параметры:
//   - projectConfig: указатель на структуру ProjectConfig с конфигурацией проекта
//   - dbName: имя базы данных для поиска
//
// Возвращает:
//   - string: имя производственной базы данных, если найдена, иначе пустая строка
//
// Функция выполняет поиск в следующем порядке:
//  1. Проверяет, является ли dbName сама производственной базой
//  2. Ищет dbName среди связанных баз (related) каждой производственной базы
//  3. Проверяет store-db как потенциальную связанную базу
//
// Пример использования:
//
//	prodName := FindProductionDatabase(cfg.ProjectConfig, "V8_DEV_DSBEKETOV_APK_TOIR3_1")
//	if prodName != "" {
//		fmt.Printf("Производственная база: %s\n", prodName)
//	} else {
//		fmt.Println("Производственная база не найдена")
//	}
//
// FindProductionDatabase находит производственную базу данных по имени.
// Ищет указанную базу данных среди производственных баз и связанных с ними баз
// в конфигурации проекта. Возвращает имя производственной базы, к которой
// относится указанная база данных.
// Параметры:
//   - projectConfig: конфигурация проекта с описанием производственных баз
//   - dbName: имя базы данных для поиска
//
// Возвращает:
//   - string: имя производственной базы данных или пустую строку, если не найдена
func FindProductionDatabase(projectConfig *config.ProjectConfig, dbName string) string {
	if projectConfig == nil || dbName == "" {
		return ""
	}

	// Проверяем каждую производственную базу
	for prodName, prodInfo := range projectConfig.Prod {
		// 1. Проверяем, является ли dbName сама производственной базой
		if prodName == dbName {
			return prodName
		}

		// 2. Ищем среди связанных баз (related)
		if prodInfo.Related != nil {
			for relatedDbName := range prodInfo.Related {
				if relatedDbName == dbName {
					return prodName
				}
			}
		}
	}
	return ""
}
