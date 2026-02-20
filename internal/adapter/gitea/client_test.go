package gitea_test

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	entity_gitea "github.com/Kargones/apk-ci/internal/entity/gitea"
)

// mockAPI implements entity_gitea.APIInterface for testing
type mockAPI struct {
	activePRFunc             func() ([]entity_gitea.PR, error)
	conflictPRFunc           func(prNumber int64) (bool, error)
	conflictFilesPRFunc      func(prNumber int64) ([]string, error)
	getCommitsFunc           func(branch string, limit int) ([]entity_gitea.Commit, error)
	getLatestCommitFunc      func(branch string) (*entity_gitea.Commit, error)
	getCommitFilesFunc       func(commitSHA string) ([]entity_gitea.CommitFile, error)
	getCommitsBetweenFunc    func(baseSHA, headSHA string) ([]entity_gitea.Commit, error)
	getFirstCommitOfBranchFunc func(branch, baseBranch string) (*entity_gitea.Commit, error)
	getBranchCommitRangeFunc func(branch string) (*entity_gitea.BranchCommitRange, error)
	getFileContentFunc       func(fileName string) ([]byte, error)
	getRepositoryContentsFunc func(filepath, branch string) ([]entity_gitea.FileInfo, error)
	analyzeProjectStructureFunc func(branch string) ([]string, error)
	getBranchesFunc          func(repo string) ([]entity_gitea.Branch, error)
	createTestBranchFunc     func() error
	deleteTestBranchFunc     func() error
	getLatestReleaseFunc     func() (*entity_gitea.Release, error)
	getReleaseByTagFunc      func(tag string) (*entity_gitea.Release, error)
	getIssueFunc             func(issueNumber int64) (*entity_gitea.Issue, error)
	addIssueCommentFunc      func(issueNumber int64, commentText string) error
	closeIssueFunc           func(issueNumber int64) error
	createPRFunc             func(head string) (entity_gitea.PR, error)
	createPRWithOptionsFunc  func(opts entity_gitea.CreatePROptions) (*entity_gitea.PRResponse, error)
	mergePRFunc              func(prNumber int64, l *slog.Logger) error
	closePRFunc              func(prNumber int64) error
	setRepositoryStateFunc   func(l *slog.Logger, operations []entity_gitea.BatchOperation, branch, commitMessage string) error
	isUserInTeamFunc         func(l *slog.Logger, username, orgName, teamName string) (bool, error)
	getTeamMembersFunc       func(orgName, teamName string) ([]string, error)
	searchOrgReposFunc       func(orgName string) ([]entity_gitea.Repository, error)
}

func (m *mockAPI) GetIssue(ctx context.Context, issueNumber int64) (*entity_gitea.Issue, error) {
	if m.getIssueFunc != nil {
		return m.getIssueFunc(issueNumber)
	}
	return &entity_gitea.Issue{ID: issueNumber, Number: issueNumber, Title: "Test Issue"}, nil
}

func (m *mockAPI) GetFileContent(ctx context.Context, fileName string) ([]byte, error) {
	if m.getFileContentFunc != nil {
		return m.getFileContentFunc(fileName)
	}
	return []byte("test content"), nil
}

func (m *mockAPI) GetConfigData(ctx context.Context, l *slog.Logger, filename string) ([]byte, error) {
	return []byte("config data"), nil
}

func (m *mockAPI) AddIssueComment(ctx context.Context, issueNumber int64, commentText string) error {
	if m.addIssueCommentFunc != nil {
		return m.addIssueCommentFunc(issueNumber, commentText)
	}
	return nil
}

func (m *mockAPI) CloseIssue(ctx context.Context, issueNumber int64) error {
	if m.closeIssueFunc != nil {
		return m.closeIssueFunc(issueNumber)
	}
	return nil
}

func (m *mockAPI) ConflictPR(ctx context.Context, prNumber int64) (bool, error) {
	if m.conflictPRFunc != nil {
		return m.conflictPRFunc(prNumber)
	}
	return false, nil
}

func (m *mockAPI) ConflictFilesPR(ctx context.Context, prNumber int64) ([]string, error) {
	if m.conflictFilesPRFunc != nil {
		return m.conflictFilesPRFunc(prNumber)
	}
	return []string{}, nil
}

