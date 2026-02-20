// Package git2storehandler содержит тесты для NR-команды nr-git2store.
package git2storehandler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Kargones/apk-ci/internal/command"
	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/Kargones/apk-ci/internal/pkg/output"
)

// mockGitOperator — мок для GitOperator.
type mockGitOperator struct {
	cloneFunc     func(ctx context.Context, l *slog.Logger) error
	switchFunc    func(ctx context.Context, l *slog.Logger) error
	setBranchFunc func(branch string)
	branch        string
}

func (m *mockGitOperator) Clone(ctx context.Context, l *slog.Logger) error {
	if m.cloneFunc != nil {
		return m.cloneFunc(ctx, l)
	}
	return nil
}

func (m *mockGitOperator) Switch(ctx context.Context, l *slog.Logger) error {
	if m.switchFunc != nil {
		return m.switchFunc(ctx, l)
	}
	return nil
}

func (m *mockGitOperator) SetBranch(branch string) {
	m.branch = branch
	if m.setBranchFunc != nil {
		m.setBranchFunc(branch)
	}
}

// mockGitFactory — мок для GitFactory.
type mockGitFactory struct {
	gitOp *mockGitOperator
	err   error
}

func (f *mockGitFactory) CreateGit(l *slog.Logger, cfg *config.Config) (GitOperator, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.gitOp, nil
}

