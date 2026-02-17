package gitea

import (
	"context"
	"log/slog"

	entity_gitea "github.com/Kargones/apk-ci/internal/entity/gitea"
)

// Compile-time проверка реализации интерфейса.
var _ Client = (*APIClient)(nil)

// APIClient реализует интерфейс Client, делегируя вызовы entity/gitea.API.
// Принимает context.Context в каждом методе (для совместимости с интерфейсом),
// но entity API контекст не использует — это будет исправлено при рефакторинге entity.
type APIClient struct {
	api    entity_gitea.APIInterface
	logger *slog.Logger
}

// NewAPIClient создаёт новый APIClient, оборачивающий entity/gitea.API.
func NewAPIClient(api *entity_gitea.API) *APIClient {
	return &APIClient{
		api:    api,
		logger: slog.Default(),
	}
}

// NewAPIClientWithLogger создаёт APIClient с пользовательским логгером.
func NewAPIClientWithLogger(api *entity_gitea.API, logger *slog.Logger) *APIClient {
	return &APIClient{
		api:    api,
		logger: logger,
	}
}

// NewAPIClientWithInterface создаёт APIClient с интерфейсом для тестирования.
func NewAPIClientWithInterface(api entity_gitea.APIInterface, logger *slog.Logger) *APIClient {
	if logger == nil {
		logger = slog.Default()
	}
	return &APIClient{
		api:    api,
		logger: logger,
	}
}

// -------------------------------------------------------------------
// PRReader
// -------------------------------------------------------------------

func (c *APIClient) GetPR(ctx context.Context, prNumber int64) (*PRResponse, error) {
	// entity API не имеет GetPR — используем workaround через список PR
	return nil, NewGiteaError(ErrGiteaAPI, "GetPR not directly supported by entity API, use ListOpenPRs", nil)
}

func (c *APIClient) ListOpenPRs(ctx context.Context) ([]PR, error) {
	entityPRs, err := c.api.ActivePR()
	if err != nil {
		return nil, NewGiteaError(ErrGiteaAPI, "ListOpenPRs failed", err)
	}
	result := make([]PR, len(entityPRs))
	for i, p := range entityPRs {
		result[i] = convertPR(p)
	}
	return result, nil
}

func (c *APIClient) ConflictPR(ctx context.Context, prNumber int64) (bool, error) {
	return c.api.ConflictPR(prNumber)
}

func (c *APIClient) ConflictFilesPR(ctx context.Context, prNumber int64) ([]string, error) {
	return c.api.ConflictFilesPR(prNumber)
}

// -------------------------------------------------------------------
// CommitReader
// -------------------------------------------------------------------

func (c *APIClient) GetCommits(ctx context.Context, branch string, limit int) ([]Commit, error) {
	entityCommits, err := c.api.GetCommits(branch, limit)
	if err != nil {
		return nil, err
	}
	return convertCommits(entityCommits), nil
}

func (c *APIClient) GetLatestCommit(ctx context.Context, branch string) (*Commit, error) {
	ec, err := c.api.GetLatestCommit(branch)
	if err != nil {
		return nil, err
	}
	result := convertCommit(*ec)
	return &result, nil
}

func (c *APIClient) GetCommitFiles(ctx context.Context, commitSHA string) ([]CommitFile, error) {
	entityFiles, err := c.api.GetCommitFiles(commitSHA)
	if err != nil {
		return nil, err
	}
	return convertCommitFiles(entityFiles), nil
}

func (c *APIClient) GetCommitsBetween(ctx context.Context, baseSHA, headSHA string) ([]Commit, error) {
	entityCommits, err := c.api.GetCommitsBetween(baseSHA, headSHA)
	if err != nil {
		return nil, err
	}
	return convertCommits(entityCommits), nil
}

func (c *APIClient) GetFirstCommitOfBranch(ctx context.Context, branch, baseBranch string) (*Commit, error) {
	ec, err := c.api.GetFirstCommitOfBranch(branch, baseBranch)
	if err != nil {
		return nil, err
	}
	result := convertCommit(*ec)
	return &result, nil
}

func (c *APIClient) GetBranchCommitRange(ctx context.Context, branch string) (*BranchCommitRange, error) {
	ecr, err := c.api.GetBranchCommitRange(branch)
	if err != nil {
		return nil, err
	}
	return convertBranchCommitRange(ecr), nil
}

// -------------------------------------------------------------------
// FileReader
// -------------------------------------------------------------------

func (c *APIClient) GetFileContent(ctx context.Context, fileName string) ([]byte, error) {
	return c.api.GetFileContent(fileName)
}

func (c *APIClient) GetRepositoryContents(ctx context.Context, filepath, branch string) ([]FileInfo, error) {
	entityFiles, err := c.api.GetRepositoryContents(filepath, branch)
	if err != nil {
		return nil, err
	}
	return convertFileInfos(entityFiles), nil
}