func (m *mockAPI) GetRepositoryContents(ctx context.Context, filepath, branch string) ([]entity_gitea.FileInfo, error) {
	if m.getRepositoryContentsFunc != nil {
		return m.getRepositoryContentsFunc(filepath, branch)
	}
	return []entity_gitea.FileInfo{}, nil
}

func (m *mockAPI) AnalyzeProjectStructure(ctx context.Context, branch string) ([]string, error) {
	if m.analyzeProjectStructureFunc != nil {
		return m.analyzeProjectStructureFunc(branch)
	}
	return []string{"file1.xml", "file2.xml"}, nil
}

func (m *mockAPI) AnalyzeProject(ctx context.Context, branch string) ([]string, error) {
	return []string{"project.xml"}, nil
}

func (m *mockAPI) GetLatestCommit(ctx context.Context, branch string) (*entity_gitea.Commit, error) {
	if m.getLatestCommitFunc != nil {
		return m.getLatestCommitFunc(branch)
	}
	return &entity_gitea.Commit{SHA: "abc123"}, nil
}

func (m *mockAPI) GetCommitFiles(ctx context.Context, commitSHA string) ([]entity_gitea.CommitFile, error) {
	if m.getCommitFilesFunc != nil {
		return m.getCommitFilesFunc(commitSHA)
	}
	return []entity_gitea.CommitFile{}, nil
}

func (m *mockAPI) GetLatestRelease(ctx context.Context) (*entity_gitea.Release, error) {
	if m.getLatestReleaseFunc != nil {
		return m.getLatestReleaseFunc()
	}
	return &entity_gitea.Release{ID: 1, TagName: "v1.0.0"}, nil
}

func (m *mockAPI) GetReleaseByTag(ctx context.Context, tag string) (*entity_gitea.Release, error) {
	if m.getReleaseByTagFunc != nil {
		return m.getReleaseByTagFunc(tag)
	}
	return &entity_gitea.Release{ID: 1, TagName: tag}, nil
}

func (m *mockAPI) IsUserInTeam(ctx context.Context, l *slog.Logger, username, orgName, teamName string) (bool, error) {
	if m.isUserInTeamFunc != nil {
		return m.isUserInTeamFunc(l, username, orgName, teamName)
	}
	return true, nil
}

func (m *mockAPI) GetCommits(ctx context.Context, branch string, limit int) ([]entity_gitea.Commit, error) {
	if m.getCommitsFunc != nil {
		return m.getCommitsFunc(branch, limit)
	}
	return []entity_gitea.Commit{{SHA: "commit1"}, {SHA: "commit2"}}, nil
}

func (m *mockAPI) GetFirstCommitOfBranch(ctx context.Context, branch, baseBranch string) (*entity_gitea.Commit, error) {
	if m.getFirstCommitOfBranchFunc != nil {
		return m.getFirstCommitOfBranchFunc(branch, baseBranch)
	}
	return &entity_gitea.Commit{SHA: "firstcommit"}, nil
}

func (m *mockAPI) GetCommitsBetween(ctx context.Context, baseSHA, headSHA string) ([]entity_gitea.Commit, error) {
	if m.getCommitsBetweenFunc != nil {
		return m.getCommitsBetweenFunc(baseSHA, headSHA)
	}
	return []entity_gitea.Commit{{SHA: "commit1"}}, nil
}

func (m *mockAPI) GetBranchCommitRange(ctx context.Context, branch string) (*entity_gitea.BranchCommitRange, error) {
	if m.getBranchCommitRangeFunc != nil {
		return m.getBranchCommitRangeFunc(branch)
	}
	return &entity_gitea.BranchCommitRange{
		FirstCommit: &entity_gitea.Commit{SHA: "first"},
		LastCommit:  &entity_gitea.Commit{SHA: "last"},
	}, nil
}

func (m *mockAPI) ActivePR(ctx context.Context) ([]entity_gitea.PR, error) {
	if m.activePRFunc != nil {
		return m.activePRFunc()
	}
	return []entity_gitea.PR{{ID: 1, Number: 1, Base: "main", Head: "feature"}}, nil
}

