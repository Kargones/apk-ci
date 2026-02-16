package version

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
	"github.com/Kargones/apk-ci/internal/pkg/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVersionHandler_Integration_RegistryAndLegacy проверяет что nr-version
// регистрируется через Registry и корректно работает. Legacy switch в main.go
// не должен обрабатывать nr-version — Registry перехватывает.
func TestVersionHandler_Integration_RegistryAndLegacy(t *testing.T) {
	// Проверяем что handler зарегистрирован
	handler, ok := command.Get(constants.ActNRVersion)
	require.True(t, ok, "nr-version должен быть зарегистрирован в Registry")

	// Проверяем что это именно VersionHandler
	_, isVersionHandler := handler.(*VersionHandler)
	assert.True(t, isVersionHandler, "handler должен быть типа *VersionHandler")

	// Выполняем через Registry (как это делает main.go)
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = handler.Execute(context.Background(), nil)
	})

	require.NoError(t, execErr)

	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err)
	assert.Equal(t, output.StatusSuccess, result.Status)
	assert.Equal(t, constants.ActNRVersion, result.Command)
}

// TestVersionHandler_Integration_StdoutStderrSeparation проверяет что
// результат идёт в stdout, а логи — в stderr. Для этого перехватываем
// stdout и убеждаемся что он содержит только JSON.
func TestVersionHandler_Integration_StdoutStderrSeparation(t *testing.T) {
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	h := &VersionHandler{}

	var execErr error
	out := testutil.CaptureStdout(t, func() {
		execErr = h.Execute(context.Background(), nil)
	})

	require.NoError(t, execErr)
	require.NotEmpty(t, out, "stdout не должен быть пустым")

	// stdout должен содержать ТОЛЬКО валидный JSON результат
	var result output.Result
	err := json.Unmarshal([]byte(out), &result)
	require.NoError(t, err, "stdout должен содержать валидный JSON Result")

	assert.Equal(t, output.StatusSuccess, result.Status)
	assert.Equal(t, constants.ActNRVersion, result.Command)
	assert.NotNil(t, result.Data)
	assert.NotNil(t, result.Metadata)
	assert.NotEmpty(t, result.Metadata.TraceID)
}