func (c *APIClient) AnalyzeProjectStructure(ctx context.Context, branch string) ([]string, error) {
	return c.api.AnalyzeProjectStructure(branch)
}

// -------------------------------------------------------------------
// BranchManager
// -------------------------------------------------------------------

func (c *APIClient) GetBranches(ctx context.Context, repo string) ([]Branch, error) {
	entityBranches, err := c.api.GetBranches(repo)
	if err != nil {
		return nil, err
	}
	return convertBranches(entityBranches), nil
}

func (c *APIClient) CreateBranch(ctx context.Context, _, _ string) error {
	// entity API CreateTestBranch uses pre-configured branch names
	return c.api.CreateTestBranch()
}

func (c *APIClient) DeleteBranch(ctx context.Context, _ string) error {
	// entity API DeleteTestBranch uses pre-configured branch name
	return c.api.DeleteTestBranch()
}

// -------------------------------------------------------------------
// ReleaseReader
// -------------------------------------------------------------------

func (c *APIClient) GetLatestRelease(ctx context.Context) (*Release, error) {
	er, err := c.api.GetLatestRelease()
	if err != nil {
		return nil, err
	}
	result := convertRelease(*er)
	return &result, nil
}

func (c *APIClient) GetReleaseByTag(ctx context.Context, tag string) (*Release, error) {
	er, err := c.api.GetReleaseByTag(tag)
	if err != nil {
		return nil, err
	}
	result := convertRelease(*er)
	return &result, nil
}

// -------------------------------------------------------------------
// IssueManager
// -------------------------------------------------------------------

func (c *APIClient) GetIssue(ctx context.Context, issueNumber int64) (*Issue, error) {
	ei, err := c.api.GetIssue(issueNumber)
	if err != nil {
		return nil, err
	}
	result := convertIssue(*ei)
	return &result, nil
}

func (c *APIClient) AddIssueComment(ctx context.Context, issueNumber int64, commentText string) error {
	return c.api.AddIssueComment(issueNumber, commentText)
}

func (c *APIClient) CloseIssue(ctx context.Context, issueNumber int64) error {
	return c.api.CloseIssue(issueNumber)
}

// -------------------------------------------------------------------
// PRManager
// -------------------------------------------------------------------

func (c *APIClient) CreatePR(ctx context.Context, head string) (PR, error) {
	ep, err := c.api.CreatePR(head)
	if err != nil {
		return PR{}, err
	}
	return convertPR(ep), nil
}

func (c *APIClient) CreatePRWithOptions(ctx context.Context, opts CreatePROptions) (*PRResponse, error) {
	entityOpts := entity_gitea.CreatePROptions{
		Title:     opts.Title,
		Body:      opts.Body,
		Head:      opts.Head,
		Base:      opts.Base,
		Assignees: opts.Assignees,
		Labels:    opts.Labels,
	}
	er, err := c.api.CreatePRWithOptions(entityOpts)
	if err != nil {
		return nil, err
	}
	result := convertPRResponse(*er)
	return &result, nil
}

func (c *APIClient) MergePR(ctx context.Context, prNumber int64) error {
	return c.api.MergePR(prNumber, c.logger)
}

func (c *APIClient) ClosePR(ctx context.Context, prNumber int64) error {
	return c.api.ClosePR(prNumber)
}

// -------------------------------------------------------------------
// RepositoryWriter
// -------------------------------------------------------------------

func (c *APIClient) SetRepositoryState(ctx context.Context, operations []BatchOperation, branch, commitMessage string) error {
	entityOps := make([]entity_gitea.BatchOperation, len(operations))
	for i, op := range operations {
		entityOps[i] = entity_gitea.BatchOperation{
			Operation: op.Operation,
			Path:      op.Path,
			Content:   op.Content,
			SHA:       op.SHA,
			FromPath:  op.FromPath,
		}
	}
	return c.api.SetRepositoryState(c.logger, entityOps, branch, commitMessage)
}

// -------------------------------------------------------------------
// TeamReader
// -------------------------------------------------------------------

func (c *APIClient) IsUserInTeam(ctx context.Context, username, orgName, teamName string) (bool, error) {
	return c.api.IsUserInTeam(c.logger, username, orgName, teamName)
}

func (c *APIClient) GetTeamMembers(ctx context.Context, orgName, teamName string) ([]string, error) {
	return c.api.GetTeamMembers(orgName, teamName)
}

// -------------------------------------------------------------------
// OrgReader
// -------------------------------------------------------------------

func (c *APIClient) SearchOrgRepos(ctx context.Context, orgName string) ([]Repository, error) {
	entityRepos, err := c.api.SearchOrgRepos(orgName)
	if err != nil {
		return nil, err
	}
	return convertRepositories(entityRepos), nil
}