func (m *mockAPI) DeleteTestBranch(ctx context.Context) error {
	if m.deleteTestBranchFunc != nil {
		return m.deleteTestBranchFunc()
	}
	return nil
}

func (m *mockAPI) CreateTestBranch(ctx context.Context) error {
	if m.createTestBranchFunc != nil {
		return m.createTestBranchFunc()
	}
	return nil
}

func (m *mockAPI) CreatePR(ctx context.Context, head string) (entity_gitea.PR, error) {
	if m.createPRFunc != nil {
		return m.createPRFunc(head)
	}
	return entity_gitea.PR{ID: 1, Number: 1, Head: head}, nil
}

func (m *mockAPI) CreatePRWithOptions(ctx context.Context, opts entity_gitea.CreatePROptions) (*entity_gitea.PRResponse, error) {
	if m.createPRWithOptionsFunc != nil {
		return m.createPRWithOptionsFunc(opts)
	}
	return &entity_gitea.PRResponse{ID: 1, Number: 1, Title: opts.Title}, nil
}

func (m *mockAPI) MergePR(ctx context.Context, prNumber int64, l *slog.Logger) error {
	if m.mergePRFunc != nil {
		return m.mergePRFunc(prNumber, l)
	}
	return nil
}

func (m *mockAPI) ClosePR(ctx context.Context, prNumber int64) error {
	if m.closePRFunc != nil {
		return m.closePRFunc(prNumber)
	}
	return nil
}

func (m *mockAPI) SetRepositoryState(ctx context.Context, l *slog.Logger, operations []entity_gitea.BatchOperation, branch, commitMessage string) error {
	if m.setRepositoryStateFunc != nil {
		return m.setRepositoryStateFunc(l, operations, branch, commitMessage)
	}
	return nil
}

func (m *mockAPI) GetTeamMembers(ctx context.Context, orgName, teamName string) ([]string, error) {
	if m.getTeamMembersFunc != nil {
		return m.getTeamMembersFunc(orgName, teamName)
	}
	return []string{"user1", "user2"}, nil
}

func (m *mockAPI) GetBranches(ctx context.Context, repo string) ([]entity_gitea.Branch, error) {
	if m.getBranchesFunc != nil {
		return m.getBranchesFunc(repo)
	}
	return []entity_gitea.Branch{{Name: "main", Label: "origin:main"}}, nil
}

func (m *mockAPI) SearchOrgRepos(ctx context.Context, orgName string) ([]entity_gitea.Repository, error) {
	if m.searchOrgReposFunc != nil {
		return m.searchOrgReposFunc(orgName)
	}
	return []entity_gitea.Repository{{ID: 1, Name: "repo1"}}, nil
}

// Tests

func TestNewAPIClient_ImplementsClient(t *testing.T) {
	api := &entity_gitea.API{}
	client := gitea.NewAPIClient(api)
	var _ gitea.Client = client
	assert.NotNil(t, client)
}

func TestNewAPIClientWithLogger(t *testing.T) {
	api := &entity_gitea.API{}
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	client := gitea.NewAPIClientWithLogger(api, logger)
	assert.NotNil(t, client)
}

func TestNewAPIClientWithInterface(t *testing.T) {
	mock := &mockAPI{}
	client := gitea.NewAPIClientWithInterface(mock, nil)
	assert.NotNil(t, client)
}

// PRReader tests

func TestAPIClient_GetPR(t *testing.T) {
	mock := &mockAPI{}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	resp, err := client.GetPR(context.Background(), 1)

	assert.Nil(t, resp)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "GetPR not directly supported")
}

