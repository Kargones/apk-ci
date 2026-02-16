// Package templateprocessor содержит утилиты для обработки шаблонов
package templateprocessor

import (
	"strings"
)

// ReplacementRule определяет правило замены строк
type ReplacementRule struct {
	SearchString      string // Строка, которую необходимо найти
	ReplacementString string // Строка, на которую будет заменена SearchString
}

// TemplateResult содержит результат обработки шаблона
type TemplateResult struct {
	FileName string // Имя файла из первой строки шаблона
	Result   string // Обработанное содержимое без первой строки
}

// ProcessMultipleTemplates разделяет входной текст по разделителю "---" и обрабатывает каждый фрагмент
// через ProcessWorkflowTemplate. Возвращает массив TemplateResult или ошибку.
func ProcessMultipleTemplates(text string, replacementRules []ReplacementRule) ([]TemplateResult, error) {
	// Разделяем текст по разделителю "---"
	fragments := strings.Split(text, "---")
	
	// Pre-allocate slice with expected capacity
	results := make([]TemplateResult, 0, len(fragments))
	
	// Обрабатываем каждый фрагмент
	for _, fragment := range fragments {
		// Убираем лишние пробелы и переносы строк в начале и конце фрагмента
		fragment = strings.TrimSpace(fragment)
		
		// Пропускаем пустые фрагменты
		if fragment == "" {
			continue
		}
		
		// Обрабатываем фрагмент через ProcessWorkflowTemplate
		result := ProcessWorkflowTemplate(fragment, replacementRules)
		results = append(results, result)
	}
	
	return results, nil
}

// ProcessWorkflowTemplate обрабатывает YAML шаблон, заменяя плейсхолдеры согласно правилам замены
// Извлекает первую строку как fileName и возвращает остальное содержимое как result
func ProcessWorkflowTemplate(template string, replacementRules []ReplacementRule) TemplateResult {
	// Разделяем шаблон на строки
	lines := strings.Split(template, "\n")
	if len(lines) == 0 {
		return TemplateResult{FileName: "", Result: ""}
	}
	
	// Извлекаем первую строку как fileName
	fileName := strings.TrimSpace(lines[0])
	
	// Объединяем остальные строки как содержимое для обработки
	var content string
	if len(lines) > 1 {
		content = strings.Join(lines[1:], "\n")
	}
	
	if len(replacementRules) == 0 {
		return TemplateResult{FileName: fileName, Result: content}
	}

	// Выполняем замены согласно правилам
	result := content
	for _, rule := range replacementRules {
		result = strings.ReplaceAll(result, rule.SearchString, rule.ReplacementString)
	}

	return TemplateResult{
		FileName: fileName,
		Result:   result,
	}
}
