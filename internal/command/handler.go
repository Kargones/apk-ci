// Package command предоставляет интерфейсы и реестр для команд приложения.
// Пакет реализует паттерн self-registration, позволяющий командам
// регистрироваться в реестре через init() без изменения main.go.
package command

import (
	"context"

	"github.com/Kargones/apk-ci/internal/config"
)

// Handler определяет интерфейс обработчика команды.
// Каждая команда приложения должна реализовывать этот интерфейс.
// Регистрация обработчиков происходит через функцию Register() в init().
type Handler interface {
	// Name возвращает имя команды для регистрации в реестре.
	// Должно соответствовать константам из internal/constants
	// (например, "service-mode-status", "nr-version").
	Name() string

	// Description возвращает описание команды для вывода в help.
	Description() string

	// Execute выполняет команду с переданным контекстом и конфигурацией.
	// Возвращает ошибку если выполнение завершилось неуспешно.
	Execute(ctx context.Context, cfg *config.Config) error
}