// mockConvertConfigOperator — мок для ConvertConfigOperator.
//nolint:dupl // similar test structure
type mockConvertConfigOperator struct {
	loadFunc        func(ctx context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error
	initDbFunc      func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	storeUnBindFunc func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	loadDbFunc      func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	dbUpdateFunc    func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	dumpDbFunc      func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	storeBindFunc   func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	storeLockFunc   func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	mergeFunc       func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	storeCommitFunc func(ctx context.Context, l *slog.Logger, cfg *config.Config) error
	setOneDBFunc    func(dbConnectString, user, pass string)
}

func (m *mockConvertConfigOperator) Load(ctx context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error {
	if m.loadFunc != nil {
		return m.loadFunc(ctx, l, cfg, infobaseName)
	}
	return nil
}

func (m *mockConvertConfigOperator) InitDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.initDbFunc != nil {
		return m.initDbFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) StoreUnBind(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.storeUnBindFunc != nil {
		return m.storeUnBindFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) LoadDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.loadDbFunc != nil {
		return m.loadDbFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) DbUpdate(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.dbUpdateFunc != nil {
		return m.dbUpdateFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) DumpDb(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.dumpDbFunc != nil {
		return m.dumpDbFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) StoreBind(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.storeBindFunc != nil {
		return m.storeBindFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) StoreLock(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.storeLockFunc != nil {
		return m.storeLockFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) Merge(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.mergeFunc != nil {
		return m.mergeFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) StoreCommit(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
	if m.storeCommitFunc != nil {
		return m.storeCommitFunc(ctx, l, cfg)
	}
	return nil
}

func (m *mockConvertConfigOperator) SetOneDB(dbConnectString, user, pass string) {
	if m.setOneDBFunc != nil {
		m.setOneDBFunc(dbConnectString, user, pass)
	}
}

// mockConvertConfigFactory — мок для ConvertConfigFactory.
type mockConvertConfigFactory struct {
	ccOp *mockConvertConfigOperator
}

func (f *mockConvertConfigFactory) CreateConvertConfig() ConvertConfigOperator {
	return f.ccOp
}

// mockBackupCreator — мок для BackupCreator.
type mockBackupCreator struct {
	createBackupFunc func(cfg *config.Config, storeRoot string) (string, error)
}

func (m *mockBackupCreator) CreateBackup(cfg *config.Config, storeRoot string) (string, error) {
	if m.createBackupFunc != nil {
		return m.createBackupFunc(cfg, storeRoot)
	}
	return "/tmp/backup_test", nil
}

// mockTempDbCreator — мок для TempDbCreator.
type mockTempDbCreator struct {
	createTempDbFunc func(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error)
}

func (m *mockTempDbCreator) CreateTempDb(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	if m.createTempDbFunc != nil {
		return m.createTempDbFunc(ctx, l, cfg)
	}
	return "/F /tmp/temp_db", nil
}

// createTestConfig создаёт тестовую конфигурацию.
func createTestConfig(t *testing.T) *config.Config {
	t.Helper()
	tmpDir := t.TempDir()
	appCfg := &config.AppConfig{} //nolint:goconst // test value
	appCfg.Paths.Bin1cv8 = "/opt/1cv8/1cv8" //nolint:goconst // test value
	return &config.Config{
		Owner:        "test-owner",
		Repo:         "test-repo",
		TmpDir:       tmpDir,
		WorkDir:      tmpDir,
		InfobaseName: "TestDB",
		AppConfig:    appCfg,
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "TestStoreDb",
		},
		SecretConfig: &config.SecretConfig{},
	}
}

// TestGit2StoreHandler_Name проверяет имя команды (AC-1).
func TestGit2StoreHandler_Name(t *testing.T) {
	h := &Git2StoreHandler{}
	got := h.Name()
	want := constants.ActNRGit2store

	if got != want {
		t.Errorf("Name() = %q, want %q", got, want)
	}
}

// TestGit2StoreHandler_Description проверяет описание команды.
func TestGit2StoreHandler_Description(t *testing.T) {
	h := &Git2StoreHandler{}
	got := h.Description()

	if got == "" {
		t.Error("Description() вернул пустую строку")
	}
	if !strings.Contains(got, "Git") || !strings.Contains(got, "хранилище") {
		t.Errorf("Description() = %q, должен содержать 'Git' и 'хранилище'", got)
	}
}

// TestGit2StoreHandler_CompileTimeCheck проверяет compile-time interface check (AC-1).
func TestGit2StoreHandler_CompileTimeCheck(t *testing.T) {
	// Эта проверка гарантирует, что Git2StoreHandler реализует command.Handler
	var _ command.Handler = (*Git2StoreHandler)(nil)
}

// TestGit2StoreHandler_Execute_ValidationErrors проверяет валидацию конфигурации (AC-7, AC-9).
func TestGit2StoreHandler_Execute_ValidationErrors(t *testing.T) {
	tests := []struct {
		name        string
		cfg         *config.Config
		wantErrCode string
	}{
		{
			name:        "nil конфигурация",
			cfg:         nil,
			wantErrCode: "CONFIG.MISSING",
		},
		{
			name: "отсутствует Bin1cv8",
			cfg: &config.Config{
				Owner:     "owner",
				Repo:      "repo",
				TmpDir:    "/tmp",
				WorkDir:   "/tmp",
				AppConfig: &config.AppConfig{},
			},
			wantErrCode: "CONFIG.BIN1CV8_MISSING",
		},
		{
			name: "отсутствует TmpDir",
			cfg: func() *config.Config {
				appCfg := &config.AppConfig{}
				appCfg.Paths.Bin1cv8 = "/opt/1cv8/1cv8"
				return &config.Config{
					Owner:     "owner",
					Repo:      "repo",
					WorkDir:   "/tmp",
					AppConfig: appCfg,
				}
			}(),
			wantErrCode: "CONFIG.TMPDIR_MISSING",
		},
		{
			name: "отсутствует Owner",
			cfg: func() *config.Config {
				appCfg := &config.AppConfig{}
				appCfg.Paths.Bin1cv8 = "/opt/1cv8/1cv8"
				return &config.Config{
					Repo:      "repo",
					TmpDir:    "/tmp",
					WorkDir:   "/tmp",
					AppConfig: appCfg,
				}
			}(),
			wantErrCode: "CONFIG.OWNER_MISSING",
		},
		{
			name: "отсутствует Repo",
			cfg: func() *config.Config {
				appCfg := &config.AppConfig{}
				appCfg.Paths.Bin1cv8 = "/opt/1cv8/1cv8"
				return &config.Config{
					Owner:     "owner",
					TmpDir:    "/tmp",
					WorkDir:   "/tmp",
					AppConfig: appCfg,
				}
			}(),
			wantErrCode: "CONFIG.REPO_MISSING",
		},
		{
			name: "отсутствует WorkDir",
			cfg: func() *config.Config {
				appCfg := &config.AppConfig{}
				appCfg.Paths.Bin1cv8 = "/opt/1cv8/1cv8"
				return &config.Config{
					Owner:     "owner",
					Repo:      "repo",
					TmpDir:    "/tmp",
					AppConfig: appCfg,
				}
			}(),
			wantErrCode: "CONFIG.WORKDIR_MISSING",
		},
		{
			name: "отсутствует InfobaseName",
			cfg: func() *config.Config {
				appCfg := &config.AppConfig{}
				appCfg.Paths.Bin1cv8 = "/opt/1cv8/1cv8"
				return &config.Config{
					Owner:     "owner",
					Repo:      "repo",
					TmpDir:    "/tmp",
					WorkDir:   "/tmp",
					AppConfig: appCfg,
					// InfobaseName не указан
				}
			}(),
			wantErrCode: "CONFIG.INFOBASE_MISSING",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &Git2StoreHandler{}
			ctx := context.Background()

			err := h.Execute(ctx, tt.cfg)

			if err == nil {
				t.Error("Execute() должен вернуть ошибку")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrCode) {
				t.Errorf("Execute() error = %v, want error containing %q", err, tt.wantErrCode)
			}
		})
	}
}

// TestGit2StoreHandler_Execute_SuccessCase проверяет успешное выполнение (AC-1, AC-2, AC-12).
func TestGit2StoreHandler_Execute_SuccessCase(t *testing.T) {
	cfg := createTestConfig(t)

	// Создаём моки
	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	// Перенаправляем stdout для проверки вывода
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("Execute() вернул ошибку: %v", err)
	}

	// Проверяем text output (AC-5)
	output := buf.String()
	if !strings.Contains(output, "успешно") {
		t.Errorf("Text output должен содержать 'успешно', got: %s", output)
	}
}

// TestGit2StoreHandler_Execute_JSONOutput проверяет JSON output (AC-4).
func TestGit2StoreHandler_Execute_JSONOutput(t *testing.T) {
	cfg := createTestConfig(t)

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	// Устанавливаем JSON формат
	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT") //nolint:errcheck

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("Execute() вернул ошибку: %v", err)
	}

	// Проверяем JSON структуру (AC-4)
	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("Не удалось распарсить JSON: %v, output: %s", jsonErr, buf.String())
		return
	}

	if result.Status != output.StatusSuccess {
		t.Errorf("JSON status = %q, want %q", result.Status, output.StatusSuccess)
	}

	if result.Command != constants.ActNRGit2store {
		t.Errorf("JSON command = %q, want %q", result.Command, constants.ActNRGit2store)
	}

	// Проверяем data (AC-4)
	dataBytes, _ := json.Marshal(result.Data)
	var data Git2StoreData
	if jsonErr := json.Unmarshal(dataBytes, &data); jsonErr != nil {
		t.Errorf("Не удалось распарсить data: %v", jsonErr)
		return
	}

	if !data.StateChanged {
		t.Error("data.StateChanged должен быть true при успешном выполнении (AC-12)")
	}

	if data.BackupPath == "" {
		t.Error("data.BackupPath должен быть заполнен (AC-8)")
	}

	if len(data.StagesCompleted) == 0 {
		t.Error("data.StagesCompleted должен содержать этапы (AC-3)")
	}
}

// TestGit2StoreHandler_Execute_DryRun проверяет dry-run режим (AC-14).
func TestGit2StoreHandler_Execute_DryRun(t *testing.T) {
	cfg := createTestConfig(t)

	h := &Git2StoreHandler{}

	_ = os.Setenv("BR_DRY_RUN", "true")
	defer os.Unsetenv("BR_DRY_RUN") //nolint:errcheck

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("Execute() в dry-run режиме вернул ошибку: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Dry-run") && !strings.Contains(output, "План") {
		t.Errorf("Dry-run output должен содержать план, got: %s", output)
	}

	// Проверяем, что выводятся этапы
	for _, stage := range allStages[:5] { // Проверяем первые 5 этапов
		if !strings.Contains(output, stage) {
			t.Errorf("Dry-run output должен содержать этап %q", stage)
		}
	}
}

// TestGit2StoreHandler_Execute_StageErrors проверяет обработку ошибок на этапах (AC-7, AC-11).
func TestGit2StoreHandler_Execute_StageErrors(t *testing.T) {
	tests := []struct {
		name        string
		setupMocks  func() (*Git2StoreHandler, *config.Config)
		wantErrCode string
		wantStage   string
	}{
		{
			name: "ошибка backup",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				return &Git2StoreHandler{
					backupCreator: &mockBackupCreator{
						createBackupFunc: func(cfg *config.Config, storeRoot string) (string, error) {
							return "", errors.New("backup failed")
						},
					},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_BACKUP",
			wantStage:   StageCreatingBackup,
		},
		{
			name: "ошибка clone",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{
					cloneFunc: func(ctx context.Context, l *slog.Logger) error {
						return errors.New("clone failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator: &mockBackupCreator{},
					gitFactory:    &mockGitFactory{gitOp: gitOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_CLONE",
			wantStage:   StageCloning,
		},
		{
			name: "ошибка checkout edt",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{
					switchFunc: func(ctx context.Context, l *slog.Logger) error {
						return errors.New("switch failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator: &mockBackupCreator{},
					gitFactory:    &mockGitFactory{gitOp: gitOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_CHECKOUT",
			wantStage:   StageCheckoutEdt,
		},
		{
			name: "ошибка load config",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					loadFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error {
						return errors.New("load failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_LOAD",
			wantStage:   StageLoadingConfig,
		},
		{
			name: "ошибка init_db",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					initDbFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
						return errors.New("init db failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_INIT_DB",
			wantStage:   StageInitDb,
		},
		{
			name: "ошибка unbind",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					storeUnBindFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
						return errors.New("unbind failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_UNBIND",
			wantStage:   StageUnbinding,
		},
		{
			name: "ошибка load_db",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					loadDbFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
						return errors.New("load db failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_LOAD_DB",
			wantStage:   StageLoadingDb,
		},
		{
			name: "ошибка dump_db",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					dumpDbFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
						return errors.New("dump db failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_DUMP",
			wantStage:   StageDumpingDb,
		},
		{
			name: "ошибка bind",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					storeBindFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
						return errors.New("bind failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_BIND",
			wantStage:   StageBinding,
		},
		{
			name: "ошибка lock",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					storeLockFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
						return errors.New("lock failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_LOCK",
			wantStage:   StageLocking,
		},
		{
			name: "ошибка merge",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					mergeFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
						return errors.New("merge failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_MERGE",
			wantStage:   StageMerging,
		},
		{
			name: "ошибка commit",
			setupMocks: func() (*Git2StoreHandler, *config.Config) {
				cfg := createTestConfig(t)
				gitOp := &mockGitOperator{}
				ccOp := &mockConvertConfigOperator{
					storeCommitFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
						return errors.New("commit failed")
					},
				}
				return &Git2StoreHandler{
					backupCreator:        &mockBackupCreator{},
					gitFactory:           &mockGitFactory{gitOp: gitOp},
					convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
				}, cfg
			},
			wantErrCode: "ERR_GIT2STORE_COMMIT",
			wantStage:   StageCommitting,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h, cfg := tt.setupMocks()
			ctx := context.Background()

			err := h.Execute(ctx, cfg)

			if err == nil {
				t.Error("Execute() должен вернуть ошибку")
				return
			}

			if !strings.Contains(err.Error(), tt.wantErrCode) {
				t.Errorf("Execute() error = %v, want error containing %q", err, tt.wantErrCode)
			}
		})
	}
}

// TestGit2StoreHandler_Execute_BackupIncludedInError проверяет включение backup path в ошибку (AC-7).
func TestGit2StoreHandler_Execute_BackupIncludedInError(t *testing.T) {
	cfg := createTestConfig(t)

	backupPath := "/tmp/backup_20260101_120000"
	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{
		storeCommitFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
			return errors.New("commit failed")
		},
	}

	h := &Git2StoreHandler{
		backupCreator: &mockBackupCreator{
			createBackupFunc: func(cfg *config.Config, storeRoot string) (string, error) {
				return backupPath, nil
			},
		},
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
	}

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Fatal("Execute() должен вернуть ошибку")
	}

	// Проверяем, что backup path включён в ошибку (AC-7)
	if !strings.Contains(err.Error(), "backup") || !strings.Contains(err.Error(), backupPath) {
		t.Errorf("Error должна содержать backup path, got: %v", err)
	}
}

// TestGit2StoreHandler_Execute_ProgressLogs проверяет логирование прогресса (AC-3).
func TestGit2StoreHandler_Execute_ProgressLogs(t *testing.T) {
	cfg := createTestConfig(t)

	// Собираем логи
	var logBuf bytes.Buffer
	handler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelInfo})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	_ = h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout
	io.Copy(io.Discard, r)

	logs := logBuf.String()

	// Проверяем, что логируются основные этапы (AC-3)
	expectedStages := []string{
		StageValidating,
		StageCreatingBackup,
		StageCloning,
	}

	for _, stage := range expectedStages {
		if !strings.Contains(logs, stage) {
			t.Errorf("Логи должны содержать этап %q, logs: %s", stage, logs)
		}
	}
}

// TestGit2StoreHandler_Execute_WithExtensions проверяет работу с расширениями (AC-13).
func TestGit2StoreHandler_Execute_WithExtensions(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.AddArray = []string{"Extension1", "Extension2"}

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout
	io.Copy(io.Discard, r)

	if err != nil {
		t.Errorf("Execute() с расширениями вернул ошибку: %v", err)
	}
}

// TestGit2StoreHandler_DeprecatedAlias проверяет deprecated alias (AC-6).
func TestGit2StoreHandler_DeprecatedAlias(t *testing.T) {
	// Проверяем, что handler зарегистрирован под deprecated alias
	handler, ok := command.Get(constants.ActGit2store)
	if !ok {
		t.Error("Handler не зарегистрирован под deprecated alias 'git2store'")
		return
	}

	// Deprecated alias возвращает DeprecatedBridge с оригинальным именем команды
	// что является правильным поведением для вывода deprecation warning
	if handler == nil {
		t.Error("Handler is nil")
		return
	}

	// Проверяем, что Name() возвращает deprecated имя (для DeprecatedBridge)
	// DeprecatedBridge.Name() возвращает deprecated имя для логирования
	name := handler.Name()
	// Должно быть либо "git2store" (deprecated bridge) либо "nr-git2store" (direct)
	if name != constants.ActGit2store && name != constants.ActNRGit2store {
		t.Errorf("Handler.Name() = %q, ожидается %q или %q", name, constants.ActGit2store, constants.ActNRGit2store)
	}

	// Проверяем Description
	desc := handler.Description()
	if desc == "" {
		t.Error("Handler.Description() вернул пустую строку")
	}
}

// TestGit2StoreData_writeText проверяет текстовый вывод (AC-5).
func TestGit2StoreData_writeText(t *testing.T) {
	data := &Git2StoreData{
		StateChanged: true,
		StagesCompleted: []StageResult{
			{Name: StageValidating, Success: true, DurationMs: 10},
			{Name: StageCreatingBackup, Success: true, DurationMs: 100},
			{Name: StageCloning, Success: false, DurationMs: 5000, Error: "clone error"},
		},
		StageCurrent: StageCloning,
		BackupPath:   "/tmp/backup_test",
		DurationMs:   5110,
	}

	var buf bytes.Buffer
	err := data.writeText(&buf)
	if err != nil {
		t.Errorf("writeText() вернул ошибку: %v", err)
	}

	output := buf.String()

	// Проверяем содержимое
	expectedContents := []string{
		"успешно",
		"/tmp/backup_test",
		StageCloning,
		"✓", // успешные этапы
		"✗", // неуспешный этап
		"clone error",
	}

	for _, expected := range expectedContents {
		if !strings.Contains(output, expected) {
			t.Errorf("writeText() output должен содержать %q, got: %s", expected, output)
		}
	}
}

// TestStageConstants проверяет количество этапов (AC-2, AC-3).
func TestStageConstants(t *testing.T) {
	// AC-2, AC-3: должно быть 14+ этапов
	minStages := 14
	if len(allStages) < minStages {
		t.Errorf("Количество этапов = %d, должно быть >= %d (AC-3)", len(allStages), minStages)
	}

	// Проверяем уникальность этапов
	seen := make(map[string]bool)
	for _, stage := range allStages {
		if seen[stage] {
			t.Errorf("Дублирующийся этап: %q", stage)
		}
		seen[stage] = true
	}
}

// TestGit2StoreHandler_Execute_TempDbForLocalBase проверяет создание временной БД для LocalBase.
func TestGit2StoreHandler_Execute_TempDbForLocalBase(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.ProjectConfig.StoreDb = constants.LocalBase

	tempDbCalled := false
	setOneDBCalled := false

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{
		setOneDBFunc: func(dbConnectString, user, pass string) {
			setOneDBCalled = true
		},
	}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
		tempDbCreator: &mockTempDbCreator{
			createTempDbFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
				tempDbCalled = true
				return "/F /tmp/temp_db", nil
			},
		},
	}

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout
	io.Copy(io.Discard, r)

	if err != nil {
		t.Errorf("Execute() вернул ошибку: %v", err)
	}

	if !tempDbCalled {
		t.Error("createTempDb должен быть вызван для LocalBase")
	}

	if !setOneDBCalled {
		t.Error("SetOneDB должен быть вызван для LocalBase")
	}
}

// TestGit2StoreHandler_Execute_TempDbError проверяет ошибку создания временной БД.
func TestGit2StoreHandler_Execute_TempDbError(t *testing.T) {
	cfg := createTestConfig(t)
	cfg.ProjectConfig.StoreDb = constants.LocalBase

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
		tempDbCreator: &mockTempDbCreator{
			createTempDbFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
				return "", errors.New("temp db creation failed")
			},
		},
	}

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Error("Execute() должен вернуть ошибку при сбое создания временной БД")
		return
	}

	if !strings.Contains(err.Error(), "ERR_GIT2STORE_TEMP_DB") {
		t.Errorf("Execute() error = %v, want error containing 'ERR_GIT2STORE_TEMP_DB'", err)
	}
}

// TestGit2StoreHandler_JSONErrorOutput проверяет JSON вывод при ошибке (AC-11).
func TestGit2StoreHandler_JSONErrorOutput(t *testing.T) {
	cfg := createTestConfig(t)

	h := &Git2StoreHandler{
		backupCreator: &mockBackupCreator{
			createBackupFunc: func(cfg *config.Config, storeRoot string) (string, error) {
				return "", errors.New("backup creation failed")
			},
		},
	}

	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT") //nolint:errcheck

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	_ = h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	// Проверяем JSON структуру ошибки (AC-11)
	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("Не удалось распарсить JSON ошибки: %v, output: %s", jsonErr, buf.String())
		return
	}

	if result.Status != output.StatusError {
		t.Errorf("JSON status = %q, want %q", result.Status, output.StatusError)
	}

	if result.Error == nil {
		t.Error("JSON error должен быть заполнен")
		return
	}

	if !strings.Contains(result.Error.Code, "ERR_GIT2STORE") {
		t.Errorf("JSON error.code = %q, want code containing 'ERR_GIT2STORE'", result.Error.Code)
	}
}

// TestCreateBackupProduction проверяет production реализацию backup.
func TestCreateBackupProduction(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		TmpDir: tmpDir,
	}
	storeRoot := "tcp://test/owner/repo"

	backupPath, err := createBackupProduction(cfg, storeRoot)
	if err != nil {
		t.Errorf("createBackupProduction() вернул ошибку: %v", err)
		return
	}

	if backupPath == "" {
		t.Error("createBackupProduction() вернул пустой путь")
	}

	// Проверяем, что директория создана
	if _, statErr := os.Stat(backupPath); os.IsNotExist(statErr) {
		t.Errorf("Директория backup не создана: %s", backupPath)
	}

	// Проверяем, что файл info создан
	infoPath := backupPath + "/backup_info.txt"
	if _, statErr := os.Stat(infoPath); os.IsNotExist(statErr) {
		t.Errorf("Файл backup_info.txt не создан: %s", infoPath)
	}
}

// TestGit2StoreHandler_Execute_DurationMs проверяет корректность duration_ms.
func TestGit2StoreHandler_Execute_DurationMs(t *testing.T) {
	cfg := createTestConfig(t)

	gitOp := &mockGitOperator{}
	// Добавляем небольшую задержку в mock для проверки duration_ms
	ccOp := &mockConvertConfigOperator{
		loadFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error {
			time.Sleep(5 * time.Millisecond)
			return nil
		},
	}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT") //nolint:errcheck

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	startTime := time.Now()
	ctx := context.Background()
	_ = h.Execute(ctx, cfg)
	elapsed := time.Since(startTime).Milliseconds()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("Не удалось распарсить JSON: %v", jsonErr)
		return
	}

	if result.Metadata == nil {
		t.Error("Metadata должен быть заполнен")
		return
	}

	// duration_ms должен быть >= 0 (быстрые операции могут быть < 1ms)
	if result.Metadata.DurationMs < 0 {
		t.Errorf("duration_ms не должен быть отрицательным, got: %d", result.Metadata.DurationMs)
	}

	if result.Metadata.DurationMs > elapsed+100 { // +100мс на погрешность
		t.Errorf("duration_ms = %d превышает фактическое время выполнения %d мс", result.Metadata.DurationMs, elapsed)
	}
}

// TestGit2StoreHandler_Execute_UpdateDbError проверяет ошибку обновления БД (AC-9).
func TestGit2StoreHandler_Execute_UpdateDbError(t *testing.T) {
	cfg := createTestConfig(t)

	updateCallCount := 0
	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{
		dbUpdateFunc: func(ctx context.Context, l *slog.Logger, cfg *config.Config) error {
			updateCallCount++
			if updateCallCount == 1 {
				return errors.New("update db failed")
			}
			return nil
		},
	}

	h := &Git2StoreHandler{
		backupCreator:        &mockBackupCreator{},
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
	}

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	if err == nil {
		t.Error("Execute() должен вернуть ошибку при сбое updating_db_1")
		return
	}

	if !strings.Contains(err.Error(), "ERR_GIT2STORE_UPDATE") {
		t.Errorf("Execute() error = %v, want error containing 'ERR_GIT2STORE_UPDATE'", err)
	}
}

// TestGit2StoreHandler_Execute_JSONOutput_ErrorsField проверяет что errors: [] присутствует при успехе (AC-4).
func TestGit2StoreHandler_Execute_JSONOutput_ErrorsField(t *testing.T) {
	cfg := createTestConfig(t)

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	_ = os.Setenv("BR_OUTPUT_FORMAT", "json")
	defer os.Unsetenv("BR_OUTPUT_FORMAT") //nolint:errcheck

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("Execute() вернул ошибку: %v", err)
		return
	}

	// Проверяем что JSON содержит "errors":[]
	jsonOutput := buf.String()
	if !strings.Contains(jsonOutput, `"errors":[]`) && !strings.Contains(jsonOutput, `"errors": []`) {
		t.Errorf("JSON output должен содержать 'errors':[], got: %s", jsonOutput)
	}
}

// TestGit2StoreHandler_Execute_InvalidTimeout проверяет warning при невалидном BR_GIT2STORE_TIMEOUT (M-2).
func TestGit2StoreHandler_Execute_InvalidTimeout(t *testing.T) {
	cfg := createTestConfig(t)

	// Собираем логи
	var logBuf bytes.Buffer
	handler := slog.NewTextHandler(&logBuf, &slog.HandlerOptions{Level: slog.LevelWarn})
	oldLogger := slog.Default()
	slog.SetDefault(slog.New(handler))
	defer slog.SetDefault(oldLogger)

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	// Устанавливаем невалидный timeout
	_ = os.Setenv("BR_GIT2STORE_TIMEOUT", "invalid_duration")
	defer os.Unsetenv("BR_GIT2STORE_TIMEOUT") //nolint:errcheck

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	_ = h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout
	io.Copy(io.Discard, r)

	logs := logBuf.String()
	if !strings.Contains(logs, "BR_GIT2STORE_TIMEOUT") || !strings.Contains(logs, "invalid_duration") {
		t.Errorf("Должен быть warning о невалидном timeout, logs: %s", logs)
	}
}

// TestGit2StoreHandler_Execute_DryRun_JSONOutput проверяет dry-run JSON output.
// Проверяем dry_run: true и наличие плана в JSON результате.
func TestGit2StoreHandler_Execute_DryRun_JSONOutput(t *testing.T) {
	cfg := createTestConfig(t)

	h := &Git2StoreHandler{}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("Execute() в dry-run JSON режиме вернул ошибку: %v", err)
	}

	jsonOutput := buf.String()

	// Проверяем, что это валидный JSON
	var result output.Result
	if unmarshalErr := json.Unmarshal([]byte(jsonOutput), &result); unmarshalErr != nil {
		t.Fatalf("Не удалось распарсить JSON: %v\nOutput: %s", unmarshalErr, jsonOutput)
	}

	// Проверяем status
	if result.Status != output.StatusSuccess {
		t.Errorf("Status должен быть %q, got %q", output.StatusSuccess, result.Status)
	}

	// Проверяем command
	if result.Command != constants.ActNRGit2store {
		t.Errorf("Command должен быть %q, got %q", constants.ActNRGit2store, result.Command)
	}

	// Проверяем dry_run: true
	if !result.DryRun {
		t.Error("DryRun должен быть true в dry-run режиме")
	}

	// Проверяем наличие плана
	if result.Plan == nil {
		t.Fatal("Plan не должен быть nil в dry-run JSON режиме")
	}

	// Проверяем что план содержит шаги
	if len(result.Plan.Steps) == 0 {
		t.Error("Plan.Steps не должен быть пустым")
	}

	// Проверяем что план содержит команду
	if result.Plan.Command != constants.ActNRGit2store {
		t.Errorf("Plan.Command должен быть %q, got %q", constants.ActNRGit2store, result.Plan.Command)
	}

	// Data должен быть nil в dry-run режиме (нет реального выполнения)
	if result.Data != nil {
		t.Error("Data должен быть nil в dry-run режиме")
	}
}

// TestGit2StoreHandler_Execute_NilProjectConfig проверяет работу при nil ProjectConfig (M-1).
// Убеждаемся, что код не паникует при cfg.ProjectConfig == nil.
func TestGit2StoreHandler_Execute_NilProjectConfig(t *testing.T) {
	tmpDir := t.TempDir()
	appCfg := &config.AppConfig{}
	appCfg.Paths.Bin1cv8 = "/opt/1cv8/1cv8"

	cfg := &config.Config{
		Owner:         "test-owner",
		Repo:          "test-repo",
		TmpDir:        tmpDir,
		WorkDir:       tmpDir,
		InfobaseName:  "TestDB",
		AppConfig:     appCfg,
		ProjectConfig: nil, // M-1: ProjectConfig явно nil
		SecretConfig:  &config.SecretConfig{},
	}

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	// Не должно быть panic при nil ProjectConfig
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout
	io.Copy(io.Discard, r)

	// Execute должен успешно завершиться (без создания temp DB, т.к. ProjectConfig == nil)
	if err != nil {
		t.Errorf("Execute() с nil ProjectConfig вернул ошибку: %v", err)
	}
}

// ==== PLAN-ONLY TESTS (Story 7.3) ====

// TestGit2StoreHandler_PlanOnly_TextOutput проверяет текстовый вывод plan-only режима.
// Story 7.3 AC-1: При BR_PLAN_ONLY=true команда выводит план без выполнения.
// Story 7.3 AC-2: Заголовок "=== OPERATION PLAN ===" (не "=== DRY RUN ===").
func TestGit2StoreHandler_PlanOnly_TextOutput(t *testing.T) {
	cfg := createTestConfig(t)

	// Не создаём mock-и: plan-only не должен вызывать никакие операции
	h := &Git2StoreHandler{}

	t.Setenv("BR_PLAN_ONLY", "true")

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	capturedOutput := buf.String()

	// Plan-only не должен возвращать ошибку для валидной конфигурации
	if err != nil {
		t.Errorf("PlanOnly Execute() unexpected error = %v", err)
	}

	// Проверяем наличие заголовков plan-only (НЕ dry-run)
	expectedParts := []string{
		"=== OPERATION PLAN ===",
		"Команда: nr-git2store",
		"Валидация: ✅ Пройдена",
		"=== END OPERATION PLAN ===",
	}

	for _, part := range expectedParts {
		if !strings.Contains(capturedOutput, part) {
			t.Errorf("PlanOnly output should contain %q, got: %s", part, capturedOutput)
		}
	}

	// НЕ должно быть заголовка DRY RUN
	if strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("PlanOnly output should NOT contain '=== DRY RUN ===', got: %s", capturedOutput)
	}

	// Проверяем что этапы перечислены в плане
	for _, stage := range allStages[:5] {
		if !strings.Contains(capturedOutput, stage) {
			t.Errorf("PlanOnly output should contain stage %q", stage)
		}
	}
}

// TestGit2StoreHandler_PlanOnly_JSONOutput проверяет JSON вывод plan-only режима.
// Story 7.3 AC-6: JSON output содержит plan_only: true и plan: {...}.
func TestGit2StoreHandler_PlanOnly_JSONOutput(t *testing.T) {
	cfg := createTestConfig(t)

	h := &Git2StoreHandler{}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("PlanOnly Execute() unexpected error = %v", err)
	}

	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Fatalf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	// AC-6: plan_only: true
	if !result.PlanOnly {
		t.Error("JSON result.PlanOnly should be true")
	}

	// Plan не должен быть nil
	if result.Plan == nil {
		t.Error("JSON result.Plan should not be nil")
	} else {
		if result.Plan.Command != constants.ActNRGit2store {
			t.Errorf("Plan.Command = %q, want %q", result.Plan.Command, constants.ActNRGit2store)
		}
		if len(result.Plan.Steps) == 0 {
			t.Error("Plan.Steps should not be empty")
		}
	}

	// dry_run НЕ должен быть true
	if result.DryRun {
		t.Error("JSON result.DryRun should be false in plan-only mode")
	}

	if result.Command != constants.ActNRGit2store {
		t.Errorf("result.Command = %q, want %q", result.Command, constants.ActNRGit2store)
	}
}

// TestGit2StoreHandler_Priority_DryRunOverPlanOnly проверяет приоритет:
// BR_DRY_RUN > BR_PLAN_ONLY. Если оба заданы, должен сработать dry-run.
func TestGit2StoreHandler_Priority_DryRunOverPlanOnly(t *testing.T) {
	cfg := createTestConfig(t)

	h := &Git2StoreHandler{}

	// Устанавливаем оба флага
	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_PLAN_ONLY", "true")

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("Execute() unexpected error = %v", err)
	}

	// Должен быть DRY RUN (приоритет выше)
	if !strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("Output should contain '=== DRY RUN ===' (priority over plan-only), got: %s", capturedOutput)
	}

	// НЕ должно быть OPERATION PLAN
	if strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should NOT contain '=== OPERATION PLAN ===' when dry-run has priority, got: %s", capturedOutput)
	}
}

