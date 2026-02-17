package store

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// CreateStores создает хранилища конфигурации для основной конфигурации и расширений.
// Функция создает основное хранилище и хранилища для всех указанных расширений.
// Параметры:
//   - l: логгер для записи сообщений
//   - cfg: конфигурация приложения
//   - storeRoot: корневой путь к хранилищу
//   - dbConnectString: строка подключения к базе данных
//   - arrayAdd: массив имен расширений для создания хранилищ
//
// Возвращает:
//   - error: ошибку, если операция не удалась
func CreateStores(l *slog.Logger, cfg *config.Config, storeRoot string, dbConnectString string, arrayAdd []string) error {
	l.Debug("Начало создания хранилищ",
		slog.String("storeRoot", storeRoot),
		slog.String("dbConnectString", dbConnectString),
		slog.Any("arrayAdd", arrayAdd),
	)

	// Создаем основное хранилище конфигурации
	mainStore := &Store{
		Path: filepath.Join(storeRoot, "main"),
	}

	l.Debug("Создание основного хранилища конфигурации",
		slog.String("path", mainStore.Path),
	)

	// Встроенная логика создания основного хранилища (из метода Create)
	r := runner.Runner{}
	r.RunString = cfg.AppConfig.Paths.Bin1cv8
	r.WorkDir = cfg.WorkDir
	r.TmpDir = cfg.WorkDir

	r.Params = append(r.Params, "@")
	r.Params = append(r.Params, "DESIGNER")
	r.Params = append(r.Params, dbConnectString)
	r.Params = append(r.Params, "/ConfigurationRepositoryF")
	r.Params = append(r.Params, mainStore.Path)
	r.Params = append(r.Params, "/ConfigurationRepositoryN")
	r.Params = append(r.Params, cfg.AppConfig.Users.StoreAdmin)
	r.Params = append(r.Params, "/ConfigurationRepositoryP")
	r.Params = append(r.Params, cfg.SecretConfig.Passwords.StoreAdminPassword)
	// Добавляем параметры отключения диалогов
	r.Params = append(r.Params, "/DisableStartupDialogs")
	r.Params = append(r.Params, "/DisableStartupMessages")
	r.Params = append(r.Params, "/DisableUnrecoverableErrorMessage")
	r.Params = append(r.Params, "/UC ServiceMode")
	// Параметры создания хранилища
	r.Params = append(r.Params, "/ConfigurationRepositoryCreate")
	r.Params = append(r.Params, "-AllowConfigurationChanges")
	r.Params = append(r.Params, "-ChangesAllowedRule")
	r.Params = append(r.Params, "ObjectIsEditableSupportEnabled")
	r.Params = append(r.Params, "-ChangesNotRecommendedRule")
	r.Params = append(r.Params, "ObjectIsEditableSupportEnabled")
	r.Params = append(r.Params, "/Out")
	r.Params = append(r.Params, "/c Создание хранилища конфигурации")

	_, err := r.RunCommand(context.Background(), l)
	if err != nil {
		l.Error("Ошибка создания хранилища конфигурации",
			slog.String("Путь", mainStore.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Error", err.Error()),
		)
		return fmt.Errorf("ошибка создания хранилища конфигурации")
	}
	if !strings.Contains(string(r.FileOut), constants.SearchMsgStoreCreateOk) {
		l.Error("Неопознанная ошибка создания хранилища конфигурации",
			slog.String("Путь", mainStore.Path),
			slog.String("Строка подключения", dbConnectString),
			slog.String("Вывод в файл", runner.TrimOut(r.FileOut)),
		)
		return fmt.Errorf("ошибка создания хранилища конфигурации")
	}

	l.Debug("Основное хранилище конфигурации успешно создано",
		slog.String("path", mainStore.Path),
	)

	// Создаем хранилища расширений для каждого элемента в arrayAdd
	for i, addName := range arrayAdd {
		l.Debug("Создание хранилища расширения",
			slog.Int("index", i),
			slog.String("extensionName", addName),
		)

		addStore := &Store{
			Path: filepath.Join(storeRoot, "add", addName),
		}

		l.Debug("Создание хранилища расширения",
			slog.String("path", addStore.Path),
			slog.String("extensionName", addName),
		)

		// Встроенная логика создания хранилища расширения (из метода CreateAdd)
		rAdd := runner.Runner{}
		rAdd.RunString = cfg.AppConfig.Paths.Bin1cv8
		rAdd.WorkDir = cfg.WorkDir
		rAdd.TmpDir = cfg.WorkDir

		rAdd.Params = append(rAdd.Params, "@")
		rAdd.Params = append(rAdd.Params, "DESIGNER")
		rAdd.Params = append(rAdd.Params, dbConnectString)
		rAdd.Params = append(rAdd.Params, "/ConfigurationRepositoryF")
		rAdd.Params = append(rAdd.Params, addStore.Path)
		rAdd.Params = append(rAdd.Params, "/ConfigurationRepositoryN")
		rAdd.Params = append(rAdd.Params, cfg.AppConfig.Users.StoreAdmin)
		rAdd.Params = append(rAdd.Params, "/ConfigurationRepositoryP")
		rAdd.Params = append(rAdd.Params, cfg.SecretConfig.Passwords.StoreAdminPassword)
		// Добавляем параметры отключения диалогов
		rAdd.Params = append(rAdd.Params, "/DisableStartupDialogs")
		rAdd.Params = append(rAdd.Params, "/DisableStartupMessages")
		rAdd.Params = append(rAdd.Params, "/DisableUnrecoverableErrorMessage")
		rAdd.Params = append(rAdd.Params, "/UC ServiceMode")
		// Параметры создания хранилища расширения
		rAdd.Params = append(rAdd.Params, "/ConfigurationRepositoryCreate")
		rAdd.Params = append(rAdd.Params, "-Extension")
		rAdd.Params = append(rAdd.Params, addName)
		rAdd.Params = append(rAdd.Params, "-AllowConfigurationChanges")
		rAdd.Params = append(rAdd.Params, "-ChangesAllowedRule")
		rAdd.Params = append(rAdd.Params, "ObjectIsEditableSupportEnabled")
		rAdd.Params = append(rAdd.Params, "-ChangesNotRecommendedRule")
		rAdd.Params = append(rAdd.Params, "ObjectIsEditableSupportEnabled")
		rAdd.Params = append(rAdd.Params, "/Out")
		rAdd.Params = append(rAdd.Params, "/c Создание хранилища конфигурации")

		_, err := rAdd.RunCommand(context.Background(), l)
		if err != nil {
			l.Error("Ошибка создания хранилища конфигурации",
				slog.String("Путь", addStore.Path),
				slog.String("Строка подключения", dbConnectString),
				slog.String("Error", err.Error()),
			)
			return fmt.Errorf("ошибка создания хранилища конфигурации")
		}
		if !strings.Contains(string(rAdd.FileOut), constants.SearchMsgStoreCreateOk) {
			l.Error("Неопознанная ошибка создания хранилища конфигурации",
				slog.String("Путь", addStore.Path),
				slog.String("Строка подключения", dbConnectString),
				slog.String("Вывод в файл", runner.TrimOut(rAdd.FileOut)),
			)
			return fmt.Errorf("ошибка создания хранилища конфигурации")
		}

		l.Debug("Хранилище расширения успешно создано",
			slog.String("path", addStore.Path),
			slog.String("extensionName", addName),
		)
	}

	l.Debug("Все хранилища успешно созданы",
		slog.String("storeRoot", storeRoot),
		slog.Int("extensionsCount", len(arrayAdd)),
	)

	return nil
}
