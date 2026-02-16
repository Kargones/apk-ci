package shadowrun

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
)

// Runner выполняет shadow-run: запускает NR-команду и legacy-версию последовательно,
// сравнивает результаты и формирует отчёт о расхождениях.
type Runner struct {
	mapping *LegacyMapping
	logger  *slog.Logger
}

// NewRunner создаёт новый Runner с заданным маппингом legacy-функций.
func NewRunner(mapping *LegacyMapping, logger *slog.Logger) *Runner {
	return &Runner{
		mapping: mapping,
		logger:  logger,
	}
}

// Execute выполняет shadow-run для NR-команды.
// Возвращает:
//   - ShadowRunResult с результатами сравнения
//   - nrOutput — захваченный stdout NR-команды (для повторного вывода)
//   - error от NR-команды (основной результат; legacy-ошибки логируются, но не возвращаются)
func (r *Runner) Execute(ctx context.Context, cfg *config.Config, handler command.Handler) (*ShadowRunResult, string, error) {
	cmdName := handler.Name()

	// Проверяем наличие legacy-маппинга
	legacyFn, hasLegacy := r.mapping.Get(cmdName)
	if !hasLegacy {
		r.logger.Warn("Shadow-run: legacy-версия не найдена, выполняется только NR",
			slog.String("command", cmdName),
		)

		// Выполняем только NR-команду
		nrStart := time.Now()
		nrOutput, nrErr := r.captureNRExecution(ctx, cfg, handler)
		nrDuration := time.Since(nrStart)

		return &ShadowRunResult{
			Enabled:    true,
			Match:      true,
			NRDuration: nrDuration,
			Warning:    fmt.Sprintf("legacy-версия для '%s' не найдена, shadow-run выполнил только NR", cmdName),
			NRError:    errString(nrErr),
		}, nrOutput, nrErr
	}

	// Предупреждение для state-changing команд (enable, disable, dbrestore и т.д.)
	if r.mapping.IsStateChanging(cmdName) {
		r.logger.Warn("Shadow-run: команда изменяет состояние, legacy-версия может вызвать побочные эффекты",
			slog.String("command", cmdName),
		)
	}

	// Выполняем NR-команду с захватом stdout
	r.logger.Info("Shadow-run: выполнение NR-версии",
		slog.String("command", cmdName),
	)
	nrStart := time.Now()
	nrOutput, nrErr := r.captureNRExecution(ctx, cfg, handler)
	nrDuration := time.Since(nrStart)

	r.logger.Info("Shadow-run: NR выполнена",
		slog.String("command", cmdName),
		slog.Duration("duration", nrDuration),
		slog.Bool("has_error", nrErr != nil),
	)

	// Выполняем legacy-команду с захватом stdout
	r.logger.Info("Shadow-run: выполнение legacy-версии",
		slog.String("command", cmdName),
	)
	legacyStart := time.Now()
	legacyOutput, legacyErr := r.captureLegacyExecution(ctx, cfg, legacyFn)
	legacyDuration := time.Since(legacyStart)

	r.logger.Info("Shadow-run: legacy выполнена",
		slog.String("command", cmdName),
		slog.Duration("duration", legacyDuration),
		slog.Bool("has_error", legacyErr != nil),
	)

	// Сравниваем результаты
	comparison := CompareResults(nrErr, legacyErr, nrOutput, legacyOutput)

	result := &ShadowRunResult{
		Enabled:        true,
		Match:          comparison.Match,
		NRDuration:     nrDuration,
		LegacyDuration: legacyDuration,
		Differences:    comparison.Differences,
		NRError:        errString(nrErr),
		LegacyError:    errString(legacyErr),
	}

	if !comparison.Match {
		r.logger.Warn("Shadow-run: обнаружены различия",
			slog.String("command", cmdName),
			slog.Int("diff_count", len(comparison.Differences)),
		)
	} else {
		r.logger.Info("Shadow-run: результаты идентичны",
			slog.String("command", cmdName),
		)
	}

	// Возвращаем ошибку NR (основной результат)
	return result, nrOutput, nrErr
}

// captureStdoutMu защищает подмену os.Stdout от concurrent access.
// TODO: Глубокий рефакторинг — научить handlers принимать io.Writer вместо прямого os.Stdout.
// Текущее решение с mutex защищает от одновременных подмен, но не от записи в os.Stdout
// из других горутин (slog, OTel). На практике slog пишет в stderr, поэтому риск минимален.
var captureStdoutMu sync.Mutex

// captureExecution выполняет функцию fn, перехватывая stdout через os.Pipe.
// Mutex защищает подмену os.Stdout от concurrent access.
// При panic внутри fn() stdout и pipe гарантированно восстанавливаются через defer.
func captureExecution(logger *slog.Logger, fn func() error) (output string, execErr error) {
	captureStdoutMu.Lock()
	defer captureStdoutMu.Unlock()

	origStdout := os.Stdout
	readPipe, writePipe, err := os.Pipe()
	if err != nil {
		logger.Warn("Shadow-run: не удалось создать pipe для захвата stdout",
			slog.String("error", err.Error()),
		)
		execErr = fn()
		return "", execErr
	}

	os.Stdout = writePipe

	var buf bytes.Buffer
	var copyErr error
	done := make(chan struct{})
	go func() {
		_, copyErr = io.Copy(&buf, readPipe)
		close(done)
	}()

	// defer гарантирует восстановление stdout и закрытие pipe при panic в fn().
	// Review #30: <-done НЕ может заблокировать: writePipe.Close() вызывается перед ним,
	// что приводит к EOF в io.Copy → goroutine завершается → close(done) → <-done разблокируется.
	defer func() {
		os.Stdout = origStdout
		_ = writePipe.Close()
		<-done
		_ = readPipe.Close()
		if copyErr != nil {
			logger.Warn("Shadow-run: ошибка при чтении захваченного stdout",
				slog.String("error", copyErr.Error()),
			)
		}
		output = buf.String()
	}()

	execErr = fn()
	return
}

// captureNRExecution выполняет NR-команду, перехватывая stdout.
func (r *Runner) captureNRExecution(ctx context.Context, cfg *config.Config, handler command.Handler) (string, error) {
	return captureExecution(r.logger, func() error {
		return handler.Execute(ctx, cfg)
	})
}

// captureLegacyExecution выполняет legacy-функцию, перехватывая stdout.
// Legacy-функции принимают *context.Context — создаём отдельный context.
func (r *Runner) captureLegacyExecution(ctx context.Context, cfg *config.Config, legacyFn LegacyFunc) (string, error) {
	return captureExecution(r.logger, func() error {
		legacyCtx := ctx
		return legacyFn(&legacyCtx, r.logger, cfg)
	})
}

// errString конвертирует error в строку; для nil возвращает пустую строку.
func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}