// TestGit2StoreHandler_Execute_NilGitConfig проверяет работу при nil GitConfig (M-2).
// Убеждаемся, что production.go не паникует при cfg.GitConfig == nil.
func TestGit2StoreHandler_Execute_NilGitConfig(t *testing.T) {
	tmpDir := t.TempDir()
	appCfg := &config.AppConfig{}
	appCfg.Paths.Bin1cv8 = "/opt/1cv8/1cv8"

	cfg := &config.Config{
		Owner:        "test-owner",
		Repo:         "test-repo",
		TmpDir:       tmpDir,
		WorkDir:      tmpDir,
		InfobaseName: "TestDB",
		AppConfig:    appCfg,
		ProjectConfig: &config.ProjectConfig{
			StoreDb: "TestStoreDb",
		},
		SecretConfig: &config.SecretConfig{},
		GitConfig:    nil, // M-2: GitConfig явно nil
	}

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	// Не должно быть panic при nil GitConfig (тест использует mock, но проверяем что код готов)
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout
	io.Copy(io.Discard, r)

	// Execute должен успешно завершиться
	if err != nil {
		t.Errorf("Execute() с nil GitConfig вернул ошибку: %v", err)
	}
}

// ==== VERBOSE / PRIORITY TESTS (Story 7.3) ====

