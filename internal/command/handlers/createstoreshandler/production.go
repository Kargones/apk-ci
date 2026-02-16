// Package createstoreshandler содержит production реализации для работы с 1C.
//
// TODO(M-1): Рефакторить на struct pattern (defaultStoreCreator, defaultTempDbCreator)
// для соответствия паттерну других handlers (storebindhandler.defaultConvertLoader).
package createstoreshandler

import (
	"context"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/one/designer"
	"github.com/Kargones/apk-ci/internal/entity/one/store"
)

// createTempDbProduction — production реализация создания временной БД.
func createTempDbProduction(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	l.Debug("Запуск создания временной базы данных")
	dbPath := filepath.Join(cfg.TmpDir, "temp_db_"+time.Now().Format("20060102_150405"))
	oneDb, err := designer.CreateTempDb(ctx, l, cfg, dbPath, cfg.AddArray)
	if err != nil {
		l.Error("Ошибка создания временной базы данных", slog.String("Описание ошибки", err.Error()))
		return "", err
	}
	l.Debug("Временная база данных создана успешно",
		slog.String("Строка соединения", oneDb.DbConnectString),
		slog.String("Полная строка соединения", oneDb.FullConnectString),
		slog.Bool("Существует", oneDb.DbExist),
	)
	return oneDb.FullConnectString, nil
}

// createStoresProduction — production реализация создания хранилищ.
func createStoresProduction(l *slog.Logger, cfg *config.Config, storeRoot string, dbConnectString string, arrayAdd []string) error {
	return store.CreateStores(l, cfg, storeRoot, dbConnectString, arrayAdd)
}
