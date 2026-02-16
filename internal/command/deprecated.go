package command

import (
	"context"
	"fmt"
	"os"

	"github.com/Kargones/apk-ci/internal/config"
)

// Deprecatable опционально реализуется deprecated handlers.
// Используется help-командой для определения deprecated статуса.
type Deprecatable interface {
	// IsDeprecated возвращает true если команда deprecated.
	IsDeprecated() bool
	// NewName возвращает новое рекомендуемое имя команды.
	NewName() string
}

// Compile-time проверки реализации интерфейсов.
var (
	_ Handler      = (*DeprecatedBridge)(nil)
	_ Deprecatable = (*DeprecatedBridge)(nil)
)

// DeprecatedBridge оборачивает handler для поддержки deprecated имён команд.
// При вызове Execute выводит warning в stderr с рекомендацией перехода на новое имя,
// затем делегирует выполнение actual handler.
//
// DeprecatedBridge является частью паттерна NR-Migration Bridge, который позволяет:
//   - Поддерживать старые имена команд во время миграции
//   - Уведомлять пользователей о необходимости миграции
//   - Постепенно переводить пользователей на новые имена без breaking changes
//
// Warning выводится в stderr (не stdout) чтобы не нарушить JSON output команд.
// Warning выводится при каждом вызове Execute (не кэшируется).
type DeprecatedBridge struct {
	// actual — реальный обработчик команды
	actual Handler
	// deprecated — старое (deprecated) имя команды
	deprecated string
	// newName — новое рекомендуемое имя команды
	newName string
}

// Name возвращает deprecated имя команды.
// Это имя используется для регистрации в реестре под старым именем.
func (b *DeprecatedBridge) Name() string {
	return b.deprecated
}

// Description делегирует вызов actual handler для получения описания команды.
func (b *DeprecatedBridge) Description() string {
	return b.actual.Description()
}

// IsDeprecated возвращает true — DeprecatedBridge всегда deprecated.
func (b *DeprecatedBridge) IsDeprecated() bool {
	return true
}

// NewName возвращает новое рекомендуемое имя команды.
func (b *DeprecatedBridge) NewName() string {
	return b.newName
}

// Execute выполняет команду через actual handler, предварительно
// выводя warning о deprecated статусе команды в stderr.
//
// Warning содержит:
//   - старое (deprecated) имя команды
//   - новое рекомендуемое имя команды
//   - слово "deprecated" для поиска в логах
//
// Warning выводится в stderr чтобы не нарушить JSON output в stdout.
// Warning выводится при каждом вызове (не кэшируется).
//
// Если context уже отменён, возвращает ctx.Err() без вывода warning
// и без вызова actual handler.
func (b *DeprecatedBridge) Execute(ctx context.Context, cfg *config.Config) error {
	// Проверяем context перед выполнением
	if err := ctx.Err(); err != nil {
		return err
	}
	// Warning в stderr (не stdout!) чтобы не нарушить JSON output
	fmt.Fprintf(os.Stderr, "WARNING: command '%s' is deprecated, use '%s' instead\n",
		b.deprecated, b.newName)
	return b.actual.Execute(ctx, cfg)
}
