// Package metrics предоставляет интерфейсы и реализации для сбора и отправки метрик
// в Prometheus Pushgateway.
//
// Пакет следует паттернам проекта apk-ci:
//   - Interface Segregation: Collector interface для абстракции
//   - Factory pattern: NewCollector выбирает реализацию на основе конфигурации
//   - Graceful degradation: NopCollector при отключённых метриках
package metrics

import (
	"context"
	"time"
)

// Collector определяет интерфейс для сбора метрик.
// Реализации: PrometheusCollector (активный) и NopCollector (no-op).
type Collector interface {
	// RecordCommandStart записывает начало выполнения команды.
	// Для CLI не требуется отслеживать "in-flight" — метод может быть no-op.
	RecordCommandStart(command, infobase string)

	// RecordCommandEnd записывает завершение команды с результатом.
	// duration — время выполнения команды.
	// success — успешно ли завершилась команда.
	RecordCommandEnd(command, infobase string, duration time.Duration, success bool)

	// Push отправляет метрики в Pushgateway.
	// Возвращает nil даже при ошибке — ошибки логируются внутри реализации.
	// Сигнатура `error` сохранена для совместимости с интерфейсом, но все реализации
	// (PrometheusCollector, NopCollector) всегда возвращают nil (AC8).
	// M-2/Review #10: явная документация паттерна "всегда nil".
	Push(ctx context.Context) error
}