// TestGit2StoreHandler_Verbose_TextOutput проверяет verbose режим: план выводится ПЕРЕД выполнением.
// Story 7.3 AC-4: В verbose режиме сначала выводится план, затем выполняется операция.
func TestGit2StoreHandler_Verbose_TextOutput(t *testing.T) {
	cfg := createTestConfig(t)

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	t.Setenv("BR_VERBOSE", "true")

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("Verbose Execute() unexpected error = %v", err)
	}

	// Проверяем что план выведен перед выполнением
	if !strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Verbose output should contain '=== OPERATION PLAN ===', got: %s", capturedOutput)
	}

	// Проверяем что реальное выполнение произошло
	if !strings.Contains(capturedOutput, "успешно") {
		t.Errorf("Verbose output should contain 'успешно', got: %s", capturedOutput)
	}
}

// TestGit2StoreHandler_Verbose_JSONOutput проверяет JSON вывод в verbose режиме.
// Story 7.3 AC-7: verbose JSON включает план в результат.
func TestGit2StoreHandler_Verbose_JSONOutput(t *testing.T) {
	cfg := createTestConfig(t)

	gitOp := &mockGitOperator{}
	ccOp := &mockConvertConfigOperator{}

	h := &Git2StoreHandler{
		gitFactory:           &mockGitFactory{gitOp: gitOp},
		convertConfigFactory: &mockConvertConfigFactory{ccOp: ccOp},
		backupCreator:        &mockBackupCreator{},
	}

	t.Setenv("BR_VERBOSE", "true")
	t.Setenv("BR_OUTPUT_FORMAT", "json")

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	if err != nil {
		t.Errorf("Verbose Execute() unexpected error = %v", err)
	}

	var result output.Result
	if jsonErr := json.Unmarshal(buf.Bytes(), &result); jsonErr != nil {
		t.Errorf("JSON unmarshal error: %v, output: %s", jsonErr, buf.String())
	}

	// Verbose включает план в JSON результат
	if result.Plan == nil {
		t.Error("Verbose JSON result.Plan should not be nil")
	}

	// Verbose — не plan-only и не dry-run
	if result.PlanOnly {
		t.Error("Verbose JSON result.PlanOnly should be false")
	}
	if result.DryRun {
		t.Error("Verbose JSON result.DryRun should be false")
	}

	// Реальное выполнение должно произойти
	if result.Status != output.StatusSuccess {
		t.Errorf("Verbose JSON result.Status = %q, want %q", result.Status, output.StatusSuccess)
	}

	// Data должна присутствовать (реальное выполнение)
	if result.Data == nil {
		t.Error("Verbose JSON result.Data should not be nil (real execution happened)")
	}
}

