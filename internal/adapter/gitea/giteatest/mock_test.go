package giteatest_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	"github.com/Kargones/apk-ci/internal/adapter/gitea/giteatest"
)

func TestNewMockClient(t *testing.T) {
	t.Parallel()

	mock := giteatest.NewMockClient()
	if mock == nil {
		t.Fatal("NewMockClient() вернул nil")
	}
}

func TestNewMockClientWithPR(t *testing.T) {
	t.Parallel()

	pr := &gitea.PRResponse{
		ID:     123,
		Number: 123,
		Title:  "Custom PR",
	}

	mock := giteatest.NewMockClientWithPR(pr)
	ctx := context.Background()

	got, err := mock.GetPR(ctx, 123)
	if err != nil {
		t.Fatalf("GetPR() error = %v", err)
	}
	if got.ID != pr.ID {
		t.Errorf("GetPR().ID = %d, want %d", got.ID, pr.ID)
	}
	if got.Title != pr.Title {
		t.Errorf("GetPR().Title = %q, want %q", got.Title, pr.Title)
	}
}

func TestNewMockClientWithCommits(t *testing.T) {
	t.Parallel()

	commits := []gitea.Commit{
		{SHA: "abc123"},
		{SHA: "def456"},
	}

	mock := giteatest.NewMockClientWithCommits(commits)
	ctx := context.Background()

	got, err := mock.GetCommits(ctx, "main", 10)
	if err != nil {
		t.Fatalf("GetCommits() error = %v", err)
	}
	if len(got) != len(commits) {
		t.Errorf("GetCommits() вернул %d коммитов, ожидалось %d", len(got), len(commits))
	}
}

func TestNewMockClientWithIssue(t *testing.T) {
	t.Parallel()

	issue := &gitea.Issue{
		ID:     100,
		Number: 100,
		Title:  "Custom Issue",
	}

	mock := giteatest.NewMockClientWithIssue(issue)
	ctx := context.Background()

	got, err := mock.GetIssue(ctx, 100)
	if err != nil {
		t.Fatalf("GetIssue() error = %v", err)
	}
	if got.Title != issue.Title {
		t.Errorf("GetIssue().Title = %q, want %q", got.Title, issue.Title)
	}
}

func TestNewMockClientWithRelease(t *testing.T) {
	t.Parallel()

	release := &gitea.Release{
		ID:      1,
		TagName: "v2.0.0",
		Name:    "Custom Release",
	}

	mock := giteatest.NewMockClientWithRelease(release)
	ctx := context.Background()

	got, err := mock.GetLatestRelease(ctx)
	if err != nil {
		t.Fatalf("GetLatestRelease() error = %v", err)
	}
	if got.TagName != release.TagName {
		t.Errorf("GetLatestRelease().TagName = %q, want %q", got.TagName, release.TagName)
	}

	gotByTag, err := mock.GetReleaseByTag(ctx, "v2.0.0")
	if err != nil {
		t.Fatalf("GetReleaseByTag() error = %v", err)
	}
	if gotByTag.Name != release.Name {
		t.Errorf("GetReleaseByTag().Name = %q, want %q", gotByTag.Name, release.Name)
	}
}

