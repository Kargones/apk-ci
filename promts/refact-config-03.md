Отредактируй файл docs\config-arch.md изходя из следующих требований:
- app.yaml, секции 
users:
  rac: "admin"
  db: "db_user"
  mssql: "mssql_user"
  и
 users:
  serviceMode:
    rac: "service_user"
    db: "service_db_user"
    mssql: "service_mssql_user" 

    содержат одинаковые данные. Поэтому можно объединить их в одну секцию users
    добавь туда поле storeAdmin для хранения имени пользователя для подключения к хранилищу конфигурации (ConfigurationRepositoryN)

- secret.yaml, 

passwords:
  rac: "secure_password"
  db: "db_password"
  mssql: "mssql_password"

  и 

  serviceMode:
  racPassword: "service_password"
  dbPassword: "service_db_password"
  mssqlPassword: "service_mssql_password"

  Содержат идентичные данные. Объедини их в одну секцию passwords.
  Добавь туда поле storeAdminPassword для хранения пароля для подключения к хранилищу конфигурации (ConfigurationRepositoryP)

Измени соответствующие структуры и связанные функции.