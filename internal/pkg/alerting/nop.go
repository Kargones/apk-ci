package alerting

import "context"

// NopAlerter — реализация Alerter, которая ничего не делает.
// Используется когда alerting отключён (enabled=false).
type NopAlerter struct{}

// NewNopAlerter создаёт Alerter, который игнорирует все алерты.
// Используется когда alerting.enabled=false или для тестов.
func NewNopAlerter() Alerter {
	return &NopAlerter{}
}

// Send ничего не делает, возвращает nil.
func (n *NopAlerter) Send(_ context.Context, _ Alert) error {
	return nil
}
