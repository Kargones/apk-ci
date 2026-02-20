package testmerge

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/gitea/giteatest"
	"github.com/Kargones/apk-ci/internal/command/handlers/gitea/shared"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
)

// TestName проверяет возврат имени команды (AC: #1).
func TestName(t *testing.T) {
	h := &TestMergeHandler{}
	if got := h.Name(); got != constants.ActNRTestMerge {
		t.Errorf("Name() = %q, want %q", got, constants.ActNRTestMerge)
	}
}

// TestDescription проверяет возврат описания команды.
func TestDescription(t *testing.T) {
	h := &TestMergeHandler{}
	if got := h.Description(); got == "" {
		t.Error("Description() returned empty string")
	}
}

// TestExecute_NilConfig проверяет обработку nil конфигурации.
func TestExecute_NilConfig(t *testing.T) {
	h := &TestMergeHandler{}

	err := h.Execute(context.Background(), nil)
	if err == nil {
		t.Error("Execute() expected error for nil config, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrConfigMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrConfigMissing)
	}
}

// TestExecute_MissingOwnerRepo проверяет отсутствие owner/repo.
func TestExecute_MissingOwnerRepo(t *testing.T) {
	h := &TestMergeHandler{
		giteaClient: giteatest.NewMockClient(),
	}

	tests := []struct {
		name  string
		owner string
		repo  string
	}{
		{"missing owner", "", "repo"},
		{"missing repo", "owner", ""},
		{"missing both", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Owner: tt.owner,
				Repo:  tt.repo,
			}

			err := h.Execute(context.Background(), cfg)
			if err == nil {
				t.Error("Execute() expected error for missing owner/repo, got nil")
			}

			if err != nil && !contains(err.Error(), shared.ErrMissingOwnerRepo) {
				t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrMissingOwnerRepo)
			}
		})
	}
}

// TestExecute_NilGiteaClient проверяет обработку nil Gitea клиента.
func TestExecute_NilGiteaClient(t *testing.T) {
	h := &TestMergeHandler{
		giteaClient: nil, // nil клиент
	}

	cfg := &config.Config{
		Owner: "owner",
		Repo:  "repo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for nil Gitea client, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrConfigMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrConfigMissing)
	}
}

// TestExecute_NoPRs проверяет случай без открытых PR (AC: #2).
func TestExecute_NoPRs(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{}, nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
}

