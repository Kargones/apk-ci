//go:build wireinject

package di

import (
	"github.com/google/wire"

	"github.com/Kargones/apk-ci/internal/config"
)

//go:generate wire

// ProviderSet объединяет все провайдеры приложения.
// Используется в InitializeApp для построения графа зависимостей.
//
// При добавлении новых провайдеров:
// 1. Создать функцию провайдера в providers.go
// 2. Добавить её в ProviderSet
// 3. Перегенерировать: go generate ./internal/di/...
var ProviderSet = wire.NewSet(
	ProvideLogger,
	ProvideOutputWriter,
	ProvideTraceID,
	ProvideFactory,
	ProvideAlerter,
	ProvideMetricsCollector,
	ProvideTracerProvider,
	wire.Struct(new(App), "*"),
)

// InitializeApp создаёт и инициализирует App через Wire DI.
// Принимает внешний Config (загруженный через config.MustLoad()).
//
// Wire генерирует реализацию этой функции в wire_gen.go.
// Функция здесь является "заглушкой" с wire.Build() вызовом.
//
// Пример использования:
//
//	cfg, err := config.MustLoad()
//	if err != nil {
//	    log.Fatal(err)
//	}
//	app, err := di.InitializeApp(cfg)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// Использование: app.Logger, app.OutputWriter, app.TraceID
//
// Циклические зависимости между провайдерами обнаруживаются на этапе
// компиляции при генерации wire_gen.go (AC4).
// Проверка nil Config выполняется в runtime в сгенерированном InitializeApp (AC3).
func InitializeApp(cfg *config.Config) (*App, error) {
	wire.Build(ProviderSet)
	return nil, nil // Wire заменит это на реальную реализацию
}
