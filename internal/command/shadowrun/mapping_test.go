package shadowrun

import (
	"context"
	"log/slog"
	"testing"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLegacyMapping(t *testing.T) {
	m := NewLegacyMapping()
	require.NotNil(t, m)
	assert.Empty(t, m.RegisteredCommands())
}

func TestLegacyMapping_RegisterAndGet(t *testing.T) {
	m := NewLegacyMapping()
	called := false
	fn := func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		called = true
		return nil
	}

	m.Register("nr-test-cmd", fn)

	got, ok := m.Get("nr-test-cmd")
	require.True(t, ok)
	require.NotNil(t, got)

	// Вызываем полученную функцию и проверяем что это та же самая
	ctx := context.Background()
	err := got(&ctx, slog.Default(), &config.Config{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestLegacyMapping_GetNotFound(t *testing.T) {
	m := NewLegacyMapping()

	got, ok := m.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, got)
}

func TestLegacyMapping_HasMapping(t *testing.T) {
	m := NewLegacyMapping()
	fn := func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error { return nil }

	m.Register("nr-cmd-1", fn)

	assert.True(t, m.HasMapping("nr-cmd-1"))
	assert.False(t, m.HasMapping("nr-cmd-2"))
}

func TestLegacyMapping_RegisteredCommands(t *testing.T) {
	m := NewLegacyMapping()
	fn := func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error { return nil }

	m.Register("nr-cmd-a", fn)
	m.Register("nr-cmd-b", fn)

	commands := m.RegisteredCommands()
	assert.Len(t, commands, 2)
	assert.Contains(t, commands, "nr-cmd-a")
	assert.Contains(t, commands, "nr-cmd-b")
}

func TestLegacyMapping_OverwriteMapping(t *testing.T) {
	m := NewLegacyMapping()
	callCount := 0
	fn1 := func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		callCount = 1
		return nil
	}
	fn2 := func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
		callCount = 2
		return nil
	}

	m.Register("nr-cmd", fn1)
	m.Register("nr-cmd", fn2) // Перезаписываем

	got, ok := m.Get("nr-cmd")
	require.True(t, ok)
	ctx := context.Background()
	_ = got(&ctx, slog.Default(), &config.Config{})
	assert.Equal(t, 2, callCount, "должна быть вызвана вторая функция")
}

func TestLegacyMapping_StateChanging(t *testing.T) {
	m := NewLegacyMapping()
	fn := func(ctx *context.Context, l *slog.Logger, cfg *config.Config) error { return nil }

	m.Register("nr-enable", fn)
	m.Register("nr-status", fn)

	m.MarkStateChanging("nr-enable")

	assert.True(t, m.IsStateChanging("nr-enable"))
	assert.False(t, m.IsStateChanging("nr-status"))
	assert.False(t, m.IsStateChanging("nr-nonexistent"))
}
