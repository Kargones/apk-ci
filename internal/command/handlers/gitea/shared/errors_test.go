// Package shared содержит общую логику для Gitea команд.
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
	assert.NotEmpty(t, ErrBranchCreate, "ErrBranchCreate should not be empty")
	assert.NotEmpty(t, ErrNoDatabases, "ErrNoDatabases should not be empty")
	assert.NotEmpty(t, ErrTemplateProcess, "ErrTemplateProcess should not be empty")
	assert.NotEmpty(t, ErrSyncFailed, "ErrSyncFailed should not be empty")

	// Проверяем формат констант (NAMESPACE.ERROR_TYPE)
	assert.Contains(t, ErrConfigMissing, ".", "ErrConfigMissing should contain namespace separator")
	assert.Contains(t, ErrMissingOwnerRepo, ".", "ErrMissingOwnerRepo should contain namespace separator")
	assert.Contains(t, ErrGiteaAPI, ".", "ErrGiteaAPI should contain namespace separator")
	assert.Contains(t, ErrBranchCreate, ".", "ErrBranchCreate should contain namespace separator")
	assert.Contains(t, ErrNoDatabases, ".", "ErrNoDatabases should contain namespace separator")
	assert.Contains(t, ErrTemplateProcess, ".", "ErrTemplateProcess should contain namespace separator")
	assert.Contains(t, ErrSyncFailed, ".", "ErrSyncFailed should contain namespace separator")
}

// TestErrorConstants_Namespaces проверяет правильные namespaces для ошибок.
func TestErrorConstants_Namespaces(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		prefix   string
	}{
		// Общие константы (реэкспорт из commonerrors)
		{"config missing", ErrConfigMissing, "CONFIG."},
		{"missing owner repo", ErrMissingOwnerRepo, "CONFIG."},
		{"gitea api", ErrGiteaAPI, "GITEA."},
		// Специфичные для Gitea команд
		{"branch create", ErrBranchCreate, "GITEA."},
		{"no databases", ErrNoDatabases, "CONFIG."},
		{"template process", ErrTemplateProcess, "TEMPLATE."},
		{"sync failed", ErrSyncFailed, "SYNC."},
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
		ErrBranchCreate,
		ErrNoDatabases,
		ErrTemplateProcess,
		ErrSyncFailed,
	}

	seen := make(map[string]bool)
	for _, c := range constants {
		assert.False(t, seen[c], "duplicate error constant: %s", c)
		seen[c] = true
	}
}

// TestErrorConstants_Values проверяет конкретные значения констант.
func TestErrorConstants_Values(t *testing.T) {
	// Специфичные для Gitea команд — проверяем точные значения
	assert.Equal(t, "GITEA.BRANCH_CREATE_FAILED", ErrBranchCreate)
	assert.Equal(t, "CONFIG.NO_DATABASES", ErrNoDatabases)
	assert.Equal(t, "TEMPLATE.PROCESS_FAILED", ErrTemplateProcess)
	assert.Equal(t, "SYNC.FAILED", ErrSyncFailed)

	// Общие константы — должны совпадать с commonerrors
	assert.Equal(t, "CONFIG.MISSING", ErrConfigMissing)
	assert.Equal(t, "CONFIG.MISSING_OWNER_REPO", ErrMissingOwnerRepo)
	assert.Equal(t, "GITEA.API_FAILED", ErrGiteaAPI)
}
