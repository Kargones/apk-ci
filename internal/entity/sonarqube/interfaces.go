// Package sonarqube provides interfaces for interacting with SonarQube API.
// This package defines the core interfaces that implement the SonarQube integration
// following SOLID principles and existing project patterns.
//
// The main interfaces are:
// - SonarQubeAPIInterface: Defines methods for interacting with SonarQube REST API
// - SonarScannerInterface: Defines methods for managing sonar-scanner executable
// - SQCommandHandlerInterface: Defines methods for handling SonarQube CLI commands
//
// These interfaces provide a clean abstraction layer between the application logic
// and the external SonarQube services, allowing for easy testing and future extensions.
package sonarqube

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"time"
)

// ProjectManager defines CRUD operations for SonarQube projects.
type ProjectManager interface {
	CreateProject(ctx context.Context, owner, repo, branch string) (*Project, error)
	GetProject(ctx context.Context, projectKey string) (*Project, error)
	UpdateProject(ctx context.Context, projectKey string, updates *ProjectUpdate) error
	DeleteProject(ctx context.Context, projectKey string) error
	ListProjects(ctx context.Context, owner, repo string) ([]Project, error)
	SetProjectTags(ctx context.Context, projectKey string, tags []string) error
}

// AnalysisReader defines read operations for analyses, metrics, issues, and quality data.
type AnalysisReader interface {
	GetAnalyses(ctx context.Context, projectKey string) ([]Analysis, error)
	GetAnalysisStatus(ctx context.Context, analysisID string) (*AnalysisStatus, error)
	GetIssues(ctx context.Context, projectKey string, params *IssueParams) ([]Issue, error)
	GetQualityGateStatus(ctx context.Context, projectKey string) (*QualityGateStatus, error)
	GetMetrics(ctx context.Context, projectKey string, metricKeys []string) (*Metrics, error)
	GetQualityProfiles(ctx context.Context, projectKey string) ([]QualityProfile, error)
	GetQualityGates(ctx context.Context) ([]QualityGate, error)
	GetRules(ctx context.Context, params *RuleParams) ([]Rule, error)
}

// AuthProvider defines authentication operations for SonarQube.
type AuthProvider interface {
	Authenticate(token string) error
	ValidateToken(ctx context.Context) error
}

// APIInterface defines methods for interacting with SonarQube REST API.
// This interface provides a complete set of operations for managing SonarQube projects,
// performing authentication, retrieving analyses and metrics, and handling issues.
type APIInterface interface {
	ProjectManager
	AnalysisReader
	AuthProvider
}

// SonarScannerInterface defines methods for managing sonar-scanner executable.
// This interface provides operations for downloading, configuring, and executing
// the sonar-scanner tool for code analysis.
type SonarScannerInterface interface {
	// Scanner Management methods
	// Download clones the sonar-scanner repository from the specified URL.
	// Returns the path to the cloned directory and an error if the clone fails.
	Download(ctx context.Context, scannerURL string, scannerVersion string) (string, error)

	// Configure configures the sonar-scanner with the provided configuration.
	// Returns an error if the configuration is invalid or cannot be applied.
	Configure(config *ScannerConfig) error

	// Execute executes the sonar-scanner with the provided context.
	// Returns the ScanResult or an error if the execution fails.
	Execute(ctx context.Context) (*ScanResult, error)

	// Configuration methods
	// SetProperty sets a property in the scanner configuration.
	SetProperty(key, value string)

	// GetProperty retrieves a property from the scanner configuration.
	// Returns the property value.
	GetProperty(key string) string

	// ValidateConfig validates the current scanner configuration.
	// Returns an error if the configuration is invalid.
	ValidateConfig() error

	// Lifecycle methods
	// Initialize initializes the scanner, preparing it for execution.
	// Returns an error if initialization fails.
	Initialize() error

	// Cleanup cleans up resources used by the scanner.
	// Returns an error if cleanup fails.
	Cleanup() error
}