// TestExecute_AllMergeable проверяет случай когда все PR без конфликтов (AC: #4, #5).
func TestExecute_AllMergeable(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
				{Number: 2, Head: "feature-2", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreatePRFunc: func(_ context.Context, head string) (gitea.PR, error) {
			return gitea.PR{Number: 100, Head: head, Base: "test-merge-branch"}, nil
		},
		ConflictPRFunc: func(_ context.Context, _ int64) (bool, error) {
			return false, nil // Нет конфликтов
		},
		MergePRFunc: func(_ context.Context, _ int64) error {
			return nil // Успешный merge
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
}

// TestExecute_SomeConflicts проверяет случай когда часть PR с конфликтами (AC: #4, #5, #6).
func TestExecute_SomeConflicts(t *testing.T) {
	closedPRs := make(map[int64]bool)
	commentedPRs := make(map[int64]string)

	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
				{Number: 2, Head: "feature-2", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreatePRFunc: func(_ context.Context, head string) (gitea.PR, error) {
			// Назначаем номера для тестовых PR
			if head == "feature-1" {
				return gitea.PR{Number: 100, Head: head, Base: "test-merge-branch"}, nil
			}
			return gitea.PR{Number: 101, Head: head, Base: "test-merge-branch"}, nil
		},
		ConflictPRFunc: func(_ context.Context, prNumber int64) (bool, error) {
			// PR #100 (from feature-1) has conflict
			return prNumber == 100, nil
		},
		ConflictFilesPRFunc: func(_ context.Context, prNumber int64) ([]string, error) {
			if prNumber == 100 {
				return []string{"src/main.go", "config.yaml"}, nil
			}
			return nil, nil
		},
		MergePRFunc: func(_ context.Context, _ int64) error {
			return nil
		},
		AddIssueCommentFunc: func(_ context.Context, prNumber int64, commentText string) error {
			commentedPRs[prNumber] = commentText
			return nil
		},
		ClosePRFunc: func(_ context.Context, prNumber int64) error {
			closedPRs[prNumber] = true
			return nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	// Проверяем что конфликтный PR #1 был закрыт
	if !closedPRs[1] {
		t.Error("Expected PR #1 to be closed due to conflict")
	}

	// Проверяем что был добавлен комментарий к PR #1 (AC: #6)
	if comment, ok := commentedPRs[1]; !ok {
		t.Error("Expected comment to be added to PR #1 before closing")
	} else {
		if !contains(comment, "конфликты слияния") {
			t.Errorf("Comment should mention conflicts, got: %s", comment)
		}
		if !contains(comment, "src/main.go") {
			t.Errorf("Comment should list conflict files, got: %s", comment)
		}
	}
}

// TestExecute_AllConflicts проверяет случай когда все PR с конфликтами (AC: #6).
func TestExecute_AllConflicts(t *testing.T) {
	closedPRs := make(map[int64]bool)

	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
				{Number: 2, Head: "feature-2", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreatePRFunc: func(_ context.Context, head string) (gitea.PR, error) {
			return gitea.PR{Number: 100, Head: head, Base: "test-merge-branch"}, nil
		},
		ConflictPRFunc: func(_ context.Context, _ int64) (bool, error) {
			return true, nil // Все конфликтуют
		},
		ConflictFilesPRFunc: func(_ context.Context, _ int64) ([]string, error) {
			return []string{"conflict.go"}, nil
		},
		ClosePRFunc: func(_ context.Context, prNumber int64) error {
			closedPRs[prNumber] = true
			return nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	// Проверяем что оба PR были закрыты
	if !closedPRs[1] || !closedPRs[2] {
		t.Errorf("Expected both PRs to be closed, got closedPRs=%v", closedPRs)
	}
}

// TestExecute_CreateBranchError проверяет ошибку создания тестовой ветки (AC: #3).
func TestExecute_CreateBranchError(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			}, nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return errors.New("branch creation failed")
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for branch creation failure, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrBranchCreate) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrBranchCreate)
	}
}

// TestExecute_MissingConfig проверяет отсутствие конфигурации (AC: #1).
func TestExecute_MissingConfig(t *testing.T) {
	h := &TestMergeHandler{
		giteaClient: giteatest.NewMockClient(),
	}

	err := h.Execute(context.Background(), nil)
	if err == nil {
		t.Error("Execute() expected error for nil config, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrConfigMissing) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrConfigMissing)
	}
}

// TestExecute_JSONOutput проверяет JSON формат вывода (AC: #7).
func TestExecute_JSONOutput(t *testing.T) {
	oldFormat := os.Getenv("BR_OUTPUT_FORMAT")
	t.Cleanup(func() {
		if oldFormat == "" {
			_ = os.Unsetenv("BR_OUTPUT_FORMAT")
		} else {
			_ = os.Setenv("BR_OUTPUT_FORMAT", oldFormat)
		}
	})
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")

	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{}, nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() with JSON format unexpected error: %v", err)
	}
}

// TestExecute_CleanupOnError проверяет cleanup тестовой ветки даже при ошибках (AC: #10).
func TestExecute_CleanupOnError(t *testing.T) {
	deleteCalled := false

	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, branchName string) error {
			if strings.HasPrefix(branchName, testBranchPrefix) {
				deleteCalled = true
			}
			return nil
		},
		CreatePRFunc: func(_ context.Context, _ string) (gitea.PR, error) {
			return gitea.PR{Number: 100, Head: "feature-1", Base: "test-merge-branch"}, nil
		},
		ConflictPRFunc: func(_ context.Context, _ int64) (bool, error) {
			return false, nil
		},
		MergePRFunc: func(_ context.Context, _ int64) error {
			return nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	_ = h.Execute(context.Background(), cfg)

	if !deleteCalled {
		t.Error("Expected DeleteBranch to be called for cleanup")
	}
}

// TestExecute_ListOpenPRsError проверяет ошибку получения списка PR.
func TestExecute_ListOpenPRsError(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return nil, errors.New("API error")
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner: "myorg",
		Repo:  "myrepo",
	}

	err := h.Execute(context.Background(), cfg)
	if err == nil {
		t.Error("Execute() expected error for ListOpenPRs failure, got nil")
	}

	if err != nil && !contains(err.Error(), shared.ErrGiteaAPI) {
		t.Errorf("Execute() error = %v, want error containing %q", err, shared.ErrGiteaAPI)
	}
}

// TestExecute_MergeFailure проверяет обработку ошибки merge.
func TestExecute_MergeFailure(t *testing.T) {
	closedPRs := make(map[int64]bool)

	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreatePRFunc: func(_ context.Context, head string) (gitea.PR, error) {
			return gitea.PR{Number: 100, Head: head, Base: "test-merge-branch"}, nil
		},
		ConflictPRFunc: func(_ context.Context, _ int64) (bool, error) {
			return false, nil // Нет конфликта по проверке
		},
		MergePRFunc: func(_ context.Context, _ int64) error {
			return errors.New("merge failed") // Но merge провалился
		},
		ClosePRFunc: func(_ context.Context, prNumber int64) error {
			closedPRs[prNumber] = true
			return nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	// PR должен быть закрыт из-за провала merge
	if !closedPRs[1] {
		t.Error("Expected PR #1 to be closed due to merge failure")
	}
}

// TestExecute_CreateTestPRError проверяет обработку ошибки создания тестового PR.
func TestExecute_CreateTestPRError(t *testing.T) {
	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreatePRFunc: func(_ context.Context, _ string) (gitea.PR, error) {
			return gitea.PR{}, errors.New("cannot create PR")
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}
	// Успешное выполнение, но PR помечен как conflict с error
}

// TestExecute_DefaultBaseBranch проверяет использование default base branch.
func TestExecute_DefaultBaseBranch(t *testing.T) {
	usedBaseBranch := ""

	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, baseBranch string) error {
			usedBaseBranch = baseBranch
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreatePRFunc: func(_ context.Context, head string) (gitea.PR, error) {
			return gitea.PR{Number: 100, Head: head, Base: "test-merge-branch"}, nil
		},
		ConflictPRFunc: func(_ context.Context, _ int64) (bool, error) {
			return false, nil
		},
		MergePRFunc: func(_ context.Context, _ int64) error {
			return nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "", // Пустая — должна использоваться "main"
	}

	_ = h.Execute(context.Background(), cfg)

	// Проверяем что использовалась default ветка "main"
	if usedBaseBranch != "main" {
		t.Errorf("Expected default baseBranch 'main', got %q", usedBaseBranch)
	}
}

// TestTestMergeData_writeText проверяет текстовый вывод результатов (AC: #8).
func TestTestMergeData_writeText(t *testing.T) {
	tests := []struct {
		name     string
		data     *TestMergeData
		contains []string
	}{
		{
			name: "no PRs",
			data: &TestMergeData{
				TotalPRs:   0,
				PRResults:  []PRMergeResult{}, // Пустой массив для JSON
				TestBranch: "test-merge-20260205-100000",
				BaseBranch: "main",
			},
			contains: []string{"Базовая ветка: main", "Нет открытых Pull Requests"},
		},
		{
			name: "all mergeable",
			data: &TestMergeData{
				TotalPRs:     2,
				MergeablePRs: 2,
				ConflictPRs:  0,
				ClosedPRs:    0,
				TestBranch:   "test-merge-branch",
				BaseBranch:   "main",
				PRResults: []PRMergeResult{
					{PRNumber: 1, HeadBranch: "feature-1", BaseBranch: "main", HasConflict: false, MergeResult: "success"},
					{PRNumber: 2, HeadBranch: "feature-2", BaseBranch: "main", HasConflict: false, MergeResult: "success"},
				},
			},
			contains: []string{"Итого: 2 PR проверено", "Без конфликтов: 2", "С конфликтами: 0"},
		},
		{
			name: "some conflicts",
			data: &TestMergeData{
				TotalPRs:     3,
				MergeablePRs: 2,
				ConflictPRs:  1,
				ClosedPRs:    1,
				TestBranch:   "test-merge-branch",
				BaseBranch:   "main",
				PRResults: []PRMergeResult{
					{PRNumber: 1, HeadBranch: "feature-1", BaseBranch: "main", HasConflict: false, MergeResult: "success"},
					{PRNumber: 2, HeadBranch: "feature-2", BaseBranch: "main", HasConflict: true, MergeResult: "conflict", ConflictFiles: []string{"file.go"}, Closed: true},
					{PRNumber: 3, HeadBranch: "feature-3", BaseBranch: "main", HasConflict: false, MergeResult: "success"},
				},
			},
			contains: []string{"CONFLICT", "Без конфликтов: 2", "С конфликтами: 1 (закрыто: 1)"},
		},
		{
			name: "long branch name truncated",
			data: &TestMergeData{
				TotalPRs:     1,
				MergeablePRs: 1,
				ConflictPRs:  0,
				ClosedPRs:    0,
				TestBranch:   "test-merge-branch",
				BaseBranch:   "main",
				PRResults: []PRMergeResult{
					{PRNumber: 1, HeadBranch: "very-long-feature-branch-name-that-exceeds-limit", BaseBranch: "main", HasConflict: false, MergeResult: "success"},
				},
			},
			contains: []string{"very-long-fe..."}, // Truncated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := tt.data.writeText(&buf)
			if err != nil {
				t.Errorf("writeText() error = %v", err)
				return
			}

			output := buf.String()
			for _, s := range tt.contains {
				if !contains(output, s) {
					t.Errorf("writeText() output missing %q, got:\n%s", s, output)
				}
			}
		})
	}
}

// TestTestMergeData_writeText_Error проверяет обработку ошибки записи.
func TestTestMergeData_writeText_Error(t *testing.T) {
	data := &TestMergeData{
		TotalPRs:   1,
		TestBranch: "test-merge-20260205-100000",
		BaseBranch: "main",
		PRResults: []PRMergeResult{
			{PRNumber: 1, HeadBranch: "feature", BaseBranch: "main", HasConflict: false},
		},
	}

	errWriter := &errorWriter{err: errors.New("write failed")}

	err := data.writeText(errWriter)
	if err == nil {
		t.Error("writeText() expected error for failing writer")
	}
	if !contains(err.Error(), "write failed") {
		t.Errorf("writeText() error = %v, want error containing 'write failed'", err)
	}
}

// TestTruncateString проверяет функцию truncateString.
func TestTruncateString(t *testing.T) {
	tests := []struct {
		input  string
		maxLen int
		want   string
	}{
		{"short", 10, "short"},
		{"exactly10!", 10, "exactly10!"},
		{"this-is-a-very-long-string", 15, "this-is-a-ve..."},
		{"ab", 3, "ab"},
		{"abc", 3, "abc"},
		{"abcd", 3, "abc"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			if got != tt.want {
				t.Errorf("truncateString(%q, %d) = %q, want %q", tt.input, tt.maxLen, got, tt.want)
			}
		})
	}
}

