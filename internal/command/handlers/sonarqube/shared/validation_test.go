// Package shared содержит общую логику для SonarQube команд.
package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestIsValidBranchForScanning проверяет валидацию веток для сканирования.
// Принимаемые ветки: "main" или "t" + 6-7 цифр.
func TestIsValidBranchForScanning(t *testing.T) {
	tests := []struct {
		name     string
		branch   string
		expected bool
	}{
		// Валидные ветки
		{"main branch", "main", true},
		{"task branch 6 digits", "t123456", true},
		{"task branch 7 digits", "t1234567", true},
		{"task branch min value", "t000000", true},
		{"task branch max 6 digits", "t999999", true},
		{"task branch max 7 digits", "t9999999", true},

		// Невалидные ветки
		{"empty branch", "", false},
		{"master branch", "master", false},
		{"develop branch", "develop", false},
		{"feature branch", "feature/login", false},
		{"task branch 5 digits", "t12345", false},
		{"task branch 8 digits", "t12345678", false},
		{"task branch with letters", "t12345a", false},
		{"task branch uppercase T", "T123456", false},
		{"task branch with hyphen", "t123-456", false},
		{"task branch with underscore", "t123_456", false},
		{"just t", "t", false},
		{"t with spaces", "t 123456", false},
		{"release branch", "release/1.0", false},
		{"hotfix branch", "hotfix/bug", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidBranchForScanning(tt.branch)
			assert.Equal(t, tt.expected, result, "branch: %s", tt.branch)
		})
	}
}

// TestIsValidBranchForScanning_EdgeCases проверяет граничные случаи.
func TestIsValidBranchForScanning_EdgeCases(t *testing.T) {
	// Граница между 5 и 6 цифрами
	assert.False(t, IsValidBranchForScanning("t99999"), "5 digits should be invalid")
	assert.True(t, IsValidBranchForScanning("t100000"), "6 digits should be valid")

	// Граница между 7 и 8 цифрами
	assert.True(t, IsValidBranchForScanning("t9999999"), "7 digits should be valid")
	assert.False(t, IsValidBranchForScanning("t10000000"), "8 digits should be invalid")

	// Unicode символы
	assert.False(t, IsValidBranchForScanning("t12345б"), "cyrillic should be invalid")
	assert.False(t, IsValidBranchForScanning("t١٢٣٤٥٦"), "arabic numerals should be invalid")
}
