// Package designer предоставляет функциональность для работы с 1С:Предприятие.
package designer

import (
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// Константы перенесены в internal/constants/constants.go

// OneDb представляет конфигурацию базы данных 1С:Предприятие.
// Содержит параметры подключения и настройки для работы с информационной базой.
type OneDb struct {
	DbConnectString   string `json:"Строка соединения,omitempty"`
	User              string `json:"Пользователь,omitempty"`
	Pass              string `json:"Пароль,omitempty"`
	FullConnectString string
	ServerDb          bool
	DbExist           bool
}

func addDisableParam(r *runner.Runner) {
	r.Params = append(r.Params, "/DisableStartupDialogs")
	r.Params = append(r.Params, "/DisableStartupMessages")
	r.Params = append(r.Params, "/DisableUnrecoverableErrorMessage")
	r.Params = append(r.Params, "/UC ServiceMode")
}
