// Package testutil содержит общие утилиты для тестирования.
package testutil

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

// CaptureStdout выполняет fn, перехватывая stdout, и возвращает вывод.
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err, "не удалось создать pipe для stdout")

	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	fn()

	_ = w.Close() //nolint:errcheck // test helper pipe close

	var buf bytes.Buffer
	_, err = buf.ReadFrom(r)
	require.NoError(t, err, "не удалось прочитать stdout")
	return buf.String()
}
