Модифицируй структуру ConvertConfig и функцию Load следующим образом:

``` go
// convert.go
type ConvertConfig struct {
	StoreRoot   string         - получать следующим образом: config.StoreRoot+cfg.Owner+"/"+ cfg.RepName
	OneDB       designer.OneDb - получать следующим образом: "/S <(*cfg.DbConfig[<Имя базы переданное в параметре dbName через action.yaml>]).OneServer>\\<dbName>"
	ConvertPair []ConvertPair  `json:"Сопоставления"`
	SourceRoot  string         `json:"-"` - Удалить
}
type ConvertPair struct {
	Source Source      `json:"Источник"`
	Store  store.Store `json:"Хранилище"`
}

type Source struct {
	Name    string `json:"Имя"`						- имя берется из cfg.ProjectName и AddArray
	RelPath string `json:"Относительный путь"`		- если основная конфигурация то ConvertPair.Source.Main=true тогда "main" иначе "add/<Source.Name>"
	Main    bool   `json:"Основная конфигурация"`	- если основная конфигурация то ConvertPair.Source.Main=true
}
```

``` go
// store.go
type Store struct {
	Name    string `json:"Имя хранилища,omitempty"` - не заполнять
	Path    string `json:"Относительный путь"`      - если основная конфигурация то ConvertPair.Source.Main=true тогда "main" иначе "add/<Source.Name>"
	User    string `json:"Пользователь,omitempty"`  - cfg.AppConfig.Users.StoreAdmin
	Pass    string `json:"Пароль,omitempty"`        - cfg.SecretConfig.Passwords.StoreAdminPassword
	Command string `json:"-"`                       - не заполнять
}
```
Добавь к этому файлу уточняющие вопросы если это необходимо и дождись ответа на них и подтверждения пользователя для продолжения работы.
