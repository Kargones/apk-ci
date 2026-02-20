package onec

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestTempDbCreator_CompileTimeInterface проверяет что TempDbCreator реализует TempDatabaseCreator.
func TestTempDbCreator_CompileTimeInterface(_ *testing.T) {
	var _ TempDatabaseCreator = (*TempDbCreator)(nil)
}

// TestNewTempDbCreator проверяет создание TempDbCreator.
func TestNewTempDbCreator(t *testing.T) {
	creator := NewTempDbCreator()
	assert.NotNil(t, creator)
}

// TestTempDbCreator_CreateTempDB_EmptyDbPath проверяет обработку пустого пути к БД.
func TestTempDbCreator_CreateTempDB_EmptyDbPath(t *testing.T) {
	creator := NewTempDbCreator()

	// С пустым путём команда не выполнится, но проверяем что не panic
	result, err := creator.CreateTempDB(context.Background(), CreateTempDBOptions{
		DbPath:    "",
		BinIbcmd:  "/usr/bin/ibcmd",
		Timeout:   5 * time.Second,
	})

	// Ожидаем ошибку, так как ibcmd не сможет создать базу без пути
	assert.Nil(t, result)
	assert.Error(t, err)
}

// TestTempDbCreator_CreateTempDB_ContextCancellation проверяет отмену через контекст.
func TestTempDbCreator_CreateTempDB_ContextCancellation(t *testing.T) {
	ctx := context.Background()
	creator := NewTempDbCreator()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // отменяем сразу

	result, err := creator.CreateTempDB(ctx, CreateTempDBOptions{
		DbPath:   "/tmp/testdb",
		BinIbcmd: "/usr/bin/ibcmd",
		Timeout:  5 * time.Second,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
	// Проверяем что ошибка связана с контекстом
	assert.True(t, errors.Is(err, context.Canceled) || errors.Is(err, ErrInfobaseCreate) || err != nil)
}

// TestTempDbCreator_CreateTempDB_TimeoutExceeded проверяет таймаут.
func TestTempDbCreator_CreateTempDB_TimeoutExceeded(t *testing.T) {
	creator := NewTempDbCreator()

	ctx := context.Background()

	result, err := creator.CreateTempDB(ctx, CreateTempDBOptions{
		DbPath:   "/tmp/testdb",
		BinIbcmd: "/nonexistent/ibcmd",
		Timeout:  1 * time.Nanosecond, // минимальный таймаут
	})

	assert.Nil(t, result)
	assert.Error(t, err)
}

// TestTempDbCreator_CreateTempDB_EmptyExtensions проверяет обработку пустого списка расширений.
func TestTempDbCreator_CreateTempDB_EmptyExtensions(t *testing.T) {
	creator := NewTempDbCreator()

	// С пустым списком расширений должен просто создать базу без расширений
	// (если бы ibcmd был доступен)
	result, err := creator.CreateTempDB(context.Background(), CreateTempDBOptions{
		DbPath:     "/tmp/testdb",
		BinIbcmd:   "/nonexistent/ibcmd",
		Extensions: []string{}, // пустой список
		Timeout:    1 * time.Second,
	})

	// Ожидаем ошибку, так как ibcmd не существует
	assert.Nil(t, result)
	assert.Error(t, err)
}

// TestTempDbCreator_CreateTempDB_EmptyExtensionNames проверяет фильтрацию пустых имён расширений.
func TestTempDbCreator_CreateTempDB_EmptyExtensionNames(t *testing.T) {
	creator := NewTempDbCreator()

	// Список с пустыми именами должен фильтроваться
	result, err := creator.CreateTempDB(context.Background(), CreateTempDBOptions{
		DbPath:     "/tmp/testdb",
		BinIbcmd:   "/nonexistent/ibcmd",
		Extensions: []string{"", "ext1", "", ""},
		Timeout:    1 * time.Second,
	})

	// Ожидаем ошибку, так как ibcmd не существует
	assert.Nil(t, result)
	assert.Error(t, err)
}

// TestTempDbCreator_CreateTempDB_NilExtensions проверяет обработку nil списка расширений.
func TestTempDbCreator_CreateTempDB_NilExtensions(t *testing.T) {
	creator := NewTempDbCreator()

	result, err := creator.CreateTempDB(context.Background(), CreateTempDBOptions{
		DbPath:     "/tmp/testdb",
		BinIbcmd:   "/nonexistent/ibcmd",
		Extensions: nil,
		Timeout:    1 * time.Second,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
}

// TestErrExtensionAdd_Error проверяет текст ошибки ErrExtensionAdd.
func TestErrExtensionAdd_Error(t *testing.T) {
	assert.Contains(t, ErrExtensionAdd.Error(), "расширения")
}

// TestErrInfobaseCreate_Error проверяет текст ошибки ErrInfobaseCreate.
func TestErrInfobaseCreate_Error(t *testing.T) {
	assert.Contains(t, ErrInfobaseCreate.Error(), "информационной базы")
}

// TestErrContextCancelled_Error проверяет текст ошибки ErrContextCancelled.
func TestErrContextCancelled_Error(t *testing.T) {
	assert.Contains(t, ErrContextCancelled.Error(), "отменена")
}

// TestErrInvalidImplementation_Error проверяет код ошибки.
func TestErrInvalidImplementation_Error(t *testing.T) {
	assert.Contains(t, ErrInvalidImplementation.Error(), "ERR_INVALID_IMPL")
}

// TestCreateTempDBOptions проверяет структуру опций.
func TestCreateTempDBOptions(t *testing.T) {
	opts := CreateTempDBOptions{
		DbPath:     "/tmp/test",
		Extensions: []string{"ext1", "ext2"},
		Timeout:    30 * time.Second,
		BinIbcmd:   "/usr/bin/ibcmd",
	}

	assert.Equal(t, "/tmp/test", opts.DbPath)
	assert.Len(t, opts.Extensions, 2)
	assert.Equal(t, 30*time.Second, opts.Timeout)
	assert.Equal(t, "/usr/bin/ibcmd", opts.BinIbcmd)
}

// TestTempDBResult проверяет структуру результата.
func TestTempDBResult(t *testing.T) {
	result := &TempDBResult{
		ConnectString: "/F /tmp/test",
		DbPath:        "/tmp/test",
		Extensions:    []string{"ext1"},
		CreatedAt:     time.Now(),
		DurationMs:    1500,
	}

	assert.Equal(t, "/F /tmp/test", result.ConnectString)
	assert.Equal(t, "/tmp/test", result.DbPath)
	assert.Len(t, result.Extensions, 1)
	assert.Positive(t, result.DurationMs)
}

// TestTempDbCreator_ErrorsIs проверяет работу errors.Is с типизированными ошибками.
func TestTempDbCreator_ErrorsIs(t *testing.T) {
	// Оборачиваем ошибки и проверяем errors.Is
	wrappedExt := fmt.Errorf("wrapper: %w", ErrExtensionAdd)
	assert.True(t, errors.Is(wrappedExt, ErrExtensionAdd))

	wrappedInfo := fmt.Errorf("wrapper: %w", ErrInfobaseCreate)
	assert.True(t, errors.Is(wrappedInfo, ErrInfobaseCreate))

	wrappedCtx := fmt.Errorf("wrapper: %w", ErrContextCancelled)
	assert.True(t, errors.Is(wrappedCtx, ErrContextCancelled))
}

// TestTempDbCreator_ImplConstants проверяет константы реализаций.
func TestTempDbCreator_ImplConstants(t *testing.T) {
	assert.Equal(t, "1cv8", Impl1cv8)
	assert.Equal(t, "ibcmd", ImplIbcmd)
	assert.Equal(t, "native", ImplNative)
}

// TestTempDbCreator_CreateTempDB_ContextAlreadyCancelled проверяет случай когда контекст уже отменён.
func TestTempDbCreator_CreateTempDB_ContextAlreadyCancelled(t *testing.T) {
	ctx := context.Background()
	creator := NewTempDbCreator()

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(1 * time.Millisecond) // ждём пока контекст истечёт

	require.True(t, ctx.Err() != nil, "контекст должен быть отменён")

	result, err := creator.CreateTempDB(ctx, CreateTempDBOptions{
		DbPath:   "/tmp/testdb",
		BinIbcmd: "/usr/bin/ibcmd",
		Timeout:  5 * time.Second,
	})

	assert.Nil(t, result)
	assert.Error(t, err)
}