// SQCommandHandlerInterface defines methods for handling SonarQube CLI commands.
// This interface provides a unified entry point for all SonarQube-related commands
// that can be executed through the CLI.
type SQCommandHandlerInterface interface {
	// Branch Operations methods
	// HandleSQScanBranch handles the sq-scan-branch command with the provided parameters.
	// Returns an error if the operation fails.
	HandleSQScanBranch(ctx context.Context, params *ScanBranchParams) error

	// HandleSQScanPR handles the sq-scan-pr command with the provided parameters.
	// Returns an error if the operation fails.
	HandleSQScanPR(ctx context.Context, params *ScanPRParams) error

	// Project Operations methods
	// HandleSQProjectUpdate handles the sq-project-update command with the provided parameters.
	// Returns an error if the operation fails.
	HandleSQProjectUpdate(ctx context.Context, params *ProjectUpdateParams) error

	// HandleSQRepoSync handles the sq-repo-sync command with the provided parameters.
	// Returns an error if the operation fails.
	HandleSQRepoSync(ctx context.Context, params *RepoSyncParams) error

	// HandleSQRepoClear handles the sq-repo-clear command with the provided parameters.
	// Returns an error if the operation fails.
	HandleSQRepoClear(ctx context.Context, params *RepoClearParams) error

	// Reporting Operations methods
	// HandleSQReportPR handles the sq-report-pr command with the provided parameters.
	// Returns an error if the operation fails.
	HandleSQReportPR(ctx context.Context, params *ReportPRParams) error

	// HandleSQReportBranch handles the sq-report-branch command with the provided parameters.
	// Returns an error if the operation fails.
	HandleSQReportBranch(ctx context.Context, params *ReportBranchParams) error

	// HandleSQReportProject handles the sq-report-project command with the provided parameters.
	// Returns an error if the operation fails.
	HandleSQReportProject(ctx context.Context, params *ReportProjectParams) error
}

// Time represents a time value from SonarQube API with custom parsing.
type Time struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for SonarQube time format.
func (st *Time) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)
	if str == "null" || str == "" {
		return nil
	}

	// Try different time formats used by SonarQube
	formats := []string{
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05+0700",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, str); err == nil {
			st.Time = t
			return nil
		}
	}

	return &json.UnmarshalTypeError{
		Value: "string",
		Type:  reflect.TypeOf(time.Time{}),
	}
}

// Project represents a project in SonarQube.
type Project struct {
	Key              string            `json:"key"`
	Name             string            `json:"name"`
	Description      string            `json:"description"`
	Qualifier        string            `json:"qualifier"`
	Visibility       string            `json:"visibility"`
	LastAnalysisDate *Time    `json:"lastAnalysisDate,omitempty"`
	Managed          bool              `json:"managed"`
	Tags             []string          `json:"tags"`
	Created          time.Time         `json:"created"`
	Updated          time.Time         `json:"updated"`
	Metadata         map[string]string `json:"metadata"`
}

// Analysis represents an analysis of code in SonarQube.
type Analysis struct {
	ID         string            `json:"id"`
	ProjectKey string            `json:"projectKey"`
	Date       *Time             `json:"date"`
	Revision   string            `json:"revision"`
	Status     AnalysisStatus    `json:"status"`
	Metrics    map[string]string `json:"metrics"`
}

// ScanResult represents the result of a sonar-scanner execution.
type ScanResult struct {
	Success    bool              `json:"success"`
	AnalysisID string            `json:"analysisId"`
	ProjectKey string            `json:"projectKey"`
	Duration   time.Duration     `json:"duration"`
	Issues     []Issue           `json:"issues"`
	Metrics    map[string]string `json:"metrics"`
	Errors     []string          `json:"errors"`
}