// -------------------------------------------------------------------
// Type conversion helpers
// -------------------------------------------------------------------

func convertPR(ep entity_gitea.PR) PR {
	return PR{
		ID:     ep.ID,
		Number: ep.Number,
		Base:   ep.Base,
		Head:   ep.Head,
	}
}

func convertCommit(ec entity_gitea.Commit) Commit {
	return Commit{
		SHA: ec.SHA,
		Commit: CommitDetails{
			Author: CommitAuthor{
				Name:  ec.Commit.Author.Name,
				Email: ec.Commit.Author.Email,
				Date:  ec.Commit.Author.Date,
			},
			Committer: CommitAuthor{
				Name:  ec.Commit.Committer.Name,
				Email: ec.Commit.Committer.Email,
				Date:  ec.Commit.Committer.Date,
			},
			Message: ec.Commit.Message,
		},
	}
}

func convertCommits(ecs []entity_gitea.Commit) []Commit {
	result := make([]Commit, len(ecs))
	for i, ec := range ecs {
		result[i] = convertCommit(ec)
	}
	return result
}

func convertCommitFiles(ecf []entity_gitea.CommitFile) []CommitFile {
	result := make([]CommitFile, len(ecf))
	for i, f := range ecf {
		result[i] = CommitFile{
			Filename: f.Filename,
			Status:   f.Status,
			Patch:    f.Patch,
		}
	}
	return result
}

func convertBranchCommitRange(ecr *entity_gitea.BranchCommitRange) *BranchCommitRange {
	result := &BranchCommitRange{}
	if ecr.FirstCommit != nil {
		fc := convertCommit(*ecr.FirstCommit)
		result.FirstCommit = &fc
	}
	if ecr.LastCommit != nil {
		lc := convertCommit(*ecr.LastCommit)
		result.LastCommit = &lc
	}
	return result
}

func convertFileInfos(efs []entity_gitea.FileInfo) []FileInfo {
	result := make([]FileInfo, len(efs))
	for i, f := range efs {
		result[i] = FileInfo{
			Name:        f.Name,
			Path:        f.Path,
			SHA:         f.SHA,
			Size:        f.Size,
			URL:         f.URL,
			HTMLURL:     f.HTMLURL,
			GitURL:      f.GitURL,
			DownloadURL: f.DownloadURL,
			Type:        f.Type,
			Content:     f.Content,
			Encoding:    f.Encoding,
			Target:      f.Target,
			Submodule:   f.Submodule,
		}
	}
	return result
}

func convertBranches(ebs []entity_gitea.Branch) []Branch {
	result := make([]Branch, len(ebs))
	for i, b := range ebs {
		result[i] = Branch{
			Label: b.Label,
			Name:  b.Name,
			Commit: BranchCommit{
				ID: b.Commit.ID,
			},
		}
	}
	return result
}

func convertRelease(er entity_gitea.Release) Release {
	assets := make([]ReleaseAsset, len(er.Assets))
	for i, a := range er.Assets {
		assets[i] = ReleaseAsset{
			ID:          a.ID,
			Name:        a.Name,
			Size:        a.Size,
			DownloadURL: a.DownloadURL,
		}
	}
	return Release{
		ID:          er.ID,
		TagName:     er.TagName,
		Name:        er.Name,
		Body:        er.Body,
		Assets:      assets,
		CreatedAt:   er.CreatedAt,
		PublishedAt: er.PublishedAt,
	}
}

func convertIssue(ei entity_gitea.Issue) Issue {
	return Issue{
		ID:     ei.ID,
		Number: ei.Number,
		Title:  ei.Title,
		Body:   ei.Body,
		State:  ei.State,
		User: IssueUser{
			Login: ei.User.Login,
			ID:    ei.User.ID,
		},
		CreatedAt: ei.CreatedAt,
		UpdatedAt: ei.UpdatedAt,
	}
}

func convertPRResponse(er entity_gitea.PRResponse) PRResponse {
	return PRResponse{
		ID:        er.ID,
		Number:    er.Number,
		HTMLURL:   er.HTMLURL,
		State:     er.State,
		Title:     er.Title,
		Body:      er.Body,
		Mergeable: er.Mergeable,
	}
}

func convertRepositories(ers []entity_gitea.Repository) []Repository {
	result := make([]Repository, len(ers))
	for i, r := range ers {
		result[i] = Repository{
			ID:       r.ID,
			Name:     r.Name,
			FullName: r.FullName,
			Owner: RepositoryOwner{
				ID:    r.Owner.ID,
				Login: r.Owner.Login,
				Type:  r.Owner.Type,
			},
			DefaultBranch: r.DefaultBranch,
			Private:       r.Private,
			Fork:          r.Fork,
		}
	}
	return result
}
