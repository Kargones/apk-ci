Файл конфигурации из которого загружается ConvertConfig следующего содержания:
```json
{
    "Имя ветки": "xml",
    "Хеш коммита": "",
    "Корень хранилища": "tcp://dev-1c-repo.apkholding.ru/gitops/TOIR",
    "Параметры подключения": {
        "Строка соединения": "/S MSK-TS-AS-001.apkholding.ru\\V8_CICF_APK_TOIR3",
        "Пользователь": "-",
        "Пароль": "-"
        },
    "Сопоставления": [
        {
            "Источник": {
                "Имя": "Main",
                "Относительный путь": "src/cfg",
                "Основная конфигурация": true
            },
            "Хранилище": {
                "Относительный путь": "MAIN",
                "Пользователь": "gitops",
                "Пароль": "gitops"
            }
        },
        {
            "Источник": {
                "Имя": "апк_ДоработкиТОИР",
                "Относительный путь": "src/cfe/апк_ДоработкиТОИР",
                "Основная конфигурация": false
            },
            "Хранилище": {
                "Относительный путь": "add/апк_ДоработкиТОИР",
                "Пользователь": "gitops",
                "Пароль": "gitops"
            }
        }
    ]
}
```
При этом в полях:
        "Пользователь": "-",
        "Пароль": "-"
передается знак "-" который означает, что значение не должно быть использовано.
Но в дебаг лог выводится следующее состояние конфигурации: 

``` json
{"time":"2025-08-07T10:35:51.849532618Z","level":"DEBUG","source":{"function":"github.com/Kargones/apk-ci/internal/config.MustLoad","file":"config.go","line":367},"msg":"Config","App info":{"version":"0.7.23.1"},"Параметры конфигурации":{"Actor":"xor","Env":"dev","Bin1cv8":"/opt/1cv8/x86_64/8.3.27.1606/1cv8","BinIbcmd":"/opt/1cv8/x86_64/8.3.27.1606/ibcmd","Connect":"","RepPath":"/tmp/4del/rep","WorkDir":"/tmp/4del","TmpDir":"/tmp","ConfigName":"config.json","G2SConfigName":"g2s.json","EdtCli":"/opt/1C/1CE/components/1c-edt-2024.2.6+7-x86_64/1cedtcli","Edt":"edt","Logger":{},"GiteaURL":"https://regdv.apkholding.ru","Owner":"test","Repo":"toir-100","AccessToken":"***","BaseBranch":"main","NewBranch":"testMerge","ProxyURL":"","Command":"git2store","IssueNumber":1,"RacPath":"rac","RacServer":"localhost","RacPort":1545,"RacUser":"","RacPassword":"","RacTimeout":30,"RacRetries":3,"DbUser":"","DbPassword":"","ConfigSystem":"config-system.yaml","ConfigProject":"config-project.yaml","ConfigSecret":"config-secret.yaml","PathIn":"/tmp/4del/rep/Tester","PathOut":"/tmp/4del/xml/Tester","WorkSpace":"/tmp/4del/ws01"}}
```
из которого видно что:
"DbUser":"","DbPassword":""

проанализируй код проекта, найди и исправь ошибку.
