Добавь во все шаги файла Test all action.yaml следующую строку:
``` yaml
          configDbData: "dbconfig.yaml"
```
Добавь нужный параметр в action.yaml
Добавь загрузку данных из этого файла в DbConfig используя этот параметр.
Проверь код с помощью go vet ./...