func TestMockClient_CustomFunctions(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	t.Run("custom GetPR function", func(t *testing.T) {
		t.Parallel()
		mock := giteatest.NewMockClient()
		mock.GetPRFunc = func(_ context.Context, prNumber int64) (*gitea.PRResponse, error) {
			return &gitea.PRResponse{
				ID:     prNumber,
				Number: prNumber,
				State:  "merged",
			}, nil
		}

		got, err := mock.GetPR(ctx, 42)
		if err != nil {
			t.Fatalf("GetPR() error = %v", err)
		}
		if got.State != "merged" {
			t.Errorf("GetPR().State = %q, want %q", got.State, "merged")
		}
	})

	t.Run("custom error return", func(t *testing.T) {
		t.Parallel()
		expectedErr := errors.New("custom error")
		mock := giteatest.NewMockClient()
		mock.GetLatestCommitFunc = func(_ context.Context, _ string) (*gitea.Commit, error) {
			return nil, expectedErr
		}

		_, err := mock.GetLatestCommit(ctx, "main")
		if err != expectedErr {
			t.Errorf("GetLatestCommit() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestMockClient_DefaultBehavior(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	mock := giteatest.NewMockClient()

	t.Run("GetPR returns default data", func(t *testing.T) {
		t.Parallel()
		pr, err := mock.GetPR(ctx, 1)
		if err != nil {
			t.Fatalf("GetPR() error = %v", err)
		}
		if pr == nil {
			t.Fatal("GetPR() вернул nil")
		}
	})

	t.Run("ListOpenPRs returns empty slice", func(t *testing.T) {
		t.Parallel()
		prs, err := mock.ListOpenPRs(ctx)
		if err != nil {
			t.Fatalf("ListOpenPRs() error = %v", err)
		}
		if prs == nil {
			t.Fatal("ListOpenPRs() вернул nil, ожидался пустой срез")
		}
	})

	t.Run("CreateBranch returns nil", func(t *testing.T) {
		t.Parallel()
		err := mock.CreateBranch(ctx, "feature/new", "main")
		if err != nil {
			t.Errorf("CreateBranch() error = %v, want nil", err)
		}
	})

	t.Run("DeleteBranch returns nil", func(t *testing.T) {
		t.Parallel()
		err := mock.DeleteBranch(ctx, "feature/old")
		if err != nil {
			t.Errorf("DeleteBranch() error = %v, want nil", err)
		}
	})

	t.Run("ConflictPR returns false", func(t *testing.T) {
		t.Parallel()
		conflict, err := mock.ConflictPR(ctx, 1)
		if err != nil {
			t.Fatalf("ConflictPR() error = %v", err)
		}
		if conflict {
			t.Error("ConflictPR() = true, want false")
		}
	})
}

// -------------------------------------------------------------------
// ISP Usage Examples
// -------------------------------------------------------------------

// usePRReader демонстрирует использование только PRReader интерфейса.
func usePRReader(reader gitea.PRReader, prNumber int64) (*gitea.PRResponse, error) {
	return reader.GetPR(context.Background(), prNumber)
}

// useCommitReader демонстрирует использование только CommitReader интерфейса.
func useCommitReader(reader gitea.CommitReader, branch string) (*gitea.Commit, error) {
	return reader.GetLatestCommit(context.Background(), branch)
}

// useFileReader демонстрирует использование только FileReader интерфейса.
func useFileReader(reader gitea.FileReader, filename string) ([]byte, error) {
	return reader.GetFileContent(context.Background(), filename)
}

func TestISPUsage_PRReader(t *testing.T) {
	t.Parallel()

	// MockClient можно использовать как PRReader
	var reader gitea.PRReader = giteatest.NewMockClient()

	pr, err := usePRReader(reader, 42)
	if err != nil {
		t.Fatalf("usePRReader() error = %v", err)
	}
	if pr.Number != 42 {
		t.Errorf("PR.Number = %d, want %d", pr.Number, 42)
	}
}

func TestISPUsage_CommitReader(t *testing.T) {
	t.Parallel()

	// MockClient можно использовать как CommitReader
	var reader gitea.CommitReader = giteatest.NewMockClient()

	commit, err := useCommitReader(reader, "main")
	if err != nil {
		t.Fatalf("useCommitReader() error = %v", err)
	}
	if commit == nil {
		t.Fatal("commit is nil")
	}
}

func TestISPUsage_FileReader(t *testing.T) {
	t.Parallel()

	mock := giteatest.NewMockClient()
	mock.GetFileContentFunc = func(_ context.Context, filename string) ([]byte, error) {
		return []byte("file content for " + filename), nil
	}

	var reader gitea.FileReader = mock

	content, err := useFileReader(reader, "README.md")
	if err != nil {
		t.Fatalf("useFileReader() error = %v", err)
	}
	expected := "file content for README.md"
	if string(content) != expected {
		t.Errorf("content = %q, want %q", string(content), expected)
	}
}

func TestISPUsage_MultipleInterfaces(t *testing.T) {
	t.Parallel()

	// Один mock может использоваться для разных интерфейсов
	mock := giteatest.NewMockClient()

	// Как PRReader
	var prReader gitea.PRReader = mock
	_, _ = prReader.GetPR(context.Background(), 1)

	// Как CommitReader
	var commitReader gitea.CommitReader = mock
	_, _ = commitReader.GetLatestCommit(context.Background(), "main")

	// Как полный Client
	var client gitea.Client = mock
	_, _ = client.GetPR(context.Background(), 1)
	_, _ = client.GetLatestCommit(context.Background(), "main")
}

// -------------------------------------------------------------------
// Test Data Functions
// -------------------------------------------------------------------

func TestPRData(t *testing.T) {
	t.Parallel()

	pr := giteatest.PRData()
	if pr.ID == 0 {
		t.Error("PRData().ID = 0, ожидалось ненулевое значение")
	}
	if pr.Number == 0 {
		t.Error("PRData().Number = 0, ожидалось ненулевое значение")
	}
	if pr.State == "" {
		t.Error("PRData().State пуст")
	}
}

func TestCommitData(t *testing.T) {
	t.Parallel()

	commits := giteatest.CommitData()
	if len(commits) == 0 {
		t.Fatal("CommitData() вернул пустой срез")
	}
	for i, commit := range commits {
		if commit.SHA == "" {
			t.Errorf("CommitData()[%d].SHA пуст", i)
		}
	}
}

func TestBranchData(t *testing.T) {
	t.Parallel()

	branches := giteatest.BranchData()
	if len(branches) == 0 {
		t.Fatal("BranchData() вернул пустой срез")
	}
	for i, branch := range branches {
		if branch.Name == "" {
			t.Errorf("BranchData()[%d].Name пуст", i)
		}
	}
}

func TestIssueData(t *testing.T) {
	t.Parallel()

	issue := giteatest.IssueData()
	if issue.ID == 0 {
		t.Error("IssueData().ID = 0, ожидалось ненулевое значение")
	}
	if issue.Title == "" {
		t.Error("IssueData().Title пуст")
	}
}

func TestReleaseData(t *testing.T) {
	t.Parallel()

	release := giteatest.ReleaseData()
	if release.ID == 0 {
		t.Error("ReleaseData().ID = 0, ожидалось ненулевое значение")
	}
	if release.TagName == "" {
		t.Error("ReleaseData().TagName пуст")
	}
	if len(release.Assets) == 0 {
		t.Error("ReleaseData().Assets пуст")
	}
}

func TestRepositoryData(t *testing.T) {
	t.Parallel()

	repo := giteatest.RepositoryData()
	if repo.ID == 0 {
		t.Error("RepositoryData().ID = 0, ожидалось ненулевое значение")
	}
	if repo.Name == "" {
		t.Error("RepositoryData().Name пуст")
	}
	if repo.Owner.Login == "" {
		t.Error("RepositoryData().Owner.Login пуст")
	}
}

// -------------------------------------------------------------------
// Example функции для документации
// -------------------------------------------------------------------

// ExampleNewMockClient демонстрирует базовое использование MockClient.
func ExampleNewMockClient() {
	mock := giteatest.NewMockClient()
	ctx := context.Background()

	pr, err := mock.GetPR(ctx, 42)
	if err != nil {
		fmt.Println("Ошибка:", err)
		return
	}
	fmt.Printf("PR #%d: %s\n", pr.Number, pr.State)
	// Output: PR #42: open
}

// ExampleNewMockClientWithPR демонстрирует создание mock с предзаданным PR.
func ExampleNewMockClientWithPR() {
	pr := &gitea.PRResponse{
		ID:        100,
		Number:    100,
		State:     "merged",
		Title:     "Feature: Add new functionality",
		Mergeable: false,
	}

	mock := giteatest.NewMockClientWithPR(pr)
	ctx := context.Background()

	result, _ := mock.GetPR(ctx, 100)
	fmt.Printf("PR #%d state: %s\n", result.Number, result.State)
	// Output: PR #100 state: merged
}

// ExampleMockClient_customFunction демонстрирует настройку custom функции.
func ExampleMockClient_customFunction() {
	mock := giteatest.NewMockClient()
	mock.GetLatestCommitFunc = func(_ context.Context, branch string) (*gitea.Commit, error) {
		return &gitea.Commit{
			SHA: "custom-sha-123",
			Commit: gitea.CommitDetails{
				Message: "feat: custom commit on " + branch,
			},
		}, nil
	}

	ctx := context.Background()
	commit, _ := mock.GetLatestCommit(ctx, "develop")
	fmt.Printf("Commit: %s\n", commit.Commit.Message)
	// Output: Commit: feat: custom commit on develop
}

// Example_iSPUsage демонстрирует использование ISP интерфейсов.
func Example_iSPUsage() {
	mock := giteatest.NewMockClient()

	// Использование как PRReader — только методы для чтения PR
	var prReader gitea.PRReader = mock
	prs, _ := prReader.ListOpenPRs(context.Background())
	fmt.Printf("Открытых PR: %d\n", len(prs))

	// Использование как ReleaseReader — только методы для чтения релизов
	var releaseReader gitea.ReleaseReader = mock
	release, _ := releaseReader.GetLatestRelease(context.Background())
	fmt.Printf("Последний релиз: %s\n", release.TagName)
	// Output: Открытых PR: 0
	// Последний релиз: v1.0.0
}
