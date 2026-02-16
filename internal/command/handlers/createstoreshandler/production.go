// Package createstoreshandler содержит production реализации для работы с 1C.
//
// TODO(M-1): Рефакторить на struct pattern (defaultStoreCreator, defaultTempDbCreator)
// для соответствия паттерну других handlers (storebindhandler.defaultConvertLoader).
package createstoreshandler

import (
	"context"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/app"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/one/store"
)

// createTempDbProduction — production реализация создания временной БД.
func createTempDbProduction(ctx *context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	return app.CreateTempDbWrapper(ctx, l, cfg)
}

// createStoresProduction — production реализация создания хранилищ.
func createStoresProduction(l *slog.Logger, cfg *config.Config, storeRoot string, dbConnectString string, arrayAdd []string) error {
	return store.CreateStores(l, cfg, storeRoot, dbConnectString, arrayAdd)
}
