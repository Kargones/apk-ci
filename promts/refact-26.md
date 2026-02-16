Измени функцию LoadServiceModeConfigFromSecret следующим образом:
- загрузка полей структуры должна производиться из:
		RacPath:     (*cfg.AppConfig).Paths.Rac
		RacServer:   Данное поле должно получаться из (*cfg.DbConfig[<Имя базы переданное в параметре dbName через action.yaml>]).OneServer
		RacPort:     (*cfg.AppConfig).Rac.Port
		RacUser:     (*cfg.AppConfig).Users.Rac
		RacPassword: (*cfg.SecretConfig).Passwords.Rac,
		DbUser:      (*cfg.AppConfig).Users.Db,
		DbPassword:  (*cfg.SecretConfig).Passwords.Db,
		RacTimeout:  (*cfg.AppConfig).Rac.Timeout
		RacRetries:  (*cfg.AppConfig).Rac.Retries
- Удали все   cfg.Logger.Debug
- Убери параметр   configSecret из функции LoadServiceModeConfigFromSecret
- убери загрузку данных из файла и проверку его

