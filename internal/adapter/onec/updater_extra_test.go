package onec

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestUpdater_CompileTimeInterface проверяет что Updater реализует DatabaseUpdater.
func TestUpdater_CompileTimeInterface(_ *testing.T) {
	var _ DatabaseUpdater = (*Updater)(nil)
}

// TestUpdater_NewWithPaths проверяет создание Updater.
func TestUpdater_NewWithPaths(t *testing.T) {
	u := NewUpdater("/usr/bin/1cv8", "/work", "/tmp")

	assert.NotNil(t, u)
	assert.Equal(t, "/usr/bin/1cv8", u.bin1cv8)
	assert.Equal(t, "/work", u.workDir)
	assert.Equal(t, "/tmp", u.tmpDir)
}

// TestUpdater_NewWithEmptyPaths проверяет создание Updater с пустыми путями.
func TestUpdater_NewWithEmptyPaths(t *testing.T) {
	u := NewUpdater("", "", "")

	assert.NotNil(t, u)
	assert.Empty(t, u.bin1cv8)
	assert.Empty(t, u.workDir)
	assert.Empty(t, u.tmpDir)
}

// TestUpdater_UpdateDBCfg_EmptyBinPath проверяет ошибку при пустом пути к 1cv8.
func TestUpdater_UpdateDBCfg_EmptyBinPath(t *testing.T) {
	u := NewUpdater("", "/work", "/tmp")

	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "путь к 1cv8 не указан")
}

// TestUpdater_UpdateDBCfg_EmptyBinPathWithOption проверяет что Bin1cv8 из опций используется.
func TestUpdater_UpdateDBCfg_EmptyBinPathWithOption(t *testing.T) {
	u := NewUpdater("", "/work", "/tmp")

	// Bin1cv8 из опций пустой, должна быть ошибка
	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
		Bin1cv8:       "", // пустой
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "путь к 1cv8 не указан")
}

// TestUpdater_UpdateDBCfg_WithContextTimeout проверяет использование таймаута.
func TestUpdater_UpdateDBCfg_WithContextTimeout(t *testing.T) {
	u := NewUpdater("/nonexistent/1cv8", "/work", "/tmp")

	ctx := context.Background()
	result, err := u.UpdateDBCfg(ctx, UpdateOptions{
		ConnectString: "/S server\\base",
		Timeout:       1 * time.Nanosecond, // минимальный таймаут
	})

	// Ожидаем ошибку и результат (метод возвращает оба)
	assert.NotNil(t, result)
	assert.Error(t, err)
	assert.False(t, result.Success)
}

// TestUpdater_UpdateDBCfg_CancelledContext проверяет отмену контекста.
func TestUpdater_UpdateDBCfg_CancelledContext(t *testing.T) {
	u := NewUpdater("/nonexistent/1cv8", "/work", "/tmp")

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем сразу

	result, err := u.UpdateDBCfg(ctx, UpdateOptions{
		ConnectString: "/S server\\base",
	})

	assert.NotNil(t, result)
	assert.Error(t, err)
	assert.False(t, result.Success)
}

// TestUpdater_UpdateDBCfg_WithExtension проверяет опции расширения.
func TestUpdater_UpdateDBCfg_WithExtension(t *testing.T) {
	u := NewUpdater("/nonexistent/1cv8", "/work", "/tmp")

	// Проверяем что с расширением не паникует
	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
		Extension:     "MyExtension",
		Timeout:       1 * time.Nanosecond,
	})

	assert.NotNil(t, result)
	assert.Error(t, err)
	assert.False(t, result.Success)
}

// TestUpdateOptions проверяет структуру опций.
func TestUpdateOptions(t *testing.T) {
	opts := UpdateOptions{
		ConnectString: "/S server\\base /N admin /P pass",
		Extension:     "TestExt",
		Timeout:       5 * time.Minute,
		Bin1cv8:       "/opt/1cv8/x86_64/1cv8",
	}

	assert.Equal(t, "/S server\\base /N admin /P pass", opts.ConnectString)
	assert.Equal(t, "TestExt", opts.Extension)
	assert.Equal(t, 5*time.Minute, opts.Timeout)
	assert.Equal(t, "/opt/1cv8/x86_64/1cv8", opts.Bin1cv8)
}

// TestUpdateResult проверяет структуру результата.
func TestUpdateResult(t *testing.T) {
	result := &UpdateResult{
		Success:    true,
		Messages:   []string{"Обновление завершено"},
		DurationMs: 5000,
	}

	assert.True(t, result.Success)
	assert.Len(t, result.Messages, 1)
	assert.Equal(t, int64(5000), result.DurationMs)
}

// TestUpdateResult_Failed проверяет структуру неудачного результата.
func TestUpdateResult_Failed(t *testing.T) {
	result := &UpdateResult{
		Success:    false,
		Messages:   []string{"Ошибка обновления", "Конфликт"},
		DurationMs: 1000,
	}

	assert.False(t, result.Success)
	assert.Len(t, result.Messages, 2)
}

// TestUpdater_UpdateDBCfg_OverridesBinPath проверяет что Bin1cv8 из опций переопределяет.
func TestUpdater_UpdateDBCfg_OverridesBinPath(t *testing.T) {
	// Updater имеет пустой путь, но опция предоставляет путь
	u := NewUpdater("", "/work", "/tmp")

	// Bin1cv8 из опций используется, но файл не существует
	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
		Bin1cv8:       "/nonexistent/1cv8",
		Timeout:       1 * time.Nanosecond,
	})

	assert.NotNil(t, result)
	assert.Error(t, err)
	assert.False(t, result.Success)
}