func TestAPIClient_ListOpenPRs(t *testing.T) {
	mock := &mockAPI{
		activePRFunc: func() ([]entity_gitea.PR, error) {
			return []entity_gitea.PR{
				{ID: 1, Number: 1, Base: "main", Head: "feature1"},
				{ID: 2, Number: 2, Base: "main", Head: "feature2"},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	prs, err := client.ListOpenPRs(context.Background())

	require.NoError(t, err)
	assert.Len(t, prs, 2)
	assert.Equal(t, int64(1), prs[0].Number)
	assert.Equal(t, "main", prs[0].Base)
	assert.Equal(t, "feature1", prs[0].Head)
}

func TestAPIClient_ListOpenPRs_Error(t *testing.T) {
	testErr := errors.New("api error")
	mock := &mockAPI{
		activePRFunc: func() ([]entity_gitea.PR, error) {
			return nil, testErr
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	prs, err := client.ListOpenPRs(context.Background())

	assert.Nil(t, prs)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ListOpenPRs failed")
}

func TestAPIClient_ConflictPR(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		conflictPRFunc: func(prNumber int64) (bool, error) {
			return true, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	hasConflict, err := client.ConflictPR(ctx, 1)

	require.NoError(t, err)
	assert.True(t, hasConflict)
}

func TestAPIClient_ConflictFilesPR(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		conflictFilesPRFunc: func(prNumber int64) ([]string, error) {
			return []string{"file1.go", "file2.go"}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	files, err := client.ConflictFilesPR(ctx, 1)

	require.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "file1.go")
}

// CommitReader tests

func TestAPIClient_GetCommits(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getCommitsFunc: func(branch string, limit int) ([]entity_gitea.Commit, error) {
			return []entity_gitea.Commit{
				{SHA: "abc123"},
				{SHA: "def456"},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	commits, err := client.GetCommits(ctx, "main", 10)

	require.NoError(t, err)
	assert.Len(t, commits, 2)
	assert.Equal(t, "abc123", commits[0].SHA)
}

func TestAPIClient_GetCommits_Error(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getCommitsFunc: func(branch string, limit int) ([]entity_gitea.Commit, error) {
			return nil, errors.New("api error")
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	commits, err := client.GetCommits(ctx, "main", 10)

	assert.Nil(t, commits)
	assert.Error(t, err)
}

func TestAPIClient_GetLatestCommit(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getLatestCommitFunc: func(branch string) (*entity_gitea.Commit, error) {
			return &entity_gitea.Commit{SHA: "latest123"}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	commit, err := client.GetLatestCommit(ctx, "main")

	require.NoError(t, err)
	assert.Equal(t, "latest123", commit.SHA)
}

func TestAPIClient_GetCommitFiles(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getCommitFilesFunc: func(commitSHA string) ([]entity_gitea.CommitFile, error) {
			return []entity_gitea.CommitFile{
				{Filename: "file1.go", Status: "added"},
				{Filename: "file2.go", Status: "modified"},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	files, err := client.GetCommitFiles(ctx, "abc123")

	require.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "file1.go", files[0].Filename)
	assert.Equal(t, "added", files[0].Status)
}

func TestAPIClient_GetCommitsBetween(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getCommitsBetweenFunc: func(baseSHA, headSHA string) ([]entity_gitea.Commit, error) {
			return []entity_gitea.Commit{{SHA: "middle1"}}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	commits, err := client.GetCommitsBetween(ctx, "base123", "head456")

	require.NoError(t, err)
	assert.Len(t, commits, 1)
}

func TestAPIClient_GetFirstCommitOfBranch(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getFirstCommitOfBranchFunc: func(branch, baseBranch string) (*entity_gitea.Commit, error) {
			return &entity_gitea.Commit{SHA: "firstcommit"}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	commit, err := client.GetFirstCommitOfBranch(ctx, "feature", "main")

	require.NoError(t, err)
	assert.Equal(t, "firstcommit", commit.SHA)
}

func TestAPIClient_GetBranchCommitRange(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getBranchCommitRangeFunc: func(branch string) (*entity_gitea.BranchCommitRange, error) {
			return &entity_gitea.BranchCommitRange{
				FirstCommit: &entity_gitea.Commit{SHA: "first"},
				LastCommit:  &entity_gitea.Commit{SHA: "last"},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	range_, err := client.GetBranchCommitRange(ctx, "feature")

	require.NoError(t, err)
	assert.Equal(t, "first", range_.FirstCommit.SHA)
	assert.Equal(t, "last", range_.LastCommit.SHA)
}

// FileReader tests

func TestAPIClient_GetFileContent(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getFileContentFunc: func(fileName string) ([]byte, error) {
			return []byte("file content here"), nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	content, err := client.GetFileContent(ctx, "test.xml")

	require.NoError(t, err)
	assert.Equal(t, []byte("file content here"), content)
}

func TestAPIClient_GetRepositoryContents(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getRepositoryContentsFunc: func(filepath, branch string) ([]entity_gitea.FileInfo, error) {
			return []entity_gitea.FileInfo{
				{Name: "file1.xml", Path: "path/file1.xml", Type: "file"},
				{Name: "dir1", Path: "path/dir1", Type: "dir"},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	files, err := client.GetRepositoryContents(ctx, "path", "main")

	require.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Equal(t, "file1.xml", files[0].Name)
	assert.Equal(t, "file", files[0].Type)
}

func TestAPIClient_AnalyzeProjectStructure(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		analyzeProjectStructureFunc: func(branch string) ([]string, error) {
			return []string{"src/file1.bsl", "src/file2.bsl"}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	files, err := client.AnalyzeProjectStructure(ctx, "main")

	require.NoError(t, err)
	assert.Len(t, files, 2)
}

// BranchManager tests

func TestAPIClient_GetBranches(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getBranchesFunc: func(repo string) ([]entity_gitea.Branch, error) {
			return []entity_gitea.Branch{
				{Name: "main", Label: "origin:main"},
				{Name: "develop", Label: "origin:develop"},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	branches, err := client.GetBranches(ctx, "test-repo")

	require.NoError(t, err)
	assert.Len(t, branches, 2)
	assert.Equal(t, "main", branches[0].Name)
}

func TestAPIClient_CreateBranch(t *testing.T) {
	called := false
	mock := &mockAPI{
		createTestBranchFunc: func() error {
			called = true
			return nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	err := client.CreateBranch(context.Background(), "new-branch", "main")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestAPIClient_DeleteBranch(t *testing.T) {
	called := false
	mock := &mockAPI{
		deleteTestBranchFunc: func() error {
			called = true
			return nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	err := client.DeleteBranch(context.Background(), "test-branch")

	require.NoError(t, err)
	assert.True(t, called)
}

// ReleaseReader tests

func TestAPIClient_GetLatestRelease(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getLatestReleaseFunc: func() (*entity_gitea.Release, error) {
			return &entity_gitea.Release{
				ID:      1,
				TagName: "v1.0.0",
				Name:    "First Release",
				Assets: []entity_gitea.ReleaseAsset{
					{ID: 1, Name: "app.zip", Size: 1024},
				},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	release, err := client.GetLatestRelease(ctx)

	require.NoError(t, err)
	assert.Equal(t, "v1.0.0", release.TagName)
	assert.Equal(t, "First Release", release.Name)
	assert.Len(t, release.Assets, 1)
}

func TestAPIClient_GetReleaseByTag(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getReleaseByTagFunc: func(tag string) (*entity_gitea.Release, error) {
			return &entity_gitea.Release{ID: 2, TagName: tag, Name: "Release " + tag}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	release, err := client.GetReleaseByTag(ctx, "v2.0.0")

	require.NoError(t, err)
	assert.Equal(t, "v2.0.0", release.TagName)
}

// IssueManager tests

func TestAPIClient_GetIssue(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getIssueFunc: func(issueNumber int64) (*entity_gitea.Issue, error) {
			return &entity_gitea.Issue{
				ID:     1,
				Number: issueNumber,
				Title:  "Test Issue",
				Body:   "Issue body",
				State:  "open",
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	issue, err := client.GetIssue(ctx, 42)

	require.NoError(t, err)
	assert.Equal(t, int64(42), issue.Number)
	assert.Equal(t, "Test Issue", issue.Title)
	assert.Equal(t, "open", issue.State)
}

func TestAPIClient_AddIssueComment(t *testing.T) {
	ctx := context.Background()
	called := false
	mock := &mockAPI{
		addIssueCommentFunc: func(issueNumber int64, commentText string) error {
			called = true
			assert.Equal(t, int64(1), issueNumber)
			assert.Equal(t, "test comment", commentText)
			return nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	err := client.AddIssueComment(ctx, 1, "test comment")

	require.NoError(t, err)
	assert.True(t, called)
}

func TestAPIClient_CloseIssue(t *testing.T) {
	ctx := context.Background()
	called := false
	mock := &mockAPI{
		closeIssueFunc: func(issueNumber int64) error {
			called = true
			return nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	err := client.CloseIssue(ctx, 1)

	require.NoError(t, err)
	assert.True(t, called)
}

// PRManager tests

func TestAPIClient_CreatePR(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		createPRFunc: func(head string) (entity_gitea.PR, error) {
			return entity_gitea.PR{ID: 1, Number: 10, Head: head, Base: "main"}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	pr, err := client.CreatePR(ctx, "feature-branch")

	require.NoError(t, err)
	assert.Equal(t, int64(10), pr.Number)
	assert.Equal(t, "feature-branch", pr.Head)
}

func TestAPIClient_CreatePRWithOptions(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		createPRWithOptionsFunc: func(opts entity_gitea.CreatePROptions) (*entity_gitea.PRResponse, error) {
			return &entity_gitea.PRResponse{
				ID:        1,
				Number:    15,
				Title:     opts.Title,
				Body:      opts.Body,
				State:     "open",
				Mergeable: true,
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	opts := gitea.CreatePROptions{
		Title:     "New PR",
		Body:      "PR description",
		Head:      "feature",
		Base:      "main",
		Assignees: []string{"user1"},
		Labels:    []int64{1, 2},
	}
	pr, err := client.CreatePRWithOptions(ctx, opts)

	require.NoError(t, err)
	assert.Equal(t, int64(15), pr.Number)
	assert.Equal(t, "New PR", pr.Title)
}

func TestAPIClient_MergePR(t *testing.T) {
	ctx := context.Background()
	called := false
	mock := &mockAPI{
		mergePRFunc: func(prNumber int64, l *slog.Logger) error {
			called = true
			return nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	err := client.MergePR(ctx, 1)

	require.NoError(t, err)
	assert.True(t, called)
}

func TestAPIClient_ClosePR(t *testing.T) {
	ctx := context.Background()
	called := false
	mock := &mockAPI{
		closePRFunc: func(prNumber int64) error {
			called = true
			return nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	err := client.ClosePR(ctx, 1)

	require.NoError(t, err)
	assert.True(t, called)
}

// RepositoryWriter tests

func TestAPIClient_SetRepositoryState(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		setRepositoryStateFunc: func(l *slog.Logger, operations []entity_gitea.BatchOperation, branch, commitMessage string) error {
			assert.Len(t, operations, 2)
			assert.Equal(t, "test-branch", branch)
			assert.Equal(t, "test commit", commitMessage)
			return nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	ops := []gitea.BatchOperation{
		{Operation: "create", Path: "file1.txt", Content: "content1"},
		{Operation: "update", Path: "file2.txt", Content: "content2"},
	}
	err := client.SetRepositoryState(ctx, ops, "test-branch", "test commit")

	require.NoError(t, err)
}

// TeamReader tests

func TestAPIClient_IsUserInTeam(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		isUserInTeamFunc: func(l *slog.Logger, username, orgName, teamName string) (bool, error) {
			return username == "testuser", nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	inTeam, err := client.IsUserInTeam(ctx, "testuser", "org1", "team1")

	require.NoError(t, err)
	assert.True(t, inTeam)
}

func TestAPIClient_GetTeamMembers(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getTeamMembersFunc: func(orgName, teamName string) ([]string, error) {
			return []string{"user1", "user2", "user3"}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	members, err := client.GetTeamMembers(ctx, "org1", "team1")

	require.NoError(t, err)
	assert.Len(t, members, 3)
}

// OrgReader tests

func TestAPIClient_SearchOrgRepos(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		searchOrgReposFunc: func(orgName string) ([]entity_gitea.Repository, error) {
			return []entity_gitea.Repository{
				{ID: 1, Name: "repo1", FullName: "org/repo1", Private: false},
				{ID: 2, Name: "repo2", FullName: "org/repo2", Private: true},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	repos, err := client.SearchOrgRepos(ctx, "org1")

	require.NoError(t, err)
	assert.Len(t, repos, 2)
	assert.Equal(t, "repo1", repos[0].Name)
	assert.False(t, repos[0].Private)
	assert.True(t, repos[1].Private)
}

// Conversion tests - edge cases

func TestAPIClient_ConvertCommits_Empty(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getCommitsFunc: func(branch string, limit int) ([]entity_gitea.Commit, error) {
			return []entity_gitea.Commit{}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	commits, err := client.GetCommits(ctx, "main", 10)

	require.NoError(t, err)
	assert.Empty(t, commits)
}

func TestAPIClient_ConvertBranchCommitRange_NilCommits(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getBranchCommitRangeFunc: func(branch string) (*entity_gitea.BranchCommitRange, error) {
			return &entity_gitea.BranchCommitRange{
				FirstCommit: nil,
				LastCommit:  nil,
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	range_, err := client.GetBranchCommitRange(ctx, "feature")

	require.NoError(t, err)
	assert.Nil(t, range_.FirstCommit)
	assert.Nil(t, range_.LastCommit)
}

func TestAPIClient_ListOpenPRs_Empty(t *testing.T) {
	mock := &mockAPI{
		activePRFunc: func() ([]entity_gitea.PR, error) {
			return []entity_gitea.PR{}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	prs, err := client.ListOpenPRs(context.Background())

	require.NoError(t, err)
	assert.Empty(t, prs)
}

func TestAPIClient_GetIssue_WithUser(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getIssueFunc: func(issueNumber int64) (*entity_gitea.Issue, error) {
			return &entity_gitea.Issue{
				ID:     1,
				Number: issueNumber,
				Title:  "Test",
				User: struct {
					Login string `json:"login"`
					ID    int64  `json:"id"`
				}{
					Login: "testuser",
					ID:    100,
				},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	issue, err := client.GetIssue(ctx, 1)

	require.NoError(t, err)
	assert.Equal(t, "testuser", issue.User.Login)
	assert.Equal(t, int64(100), issue.User.ID)
}

func TestAPIClient_GetLatestRelease_WithAssets(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getLatestReleaseFunc: func() (*entity_gitea.Release, error) {
			return &entity_gitea.Release{
				ID:          1,
				TagName:     "v1.0.0",
				Name:        "Release",
				Body:        "Description",
				CreatedAt:   "2024-01-01",
				PublishedAt: "2024-01-02",
				Assets: []entity_gitea.ReleaseAsset{
					{ID: 1, Name: "app.zip", Size: 1024, DownloadURL: "http://example.com/app.zip"},
					{ID: 2, Name: "src.tar.gz", Size: 2048, DownloadURL: "http://example.com/src.tar.gz"},
				},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	release, err := client.GetLatestRelease(ctx)

	require.NoError(t, err)
	assert.Len(t, release.Assets, 2)
	assert.Equal(t, "app.zip", release.Assets[0].Name)
	assert.Equal(t, "http://example.com/app.zip", release.Assets[0].DownloadURL)
}

func TestAPIClient_GetBranches_WithCommit(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		getBranchesFunc: func(repo string) ([]entity_gitea.Branch, error) {
			return []entity_gitea.Branch{
				{
					Name:  "main",
					Label: "origin:main",
					Commit: struct {
						ID string `json:"id"`
					}{ID: "abc123"},
				},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	branches, err := client.GetBranches(ctx, "repo")

	require.NoError(t, err)
	assert.Len(t, branches, 1)
	assert.Equal(t, "abc123", branches[0].Commit.ID)
}

func TestAPIClient_SearchOrgRepos_WithOwner(t *testing.T) {
	ctx := context.Background()
	mock := &mockAPI{
		searchOrgReposFunc: func(orgName string) ([]entity_gitea.Repository, error) {
			return []entity_gitea.Repository{
				{
					ID:            1,
					Name:          "repo1",
					FullName:      "org/repo1",
					DefaultBranch: "main",
					Private:       false,
					Fork:          true,
					Owner: entity_gitea.RepositoryOwner{
						ID:    10,
						Login: "org",
						Type:  "Organization",
					},
				},
			}, nil
		},
	}
	client := gitea.NewAPIClientWithInterface(mock, nil)

	repos, err := client.SearchOrgRepos(ctx, "org")

	require.NoError(t, err)
	assert.Len(t, repos, 1)
	assert.Equal(t, "org", repos[0].Owner.Login)
	assert.Equal(t, "Organization", repos[0].Owner.Type)
	assert.True(t, repos[0].Fork)
}