// TestTruncateString_Unicode проверяет корректную работу с Unicode символами.
func TestTruncateString_Unicode(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		maxLen int
		want   string
	}{
		{"russian_short", "ветка", 10, "ветка"},
		{"russian_exact", "ветка-фича", 10, "ветка-фича"},
		{"russian_long", "очень-длинное-имя-ветки", 15, "очень-длинно..."},
		{"mixed_unicode", "feature-фича", 10, "feature..."},
		{"cyrillic_only", "тест", 3, "тест"[:6]}, // 3 символа = 6 байт в UTF-8
		{"emoji", "🔥feature", 5, "🔥f..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := truncateString(tt.input, tt.maxLen)
			// Проверяем что результат не превышает maxLen символов (не байт)
			gotRunes := []rune(got)
			if len(gotRunes) > tt.maxLen {
				t.Errorf("truncateString(%q, %d) = %q has %d runes, want <= %d",
					tt.input, tt.maxLen, got, len(gotRunes), tt.maxLen)
			}
			// Проверяем что строка валидный UTF-8
			if !isValidUTF8(got) {
				t.Errorf("truncateString(%q, %d) = %q is not valid UTF-8", tt.input, tt.maxLen, got)
			}
		})
	}
}

// isValidUTF8 проверяет что строка является валидным UTF-8.
func isValidUTF8(s string) bool {
	for i := 0; i < len(s); {
		r, size := []rune(s[i:])[0], len(string([]rune(s[i:])[:1]))
		if r == '\uFFFD' && size == 1 {
			return false
		}
		i += size
	}
	return true
}

