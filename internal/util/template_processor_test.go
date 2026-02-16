package templateprocessor

import (
	"strings"
	"testing"
)

// getTestTemplate возвращает тестовый YAML шаблон, имитирующий содержимое MenuMain
func getTestTemplate() string {
	return `on:
  workflow_dispatch:
    inputs:
      restore_DB:
        description: 'Восстановление базы из бекапа продуктивного контура [TEST]'
        required: false
        default: 'false'
        type: choice
        options:
          - 'false'
          - 'true'
      service_mode_enable:
        description: 'Включить сервисный режим'
        required: false
        default: 'false'
        type: choice
        options:
          - 'false'
          - 'true'
      load_cfg:
        description: 'Загрузка конфигурации из хранилища [TEST]'
        required: false
        default: 'false'
        type: choice
        options:
          - 'false'
          - 'true'
      DbName:
        description: 'Имя базы данных'
        required: true
        default: '$TestBaseReplace$'
        type: choice
        options:$TestBaseReplaceAll$
      update_conf:
        description: 'Применение обновленной конфигурации [TEST]'
        required: false
        default: 'false'
        type: choice
        options:
          - 'false'
          - 'true'
jobs:
  db-update-test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3`
}

func TestProcessWorkflowTemplate(t *testing.T) {
	tests := []struct {
		name             string
		template         string
		replacementRules []ReplacementRule
		expectedFile     string
		want             struct {
			containsDefault    string
			containsOptions    []string
			containsUpdateConf bool
		}
	}{
		{
			name:             "empty replacement rules",
			template:         "test-workflow.yaml\n" + getTestTemplate(),
			replacementRules: []ReplacementRule{},
			expectedFile:     "test-workflow.yaml",
			want: struct {
				containsDefault    string
				containsOptions    []string
				containsUpdateConf bool
			}{
				containsDefault:    "$TestBaseReplace$",             // должно остаться без изменений
				containsOptions:    []string{"$TestBaseReplaceAll$"}, // должно остаться без изменений
				containsUpdateConf: true,
			},
		},
		{
			name:     "single replacement rule",
			template: "single-db.yaml\n" + getTestTemplate(),
			replacementRules: []ReplacementRule{
				{SearchString: "$TestBaseReplace$", ReplacementString: "V8_TEST_DB1"},
				{SearchString: "$TestBaseReplaceAll$", ReplacementString: "\n          - V8_TEST_DB1"},
			},
			expectedFile: "single-db.yaml",
			want: struct {
				containsDefault    string
				containsOptions    []string
				containsUpdateConf bool
			}{
				containsDefault:    "V8_TEST_DB1",
				containsOptions:    []string{"- V8_TEST_DB1"},
				containsUpdateConf: true,
			},
		},
		{
			name:     "multiple replacement rules",
			template: "multi-db.yaml\n" + getTestTemplate(),
			replacementRules: []ReplacementRule{
				{SearchString: "$TestBaseReplace$", ReplacementString: "V8_TEST_DB1"},
				{SearchString: "$TestBaseReplaceAll$", ReplacementString: "\n          - V8_TEST_DB1\n          - V8_TEST_DB2\n          - V8_TEST_DB3"},
			},
			expectedFile: "multi-db.yaml",
			want: struct {
				containsDefault    string
				containsOptions    []string
				containsUpdateConf bool
			}{
				containsDefault:    "V8_TEST_DB1", // первая база
				containsOptions:    []string{"- V8_TEST_DB1", "- V8_TEST_DB2", "- V8_TEST_DB3"},
				containsUpdateConf: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessWorkflowTemplate(tt.template, tt.replacementRules)

			// Проверяем имя файла
			if result.FileName != tt.expectedFile {
				t.Errorf("ProcessWorkflowTemplate().FileName = %v, want %v", result.FileName, tt.expectedFile)
			}

			// Проверяем замену default значения
			if !strings.Contains(result.Result, "default: '"+tt.want.containsDefault+"'") {
				t.Errorf("ProcessWorkflowTemplate() default value = %v, want %v", result.Result, tt.want.containsDefault)
			}

			// Проверяем замену опций
			for _, option := range tt.want.containsOptions {
				if !strings.Contains(result.Result, option) {
					t.Errorf("ProcessWorkflowTemplate() missing option = %v", option)
				}
			}

			// Проверяем наличие поля update_conf
			if tt.want.containsUpdateConf {
				if !strings.Contains(result.Result, "update_conf:") {
					t.Errorf("ProcessWorkflowTemplate() missing update_conf field")
				}
			}

			// Проверяем, что плейсхолдеры заменены (кроме случая с пустыми правилами)
			if len(tt.replacementRules) > 0 {
				if strings.Contains(result.Result, "$TestBaseReplace$") {
					t.Errorf("ProcessWorkflowTemplate() still contains $TestBaseReplace$ placeholder")
				}
				if strings.Contains(result.Result, "$TestBaseReplaceAll$") {
					t.Errorf("ProcessWorkflowTemplate() still contains $TestBaseReplaceAll$ placeholder")
				}
			}
		})
	}
}

