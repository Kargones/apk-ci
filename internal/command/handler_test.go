package command

import (
	"context"

	"github.com/Kargones/apk-ci/internal/config"
)

// compileTimeHandler — проверка компиляции для интерфейса Handler.
// Этот тип используется только для compile-time проверки, что структура
// правильно реализует интерфейс Handler.
type compileTimeHandler struct{}

// Name реализует Handler.Name.
func (h *compileTimeHandler) Name() string {
	return "compile-time-check"
}

// Description реализует Handler.Description.
func (h *compileTimeHandler) Description() string {
	return "compile-time check handler"
}

// Execute реализует Handler.Execute.
func (h *compileTimeHandler) Execute(_ context.Context, _ *config.Config) error {
	return nil
}

// Compile-time проверка: compileTimeHandler должен реализовывать Handler.
// Если интерфейс изменится, компиляция тестов упадёт.
var _ Handler = (*compileTimeHandler)(nil)