// TestGit2StoreHandler_Priority_DryRunOverVerbose проверяет приоритет dry-run над verbose.
// AC-9: dry-run имеет высший приоритет над verbose.
func TestGit2StoreHandler_Priority_DryRunOverVerbose(t *testing.T) {
	cfg := createTestConfig(t)

	// Не создаём mock-и: dry-run не должен вызывать никакие операции
	h := &Git2StoreHandler{}

	t.Setenv("BR_DRY_RUN", "true")
	t.Setenv("BR_VERBOSE", "true")

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("Priority Execute() unexpected error = %v", err)
	}

	// Должен быть dry-run заголовок, НЕ operation plan
	if !strings.Contains(capturedOutput, "=== DRY RUN ===") {
		t.Errorf("Output should contain '=== DRY RUN ===' (dry-run priority), got: %s", capturedOutput)
	}
	if strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should NOT contain '=== OPERATION PLAN ===' when dry-run active, got: %s", capturedOutput)
	}
}

// TestGit2StoreHandler_Priority_PlanOnlyOverVerbose проверяет приоритет plan-only над verbose.
// Plan-only останавливает выполнение (показывает план, не выполняет).
// Verbose показывает план и выполняет. Plan-only имеет приоритет.
func TestGit2StoreHandler_Priority_PlanOnlyOverVerbose(t *testing.T) {
	cfg := createTestConfig(t)

	// Не создаём mock-и: plan-only не должен вызывать никакие операции
	h := &Git2StoreHandler{}

	t.Setenv("BR_PLAN_ONLY", "true")
	t.Setenv("BR_VERBOSE", "true")

	// Перенаправляем stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	ctx := context.Background()
	err := h.Execute(ctx, cfg)

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)
	capturedOutput := buf.String()

	if err != nil {
		t.Errorf("Priority Execute() unexpected error = %v", err)
	}

	// Должен быть plan-only заголовок
	if !strings.Contains(capturedOutput, "=== OPERATION PLAN ===") {
		t.Errorf("Output should contain '=== OPERATION PLAN ===', got: %s", capturedOutput)
	}

	// НЕ должно быть реального выполнения (Прогресс выводится только writeText при реальном выполнении)
	if strings.Contains(capturedOutput, "Прогресс:") {
		t.Errorf("Output should NOT contain 'Прогресс:' (plan-only, no execution), got: %s", capturedOutput)
	}
}
