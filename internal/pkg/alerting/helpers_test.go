package alerting

import (
	"sync"

	"github.com/Kargones/apk-ci/internal/pkg/logging"
)

// testLogger реализует logging.Logger для тестирования.
// Thread-safe через sync.Mutex для использования в concurrent тестах (M-2/Review #8).
type testLogger struct {
	mu        sync.Mutex
	debugMsgs []string
	infoMsgs  []string
	warnMsgs  []string
	errorMsgs []string
}

func (l *testLogger) Debug(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.debugMsgs = append(l.debugMsgs, msg)
}
func (l *testLogger) Info(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.infoMsgs = append(l.infoMsgs, msg)
}
func (l *testLogger) Warn(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnMsgs = append(l.warnMsgs, msg)
}
func (l *testLogger) Error(msg string, args ...any) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.errorMsgs = append(l.errorMsgs, msg)
}
func (l *testLogger) With(_ ...any) logging.Logger { return l }
