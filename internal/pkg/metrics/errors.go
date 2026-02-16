package metrics

import "errors"

var (
	// ErrPushgatewayURLRequired возвращается если не указан URL Pushgateway при включённых метриках.
	ErrPushgatewayURLRequired = errors.New("pushgateway URL is required when metrics enabled")

	// ErrJobNameRequired возвращается если не указано имя job.
	ErrJobNameRequired = errors.New("job name is required")

	// ErrInvalidTimeout возвращается если указан невалидный таймаут.
	ErrInvalidTimeout = errors.New("timeout must be positive")

	// ErrPushgatewayURLInvalid возвращается если URL Pushgateway имеет невалидный формат.
	ErrPushgatewayURLInvalid = errors.New("pushgateway URL has invalid format")
)
