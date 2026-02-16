package output

import (
	"io"
	"time"
)

// WritePlanOnlyResult пишет результат plan-only режима (план без выполнения).
// Извлечено из handler'ов для устранения дублирования (CR-7.3 #2).
// Story 7.3 AC-6: JSON вывод содержит plan_only: true и plan: {...}.
func WritePlanOnlyResult(w io.Writer, format, command, traceID, apiVersion string, start time.Time, plan *DryRunPlan) error {
	return writePlanResult(w, format, command, traceID, apiVersion, start, plan, true)
}

// WriteDryRunResult пишет результат dry-run режима (план без выполнения).
// Извлечено из handler'ов для устранения дублирования (CR-7.3 #3).
func WriteDryRunResult(w io.Writer, format, command, traceID, apiVersion string, start time.Time, plan *DryRunPlan) error {
	return writePlanResult(w, format, command, traceID, apiVersion, start, plan, false)
}

// writePlanResult — общая реализация для dry-run и plan-only.
func writePlanResult(w io.Writer, format, command, traceID, apiVersion string, start time.Time, plan *DryRunPlan, planOnly bool) error {
	duration := time.Since(start)

	if format != FormatJSON {
		if planOnly {
			return plan.WritePlanText(w)
		}
		return plan.WriteText(w)
	}

	result := &Result{
		Status:  StatusSuccess,
		Command: command,
		Plan:    plan,
		Metadata: &Metadata{
			DurationMs: duration.Milliseconds(),
			TraceID:    traceID,
			APIVersion: apiVersion,
		},
	}

	if planOnly {
		result.PlanOnly = true
	} else {
		result.DryRun = true
	}

	writer := NewWriter(format)
	return writer.Write(w, result)
}
