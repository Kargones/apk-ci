// Package smoketest содержит smoke-тесты системной целостности apk-ci.
//
// Smoke-тесты проверяют:
//   - Регистрацию всех NR-команд в глобальном реестре
//   - Корректность deprecated aliases (DeprecatedBridge)
//   - Валидность Name() и Description() каждого handler
//   - Уникальность и детерминированность списка команд
//
// Это НЕ unit-тесты отдельных handlers — это тесты уровня системной целостности.
// Unit-тесты бизнес-логики находятся в handler_test.go каждого handler-пакета.
//
// Запуск: make test-smoke
package smoketest
