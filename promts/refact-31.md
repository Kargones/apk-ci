Из лога видно что в функцию GetConfigData  происходит передача параметра filename":"config-system.yaml хотя входящий параметр пустой и значение должно браться из константы:
ConfigSystemDefault  string = " https://regdv.apkholding.ru/api/v1/repos/gitops-tools/gitops_congif/contents/app.yaml?ref=main"
Значение по умолчанию установлено на файл расположенный в репозитории для хранения конфигурации системы.
Найди почему вместо значения из константы подставляется 