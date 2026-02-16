// Package shared содержит общие компоненты для всех command handlers.
package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestErrorConstants проверяет что константы ошибок определены корректно.
func TestErrorConstants(t *testing.T) {
	// Проверяем что константы не пустые
	assert.NotEmpty(t, ErrConfigMissing, "ErrConfigMissing should not be empty")
	assert.NotEmpty(t, ErrMissingOwnerRepo, "ErrMissingOwnerRepo should not be empty")
	assert.NotEmpty(t, ErrGiteaAPI, "ErrGiteaAPI should not be empty")
	assert.NotEmpty(t, ErrSonarQubeAPI, "ErrSonarQubeAPI should not be empty")

	// Проверяем формат констант (NAMESPACE.ERROR_TYPE)
	assert.Contains(t, ErrConfigMissing, ".", "ErrConfigMissing should contain namespace separator")
	assert.Contains(t, ErrMissingOwnerRepo, ".", "ErrMissingOwnerRepo should contain namespace separator")
	assert.Contains(t, ErrGiteaAPI, ".", "ErrGiteaAPI should contain namespace separator")
	assert.Contains(t, ErrSonarQubeAPI, ".", "ErrSonarQubeAPI should contain namespace separator")
}

// TestErrorConstants_Namespaces проверяет правильные namespaces для ошибок.
func TestErrorConstants_Namespaces(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		prefix   string
	}{
		{"config missing", ErrConfigMissing, "CONFIG."},
		{"missing owner repo", ErrMissingOwnerRepo, "CONFIG."},
		{"gitea api", ErrGiteaAPI, "GITEA."},
		{"sonarqube api", ErrSonarQubeAPI, "SONARQUBE."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.True(t, len(tt.constant) > len(tt.prefix), "constant should be longer than prefix")
			assert.Equal(t, tt.prefix, tt.constant[:len(tt.prefix)], "constant should start with correct namespace")
		})
	}
}

// TestErrorConstants_Uniqueness проверяет уникальность констант.
func TestErrorConstants_Uniqueness(t *testing.T) {
	constants := []string{
		ErrConfigMissing,
		ErrMissingOwnerRepo,
		ErrGiteaAPI,
		ErrSonarQubeAPI,
	}

	seen := make(map[string]bool)
	for _, c := range constants {
		assert.False(t, seen[c], "duplicate error constant: %s", c)
		seen[c] = true
	}
}
