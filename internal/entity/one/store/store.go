// Package store предоставляет функциональность для работы с хранилищем конфигураций 1С
package store

import (
	"os"
	"path"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// TCPProtocol определяет префикс для TCP соединений
const TCPProtocol = "tcp://"

// Константы перенесены в internal/constants/constants.go

// Store представляет хранилище конфигурации 1С.
// Содержит информацию о подключении к хранилищу конфигураций,
// включая имя, путь, учетные данные и команды для работы с хранилищем.
type Store struct {
	Name    string `json:"Имя хранилища,omitempty"`
	Path    string `json:"Относительный путь"`
	User    string `json:"Пользователь"`
	Pass    string `json:"Пароль"`
	Command string `json:"-"`
}

// Record представляет запись в хранилище конфигураций.
// Содержит информацию о версии конфигурации, пользователе,
// дате создания и комментарии к изменениям в хранилище.
// Record представляет запись в хранилище конфигураций.
// Содержит информацию о версии конфигурации, пользователе,
// дате создания и комментарии к изменениям в хранилище.
// Record представляет запись в хранилище конфигураций.
// Содержит информацию о версии конфигурации, пользователе,
// дате создания и комментарии к изменениям в хранилище.
type Record struct {
	Version     int       `json:"Версия хранилища"`
	ConfVersion string    `json:"Версия конфигурации"`
	User        string    `json:"Пользователь"`
	Date        time.Time `json:"Дата время"`
	Comment     string    `json:"Комментарий,omitempty"`
}

// User представляет пользователя системы.
// Содержит информацию о пользователе хранилища конфигураций,
// включая имя в хранилище, доменное имя и электронную почту.
type User struct {
	StoreUserName string `json:"Имя пользователя в хранилище"`
	AccountName   string `json:"Доменное имя"`
	Email         string `json:"e-mail"`
}

func fullPathStore(storeRoot, relativePath string) (string, string) {
	var fullPathStore string
	if storeRoot[0:6] != TCPProtocol {
		fullPathStore = path.Join(storeRoot, relativePath)
	} else {
		fullPathStore = storeRoot + "/" + relativePath
	}
	return relativePath, fullPathStore
}

// CommentAdd добавляет комментарий к команде runner.
// Функция разбивает многострочный комментарий на отдельные строки
// и добавляет каждую строку как отдельный параметр -comment.
// Параметры:
//   - r: указатель на runner для добавления параметров
//   - comment: текст комментария (может быть многострочным)
func CommentAdd(r *runner.Runner, comment string) {
	as := strings.Split(comment, "\n")
	for _, s := range as {
		r.Params = append(r.Params, "-comment")
		r.Params = append(r.Params, "\""+s+"\"")
	}
}

// GetStoreParam возвращает параметры для работы с хранилищем конфигурации
// GetStoreParam создает и настраивает runner для работы с хранилищем конфигурации.
// Функция формирует базовые параметры подключения к хранилищу и базе данных.
// Параметры:
//   - dbConnectString: строка подключения к базе данных
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - runner.Runner: настроенный runner с базовыми параметрами
func (s *Store) GetStoreParam(dbConnectString string, cfg *config.Config) runner.Runner {
	r := runner.Runner{}
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.TmpDir = cfg.WorkDir

	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, dbConnectString)
	r.Params = append(r.Params, "/ConfigurationRepositoryF")
	r.Params = append(r.Params, s.Path)
	r.Params = append(r.Params, "/ConfigurationRepositoryN")
	r.Params = append(r.Params, s.User)
	r.Params = append(r.Params, "/ConfigurationRepositoryP")
	r.Params = append(r.Params, s.Pass)
	addDisableParam(&r)

	return r
}

// exists проверяет существование файла или директории
func exists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}

// addDisableParam добавляет параметры отключения диалогов
func addDisableParam(r *runner.Runner) {
	r.Params = append(r.Params, "/DisableStartupDialogs")
	r.Params = append(r.Params, "/DisableStartupMessages")
	r.Params = append(r.Params, "/DisableUnrecoverableErrorMessage")
	r.Params = append(r.Params, "/UC ServiceMode")
}