// Issue represents a code quality issue in SonarQube.
type Issue struct {
	Key       string    `json:"key"`
	Rule      string    `json:"rule"`
	Severity  string    `json:"severity"`
	Component string    `json:"component"`
	Line      int       `json:"line"`
	Message   string    `json:"message"`
	Type      string    `json:"type"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ProjectUpdate represents updates to apply to a project.
type ProjectUpdate struct {
	Name        string
	Description string
	Visibility  string
	Tags        []string
}

// IssueParams represents parameters for retrieving issues.
type IssueParams struct {
	ComponentKeys []string
	Rules         []string
	Severities    []string
	Statuses      []string
}

// AnalysisStatus represents the status of an analysis.
type AnalysisStatus struct {
	Status string
}

// QualityGateStatus represents the status of a quality gate.
type QualityGateStatus struct {
	Status string
}

// Metrics represents metrics retrieved from SonarQube.
type Metrics struct {
	Metrics map[string]float64
}

// QualityProfile represents a quality profile in SonarQube.
type QualityProfile struct {
	Key       string `json:"key"`
	Name      string `json:"name"`
	Language  string `json:"language"`
	IsDefault bool   `json:"isDefault"`
	Rules     []Rule `json:"rules"`
}

// QualityGate represents a quality gate in SonarQube.
type QualityGate struct {
	ID         int                `json:"id"`
	Name       string             `json:"name"`
	IsDefault  bool               `json:"isDefault"`
	Conditions []QualityCondition `json:"conditions"`
}

// QualityCondition represents a condition in a quality gate.
type QualityCondition struct {
	ID        int    `json:"id"`
	Metric    string `json:"metric"`
	Operator  string `json:"op"`
	Threshold string `json:"threshold"`
}

// Rule represents a rule in SonarQube.
type Rule struct {
	Key        string      `json:"key"`
	Name       string      `json:"name"`
	Severity   string      `json:"severity"`
	Language   string      `json:"lang"`
	Repository string      `json:"repo"`
	Type       string      `json:"type"`
	Tags       []string    `json:"tags"`
	Params     []RuleParam `json:"params"`
}

// RuleParam represents a parameter of a rule.
type RuleParam struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// RuleParams represents parameters for retrieving rules.
type RuleParams struct {
	Repositories []string
	Languages    []string
	Tags         []string
	IsActive     *bool
}

// ScannerConfig represents configuration for sonar-scanner.
type ScannerConfig struct {
	ScannerURL     string            `yaml:"scannerUrl"`
	ScannerVersion string            `yaml:"scannerVersion"`
	JavaOpts       string            `yaml:"javaOpts"`
	Properties     map[string]string `yaml:"properties"`
	Timeout        time.Duration     `yaml:"timeout"`
	WorkDir        string            `yaml:"workDir"`
	TempDir        string            `yaml:"tempDir"`
}

// ScanBranchParams represents parameters for the sq-scan-branch command.
type ScanBranchParams struct {
	Owner      string
	Repo       string
	Branch     string
	CommitHash string
	SourceDir  string
}

// ScanPRParams represents parameters for the sq-scan-pr command.
type ScanPRParams struct {
	Owner string
	Repo  string
	PR    int
}

// ProjectUpdateParams represents parameters for the sq-project-update command.
type ProjectUpdateParams struct {
	Owner string
	Repo  string
}

// RepoSyncParams represents parameters for the sq-repo-sync command.
type RepoSyncParams struct {
	Owner string
	Repo  string
}

// RepoClearParams represents parameters for the sq-repo-clear command.
type RepoClearParams struct {
	Owner string
	Repo  string
	Force bool
}

// ReportPRParams represents parameters for the sq-report-pr command.
type ReportPRParams struct {
	Owner string
	Repo  string
	PR    int
}

// ReportBranchParams represents parameters for the sq-report-branch command.
type ReportBranchParams struct {
	Owner           string
	Repo            string
	Branch          string
	FirstCommitHash string
	LastCommitHash  string
}

// ReportProjectParams represents parameters for the sq-report-project command.
type ReportProjectParams struct {
	Owner string
	Repo  string
}
