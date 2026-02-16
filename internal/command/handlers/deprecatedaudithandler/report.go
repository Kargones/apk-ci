package deprecatedaudithandler

import "fmt"

// DeprecatedAliasInfo — информация о deprecated alias.
type DeprecatedAliasInfo struct {
	DeprecatedName string `json:"deprecated_name"`
	NRName         string `json:"nr_name"`
	HandlerPackage string `json:"handler_package"`
}

// TodoInfo — информация о TODO/deprecated комментарии.
type TodoInfo struct {
	File string `json:"file"`
	Line int    `json:"line"`
	Text string `json:"text"`
	Tag  string `json:"tag"`
}

// LegacyCaseInfo — информация о legacy case в switch.
type LegacyCaseInfo struct {
	File      string `json:"file"`
	Line      int    `json:"line"`
	CaseValue string `json:"case_value"`
	Note      string `json:"note,omitempty"`
}

// AuditReport — полный отчёт аудита deprecated кода.
type AuditReport struct {
	DeprecatedAliases []DeprecatedAliasInfo `json:"deprecated_aliases"`
	TodoComments      []TodoInfo            `json:"todo_comments"`
	LegacyCases       []LegacyCaseInfo      `json:"legacy_cases"`
	Summary           AuditSummary          `json:"summary"`
}

// AuditSummary — сводная статистика аудита.
type AuditSummary struct {
	TotalDeprecatedAliases int    `json:"total_deprecated_aliases"`
	TotalTodoH7            int    `json:"total_todo_h7"`
	TotalLegacyCases       int    `json:"total_legacy_cases"`
	ReadyForRemoval        string `json:"ready_for_removal"`
	Message                string `json:"message"`
}

// buildAuditReport формирует полный отчёт аудита с summary.
func buildAuditReport(aliases []DeprecatedAliasInfo, todos []TodoInfo, legacyCases []LegacyCaseInfo) *AuditReport {
	// Считаем только TODO(H-7) для summary
	todoH7Count := 0
	for _, t := range todos {
		if t.Tag == "H-7" {
			todoH7Count++
		}
	}

	readyForRemoval := "yes"
	if len(aliases) > 0 || len(legacyCases) > 0 || todoH7Count > 0 {
		readyForRemoval = "no"
		// partial: есть deprecated aliases И TODO, но нет legacy case-веток в switch
		// (legacy code уже вынесен в registry, осталось только удалить aliases и TODO)
		if len(aliases) > 0 && todoH7Count > 0 && len(legacyCases) == 0 {
			readyForRemoval = "partial"
		}
	}

	message := fmt.Sprintf("%d deprecated aliases, %d TODO(H-7) комментариев, %d legacy case-веток",
		len(aliases), todoH7Count, len(legacyCases))

	return &AuditReport{
		DeprecatedAliases: aliases,
		TodoComments:      todos,
		LegacyCases:       legacyCases,
		Summary: AuditSummary{
			TotalDeprecatedAliases: len(aliases),
			TotalTodoH7:            todoH7Count,
			TotalLegacyCases:       len(legacyCases),
			ReadyForRemoval:        readyForRemoval,
			Message:                message,
		},
	}
}
