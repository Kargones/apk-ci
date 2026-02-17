// Package edt предоставляет функциональность для работы с EDT (Enterprise Development Tools)
package edt

const (
	// XML2edt - путь к утилите xml2edt.
	XML2edt string = "xml2edt"
	// Edt2xml - путь к утилите edt2xml.
	Edt2xml   string = "edt2xml"
	formatXML string = "xml"
	formatEDT string = "edt"
)

// Convert представляет конфигурацию для конвертации между форматами EDT и XML.
// Содержит информацию о коммите, источнике, приемнике и сопоставлениях путей
// для выполнения процесса конвертации проекта.
type Convert struct {
	CommitSha1  string    `json:"Хеш коммита"`
	Source      Data      `json:"Источник"`
	Distination Data      `json:"Приемник"`
	Mappings    []Mapping `json:"Сопоставление путей"`
}

// Data представляет данные источника или приемника конвертации.
// Содержит информацию о формате данных и ветке репозитория
// для процесса конвертации.
type Data struct {
	Format string `json:"Формат"`
	Branch string `json:"Ветка"`
}

// Mapping представляет сопоставление путей для конвертации.
// Определяет соответствие между путями источника и приемника
// в процессе конвертации файлов проекта.
type Mapping struct {
	SourcePath      string `json:"Путь источника"`
	DistinationPath string `json:"Путь приемника"`
}

// Cli представляет интерфейс командной строки для работы с EDT.
// Содержит настройки путей, направления конвертации, рабочей области
// и информацию о последней ошибке выполнения операций.
// EdtCli представляет клиент командной строки для работы с EDT.
// Содержит настройки путей, направления конвертации, рабочей области
// и информацию о последней ошибке выполнения операций.
// Cli представляет клиент командной строки для работы с EDT.
// Содержит настройки путей, направления конвертации, рабочей области
// и информацию о последней ошибке выполнения операций.
type Cli struct {
	CliPath   string
	Direction string
	PathIn    string
	PathOut   string
	WorkSpace string
	Operation string
	LastErr   error
}