func TestProcessMultipleTemplates(t *testing.T) {
	tests := []struct {
		name             string
		input            string
		replacementRules []ReplacementRule
		expected         int // количество ожидаемых результатов
	}{
		{
			name:             "single_template",
			input:            "workflow-single.yaml\non:\n  workflow_dispatch:",
			replacementRules: []ReplacementRule{{SearchString: "$TestBaseReplace$", ReplacementString: "V8_TEST_DB1"}},
			expected:         1,
		},
		{
			name:  "multiple_templates",
			input: "workflow-first.yaml\nfirst template\n---\nworkflow-second.yaml\nsecond template",
			replacementRules: []ReplacementRule{
				{SearchString: "$TestBaseReplace$", ReplacementString: "V8_TEST_DB1"},
				{SearchString: "$TestBaseReplaceAll$", ReplacementString: "\n          - V8_TEST_DB1\n          - V8_TEST_DB2"},
			},
			expected: 2,
		},
		{
			name:             "empty_fragments",
			input:            "workflow-test.yaml\ntemplate content\n---\n\n---\nworkflow-another.yaml\nanother content",
			replacementRules: []ReplacementRule{{SearchString: "$TestBaseReplace$", ReplacementString: "V8_TEST_DB1"}},
			expected:         2,
		},
		{
			name:             "no_separator",
			input:            "workflow-nosep.yaml\nsingle template without separator",
			replacementRules: []ReplacementRule{{SearchString: "$TestBaseReplace$", ReplacementString: "V8_TEST_DB1"}},
			expected:         1,
		},
		{
			name:             "empty_input",
			input:            "",
			replacementRules: []ReplacementRule{{SearchString: "TestBaseReplace1", ReplacementString: "V8_TEST_DB1"}},
			expected:         0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := ProcessMultipleTemplates(tt.input, tt.replacementRules)

			if err != nil {
				t.Errorf("ProcessMultipleTemplates() error = %v", err)
				return
			}

			if len(results) != tt.expected {
				t.Errorf("ProcessMultipleTemplates() got %d results, expected %d", len(results), tt.expected)
				return
			}

			// Проверяем, что каждый результат имеет корректную структуру
			for i, result := range results {
				if result.FileName == "" {
					t.Errorf("Result %d: FileName is empty", i)
				}
				if result.Result == "" && tt.input != "" {
					t.Errorf("Result %d: Result is empty for non-empty input", i)
				}
			}
		})
	}
}

// TestProcessWorkflowTemplateStructure проверяет структуру результирующего YAML
func TestProcessWorkflowTemplateStructure(t *testing.T) {
	replacementRules := []ReplacementRule{
				{SearchString: "$TestBaseReplace$", ReplacementString: "V8_TEST_DB1"},
				{SearchString: "$TestBaseReplaceAll$", ReplacementString: "\n          - V8_TEST_DB1\n          - V8_TEST_DB2"},
			}
	result := ProcessWorkflowTemplate("test-structure.yaml\n"+getTestTemplate(), replacementRules)

	// Проверяем имя файла
	if result.FileName != "test-structure.yaml" {
		t.Errorf("ProcessWorkflowTemplate().FileName = %v, want %v", result.FileName, "test-structure.yaml")
	}

	// Проверяем основные секции YAML
	requiredSections := []string{
		"on:",
		"workflow_dispatch:",
		"inputs:",
		"restore_DB:",
		"service_mode_enable:",
		"load_cfg:",
		"DbName:",
		"update_conf:",
		"jobs:",
		"db-update-test:",
		"steps:",
	}

	for _, section := range requiredSections {
		if !strings.Contains(result.Result, section) {
			t.Errorf("ProcessWorkflowTemplate() missing required section: %v", section)
		}
	}
}
