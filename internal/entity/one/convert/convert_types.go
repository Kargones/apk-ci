// Package convert предоставляет функциональность для конвертации конфигураций 1С
package convert

import (
	"github.com/Kargones/apk-ci/internal/entity/one/designer"
	"github.com/Kargones/apk-ci/internal/entity/one/store"
)

// MergeSettingsString - строка настроек слияния конфигураций.
const MergeSettingsString string = `<?xml version="1.0" encoding="UTF-8"?>
<Settings xmlns="http://v8.1c.ru/8.3/config/merge/settings" xmlns:xs="http://www.w3.org/2001/XMLSchema" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" version="1.2" platformVersion="8.3.11">
	<Parameters>
		<AllowMainConfigurationObjectDeletion>true</AllowMainConfigurationObjectDeletion>
	</Parameters>
</Settings>`

// Константы перенесены в internal/constants/constants.go

// Config представляет конфигурацию для процесса конвертации данных.
// Содержит параметры источника, назначения и правила преобразования.
// Config представляет конфигурацию для процесса конвертации данных.
// Содержит параметры источника, назначения и правила преобразования.
type Config struct {
	StoreRoot string         `json:"Корень хранилища"`
	OneDB     designer.OneDb `json:"Параметры подключения"`
	Pair      []Pair         `json:"Сопоставления"`
}

// Pair представляет пару "источник-назначение" для операции конвертации.
// Определяет связь между исходными и целевыми данными.
// Pair представляет пару "источник-назначение" для операции конвертации.
// Определяет связь между исходными и целевыми данными.
type Pair struct {
	Source Source      `json:"Источник"`
	Store  store.Store `json:"Хранилище"`
}

// Source представляет источник данных для операции конвертации.
// Содержит информацию о расположении и параметрах доступа к исходным данным.
type Source struct {
	Name    string `json:"Имя"`
	RelPath string `json:"Относительный путь"`
	Main    bool   `json:"Основная конфигурация"`
}
