// Package onec содержит адаптеры для работы с 1C:Предприятие.
package onec

import "errors"

// Типизированные ошибки для операций с 1C.
// MEDIUM-1 fix (Review #4): использование errors.Is вместо строковых проверок.
var (
	// ErrExtensionAdd — ошибка при добавлении расширения в базу данных.
	ErrExtensionAdd = errors.New("ошибка добавления расширения")

	// ErrInfobaseCreate — ошибка при создании информационной базы.
	ErrInfobaseCreate = errors.New("ошибка создания информационной базы")

	// ErrContextCancelled — операция отменена через context.
	ErrContextCancelled = errors.New("операция отменена")

	// ErrInvalidImplementation возвращается при невалидном значении config.implementations.
	// Код ошибки: ERR_INVALID_IMPL (AC-4)
	ErrInvalidImplementation = errors.New("ERR_INVALID_IMPL: невалидное значение реализации")
)
