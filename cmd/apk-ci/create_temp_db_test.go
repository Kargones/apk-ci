// Package main содержит тесты для команды create-temp-db.
package main

import (
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/constants"
)

// TestCreateTempDbCommandExists проверяет, что команда create-temp-db добавлена в константы
func TestCreateTempDbCommandExists(t *testing.T) {
	// Проверяем, что константа ActCreateTempDb определена
	if constants.ActCreateTempDb == "" {
		t.Error("Константа ActCreateTempDb не должна быть пустой")
	}

	// Проверяем правильное значение константы
	expectedValue := "create-temp-db"
	if constants.ActCreateTempDb != expectedValue {
		t.Errorf("Ожидалось значение %s, получено %s", expectedValue, constants.ActCreateTempDb)
	}
}

// TestCreateTempDbEnvironmentSetup проверяет установку переменных окружения для команды
func TestCreateTempDbEnvironmentSetup(t *testing.T) {
	// Сохраняем оригинальные переменные окружения
	originalCommand := os.Getenv("BR_COMMAND")
	originalLogLevel := os.Getenv("BR_LOGLEVEL")

	defer func() {
		// Восстанавливаем оригинальные значения
		if originalCommand == "" {
			_ = os.Unsetenv("BR_COMMAND")
		} else {
			_ = os.Setenv("BR_COMMAND", originalCommand)
		}
		if originalLogLevel == "" {
			_ = os.Unsetenv("BR_LOGLEVEL")
		} else {
			_ = os.Setenv("BR_LOGLEVEL", originalLogLevel)
		}
	}()

	// Устанавливаем переменные окружения для команды create-temp-db
	err := os.Setenv("BR_COMMAND", constants.ActCreateTempDb)
	if err != nil {
		t.Fatalf("Не удалось установить BR_COMMAND: %v", err)
	}

	err = _ = os.Setenv("BR_LOGLEVEL", "Debug")
	if err != nil {
		t.Fatalf("Не удалось установить BR_LOGLEVEL: %v", err)
	}

	// Проверяем, что переменные установлены корректно
	if os.Getenv("BR_COMMAND") != constants.ActCreateTempDb {
		t.Error("BR_COMMAND не установлена корректно")
	}

	if os.Getenv("BR_LOGLEVEL") != "Debug" {
		t.Error("BR_LOGLEVEL не установлена корректно")
	}

	t.Logf("Переменные окружения для команды %s установлены корректно", constants.ActCreateTempDb)
}
