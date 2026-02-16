При запуске такого шага:
inputs.action_1_dbName = "V8_DEV_DSBEKETOV_APK_TOIR3"
action_1_dbName: 
    steps:
      # Step 1: Service Mode Enable DB
      - name: Service Mode Enable DB
        if: ${{ inputs.action_1 == true }}
        uses: https://${{ secrets.READER_TOKEN }}:@git.benadis.ru/gitops/benadis-runner@${{ inputs.version }}
        with:
          giteaURL: ${{ gitea.server_url }}
          repository: ${{ gitea.repository }}
          accessToken: ${{ secrets.TOKEN_FULL }}
          actor: ${{ gitea.actor }}
          configSystem: "app.yaml"
          configProject: "project.yaml"
          configSecret: "secret.yaml"
          command: 'service-mode-enable-db'
          dbName: ${{ inputs.action_1_dbName }}
          logLevel: 'Debug'

возникает такой лог:
{"time":"2025-08-09T06:29:14.661509146Z","level":"DEBUG","source":{"function":"github.com/Kargones/apk-ci/internal/config.MustLoad","file":"config.go","line":320},"msg":"Проверка переменных","App info":{"version":"0.8.7.1"},"Проверка переменных среды":"/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin:/usr/local/go/bin:/opt/1C/1CE/components/1c-edt-2024.2.5+16-x86_64:/opt/1cv8/x86_64/8.3.27.1606"}
{"time":"2025-08-09T06:29:14.700911094Z","level":"DEBUG","source":{"function":"github.com/Kargones/apk-ci/internal/config.GetConfigData","file":"config.go","line":661},"msg":"Config GetConfigData HTTP request","App info":{"version":"0.8.7.1"},"fileURL":"https://regdv.apkholding.ru/api/v1/repos/test/toir-100/contents/app.yaml?ref=main","filename":"app.yaml"}
{"time":"2025-08-09T06:29:14.72618046Z","level":"DEBUG","source":{"function":"github.com/Kargones/apk-ci/internal/config.GetConfigData","file":"config.go","line":661},"msg":"Config GetConfigData HTTP request","App info":{"version":"0.8.7.1"},"fileURL":"https://regdv.apkholding.ru/api/v1/repos/test/toir-100/contents/project.yaml?ref=main","filename":"project.yaml"}
{"time":"2025-08-09T06:29:14.772555498Z","level":"DEBUG","source":{"function":"github.com/Kargones/apk-ci/internal/config.MustLoad","file":"config.go","line":456},"msg":"Config","App info":{"version":"0.8.7.1"},"Параметры конфигурации":{"Actor":"xor","Env":"dev","Command":"service-mode-enable-db","Logger":{},"ConfigSystem":"# Конфигурация приложения benadis-runner\r\n# Системные настройки приложения\r\napp:\r\n  logLevel: \"Debug\"\r\n  workDir: \"/tmp/benadis\"\r\n  tmpDir: \"/tmp/benadis/temp\"\r\n  timeout: 30\r\n\r\n# Пути к исполняемым файлам 1С\r\npaths:\r\n  bin1cv8: \"/opt/1cv8/x86_64/8.3.27.1606/1cv8\"\r\n  binIbcmd: \"/opt/1cv8/x86_64/8.3.27.1606/ibcmd\"\r\n  edtCli: \"/opt/1C/1CE/components/1c-edt-2024.2.6+7-x86_64/1cedtcli\"\r\n  rac: \"/opt/1cv8/x86_64/8.3.27.1606/rac\"\r\n\r\n# Настройки RAC (Remote Administration Console)\r\nrac:\r\n  port: 1545\r\n  timeout: 30\r\n  retries: 3\r\n\r\n# Пользователи системы\r\nusers:\r\n  rac: \"admin\"\r\n  db: \"db_user\"\r\n  mssql: \"mssql_user\"\r\n  storeAdmin: \"store_admin\"\r\n\r\n# Настройки для модуля dbrestore\r\ndbrestore:\r\n  database: \"master\"\r\n  timeout: \"50s\"\r\n  autotimeout: true","ConfigProject":"# База для для сборки конфигурации в хранилище. Может быть только одна на проект.\r\nstore-db: V8_DEV_DSBEKETOV_STORE_ERP\r\n# Список оперативных баз 1С привязанных к проекту.\r\nprod:\r\n# Продуктивная база. Может быть несколько оперативных баз привязанных к одному проекту.\r\n  V8_OPER_APK_TOIR3:\r\n    # Наименование оперативной базы. Будет использоваться в меню выбора базы.\r\n    dbName: \"База ТОИР\"\r\n    # Заблокированные к установке расширения. Действует для всех привязанных баз\r\n    add-disable:\r\n      - апк_ДоработкиТОИР\r\n    # Привязанные к продуктивной базы тестового контура, демо и дев.\r\n    related:\r\n      V8_DEV_DSBEKETOV_APK_TOIR3:\r\n      V8_DEV_DSBEKETOV_APK_TOIR3_1:\r\n\r\n\r\n\r\n\r\n","ConfigSecret":"secret.yaml","AppConfig":null,"ProjectConfig":null,"SecretConfig":null,"DbConfig":null,"InfobaseName":"","TerminateSessions":false,"IssueNumber":1,"GiteaURL":"https://regdv.apkholding.ru","Owner":"test","Repo":"toir-100","BaseBranch":"main","NewBranch":"testMerge","ProxyURL":"","Bin1cv8":"/opt/1cv8/x86_64/8.3.27.1606/1cv8","BinIbcmd":"/opt/1cv8/x86_64/8.3.27.1606/ibcmd","Connect":"","RepPath":"/tmp/4del/rep","WorkDir":"/tmp/4del","TmpDir":"/tmp","ConfigName":"config.json","G2SConfigName":"g2s.json","EdtCli":"/opt/1C/1CE/components/1c-edt-2024.2.6+7-x86_64/1cedtcli","Edt":"edt","AccessToken":"***","RacPath":"rac","RacServer":"localhost","RacPort":1545,"RacUser":"","RacPassword":"","RacTimeout":30,"RacRetries":3,"DbUser":"","DbPassword":"","PathIn":"/tmp/4del/rep/Tester","PathOut":"/tmp/4del/xml/Tester","WorkSpace":"/tmp/4del/ws01"}}
Из лога видно что конфигурация из файлов указанных в:
          configSystem: "app.yaml"
          configProject: "project.yaml"
          configSecret: "secret.yaml"
Проверь код парсинга переданных параметров о загрузки конфигурации на основе данных из этих параметров и исправь его.
Данные файлов конфигурации должны получатся в результате загрузки их с помощью GetConfigData в которую передаются параметры из:
          configSystem: "app.yaml"
          configProject: "project.yaml"
          configSecret: "secret.yaml"

По окончании выполнения этих инструкций выполни go vet ./... и исправь ошибки.