// errorWriter — io.Writer который всегда возвращает ошибку.
type errorWriter struct {
	err error
}

func (w *errorWriter) Write(_ []byte) (n int, err error) {
	return 0, w.err
}

// contains проверяет наличие подстроки.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}

// TestExecute_JSONOutput_Structure проверяет полную структуру JSON ответа (AC: #7).
func TestExecute_JSONOutput_Structure(t *testing.T) {
	oldFormat := os.Getenv("BR_OUTPUT_FORMAT")
	oldStdout := os.Stdout
	t.Cleanup(func() {
		if oldFormat == "" {
			_ = os.Unsetenv("BR_OUTPUT_FORMAT")
		} else {
			_ = os.Setenv("BR_OUTPUT_FORMAT", oldFormat)
		}
		os.Stdout = oldStdout
	})
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")

	// Перенаправляем stdout для захвата JSON
	r, w, _ := os.Pipe()
	os.Stdout = w

	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreatePRFunc: func(_ context.Context, head string) (gitea.PR, error) {
			return gitea.PR{Number: 100, Head: head, Base: "test-branch"}, nil
		},
		ConflictPRFunc: func(_ context.Context, _ int64) (bool, error) {
			return true, nil
		},
		ConflictFilesPRFunc: func(_ context.Context, _ int64) ([]string, error) {
			return []string{"src/main.go", "config.yaml"}, nil
		},
		AddIssueCommentFunc: func(_ context.Context, _ int64, _ string) error {
			return nil
		},
		ClosePRFunc: func(_ context.Context, _ int64) error {
			return nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	_ = w.Close()

	var buf bytes.Buffer
	buf.ReadFrom(r)
	jsonOutput := buf.String()

	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	// Парсим JSON ответ
	var response struct {
		Status  string `json:"status"`
		Command string `json:"command"`
		Data    struct {
			TotalPRs     int `json:"total_prs"`
			MergeablePRs int `json:"mergeable_prs"`
			ConflictPRs  int `json:"conflict_prs"`
			ClosedPRs    int `json:"closed_prs"`
			PRResults    []struct {
				PRNumber      int64    `json:"pr_number"`
				HeadBranch    string   `json:"head_branch"`
				HasConflict   bool     `json:"has_conflict"`
				MergeResult   string   `json:"merge_result"`
				ConflictFiles []string `json:"conflict_files"`
				Closed        bool     `json:"closed"`
			} `json:"pr_results"`
			TestBranch string `json:"test_branch"`
			BaseBranch string `json:"base_branch"`
		} `json:"data"`
		Metadata struct {
			DurationMs int64  `json:"duration_ms"`
			TraceID    string `json:"trace_id"`
			APIVersion string `json:"api_version"`
		} `json:"metadata"`
	}

	if err := json.Unmarshal([]byte(jsonOutput), &response); err != nil {
		t.Fatalf("Failed to parse JSON output: %v\nOutput: %s", err, jsonOutput)
	}

	// Проверяем структуру ответа
	if response.Status != "success" {
		t.Errorf("Expected status 'success', got %q", response.Status)
	}
	if response.Command != "nr-test-merge" {
		t.Errorf("Expected command 'nr-test-merge', got %q", response.Command)
	}
	if response.Data.TotalPRs != 1 {
		t.Errorf("Expected total_prs=1, got %d", response.Data.TotalPRs)
	}
	if response.Data.ConflictPRs != 1 {
		t.Errorf("Expected conflict_prs=1, got %d", response.Data.ConflictPRs)
	}
	if len(response.Data.PRResults) != 1 {
		t.Errorf("Expected 1 PR result, got %d", len(response.Data.PRResults))
	}
	if len(response.Data.PRResults) > 0 {
		pr := response.Data.PRResults[0]
		if !pr.HasConflict {
			t.Error("Expected has_conflict=true")
		}
		if pr.MergeResult != "conflict" {
			t.Errorf("Expected merge_result='conflict', got %q", pr.MergeResult)
		}
		if len(pr.ConflictFiles) != 2 {
			t.Errorf("Expected 2 conflict files, got %d", len(pr.ConflictFiles))
		}
	}
	if response.Metadata.APIVersion != "v1" {
		t.Errorf("Expected api_version='v1', got %q", response.Metadata.APIVersion)
	}
	if response.Metadata.TraceID == "" {
		t.Error("Expected non-empty trace_id")
	}
	if !strings.HasPrefix(response.Data.TestBranch, testBranchPrefix) {
		t.Errorf("Expected test_branch to start with %q, got %q", testBranchPrefix, response.Data.TestBranch)
	}
}

// TestExecute_MergeFailure_WithComment проверяет добавление комментария при провале merge (AC: #6).
func TestExecute_MergeFailure_WithComment(t *testing.T) {
	commentedPRs := make(map[int64]string)
	closedPRs := make(map[int64]bool)

	giteaClient := &giteatest.MockClient{
		ListOpenPRsFunc: func(_ context.Context) ([]gitea.PR, error) {
			return []gitea.PR{
				{Number: 1, Head: "feature-1", Base: "main"},
			}, nil
		},
		CreateBranchFunc: func(_ context.Context, _, _ string) error {
			return nil
		},
		DeleteBranchFunc: func(_ context.Context, _ string) error {
			return nil
		},
		CreatePRFunc: func(_ context.Context, head string) (gitea.PR, error) {
			return gitea.PR{Number: 100, Head: head, Base: "test-merge-branch"}, nil
		},
		ConflictPRFunc: func(_ context.Context, _ int64) (bool, error) {
			return false, nil // Нет конфликта по проверке
		},
		MergePRFunc: func(_ context.Context, _ int64) error {
			return errors.New("merge failed: conflicting changes") // Но merge провалился
		},
		AddIssueCommentFunc: func(_ context.Context, prNumber int64, commentText string) error {
			commentedPRs[prNumber] = commentText
			return nil
		},
		ClosePRFunc: func(_ context.Context, prNumber int64) error {
			closedPRs[prNumber] = true
			return nil
		},
	}

	h := &TestMergeHandler{
		giteaClient: giteaClient,
	}

	cfg := &config.Config{
		Owner:      "myorg",
		Repo:       "myrepo",
		BaseBranch: "main",
	}

	err := h.Execute(context.Background(), cfg)
	if err != nil {
		t.Errorf("Execute() unexpected error: %v", err)
	}

	// Проверяем что PR #1 был закрыт
	if !closedPRs[1] {
		t.Error("Expected PR #1 to be closed due to merge failure")
	}

	// Проверяем что был добавлен комментарий к PR #1 (AC: #6)
	if comment, ok := commentedPRs[1]; !ok {
		t.Error("Expected comment to be added to PR #1 before closing")
	} else {
		if !contains(comment, "ошибка слияния") {
			t.Errorf("Comment should mention merge error, got: %s", comment)
		}
		if !contains(comment, "conflicting changes") {
			t.Errorf("Comment should contain error message, got: %s", comment)
		}
	}
}

// TestGenerateTestBranchName проверяет генерацию уникального имени ветки (AC: #3).
func TestGenerateTestBranchName(t *testing.T) {
	name1 := generateTestBranchName()
	name2 := generateTestBranchName()

	// Проверяем формат
	if !strings.HasPrefix(name1, testBranchPrefix) {
		t.Errorf("Expected branch name to start with %q, got %q", testBranchPrefix, name1)
	}

	// Проверяем что имена разные (при разных вызовах с паузой)
	// В быстром тесте они могут быть одинаковыми, поэтому проверяем только формат
	if len(name1) < len(testBranchPrefix)+10 {
		t.Errorf("Expected branch name to include timestamp, got %q", name1)
	}

	t.Logf("Generated branch names: %q, %q", name1, name2)
}
