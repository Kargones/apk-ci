package metrics

import (
	"context"
	"time"
)

// NopCollector — no-op реализация Collector.
// Используется когда метрики отключены (Config.Enabled = false).
type NopCollector struct{}

// NewNopCollector создаёт NopCollector.
func NewNopCollector() *NopCollector {
	return &NopCollector{}
}

// RecordCommandStart — no-op, ничего не делает.
func (c *NopCollector) RecordCommandStart(command, infobase string) {}

// RecordCommandEnd — no-op, ничего не делает.
func (c *NopCollector) RecordCommandEnd(command, infobase string, duration time.Duration, success bool) {
}

// Push — no-op, всегда возвращает nil.
func (c *NopCollector) Push(ctx context.Context) error {
	return nil
}
