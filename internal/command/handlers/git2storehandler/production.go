// Package git2storehandler содержит production реализации для синхронизации Git → хранилище 1C.
package git2storehandler

import (
	"context"
	"log/slog"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/one/convert"
	"github.com/Kargones/apk-ci/internal/entity/one/designer"
	"github.com/Kargones/apk-ci/internal/git"
)

// gitWrapper — обёртка для git.Git, реализующая GitOperator.
type gitWrapper struct {
	git *git.Git
}

// Clone клонирует репозиторий.
func (g *gitWrapper) Clone(ctx *context.Context, l *slog.Logger) error {
	return g.git.Clone(ctx, l)
}

// Switch переключается на ветку.
func (g *gitWrapper) Switch(ctx context.Context, l *slog.Logger) error {
	return g.git.Switch(ctx, l)
}

// SetBranch устанавливает ветку.
func (g *gitWrapper) SetBranch(branch string) {
	g.git.Branch = branch
}

// createGitProduction — production реализация создания GitOperator.
// SECURITY: Token включается в URL для аутентификации git clone.
// Ошибки git операций могут содержать URL — внутренняя реализация git.Git
// должна sanitize credentials перед логированием (см. internal/git/git.go).
func createGitProduction(l *slog.Logger, cfg *config.Config) (GitOperator, error) {
	// Формируем URL репозитория аналогично app.InitGit
	// ВАЖНО: Token в URL — стандартный паттерн для git clone с аутентификацией
	connectString := strings.Replace(cfg.GiteaURL, "https://", "https://"+url.PathEscape(cfg.AccessToken)+":@", 1)
	repURL := connectString + "/" + cfg.Owner + "/" + cfg.Repo + ".git"

	// Логируем URL без credentials для отладки
	safeURL := cfg.GiteaURL + "/" + cfg.Owner + "/" + cfg.Repo + ".git"
	l.Debug("Создание GitOperator", slog.String("repo_url", safeURL))

	// H-2 fix: Проверка cfg.GitConfig на nil, использование значения по умолчанию
	var gitTimeout time.Duration
	if cfg.GitConfig != nil {
		gitTimeout = cfg.GitConfig.Timeout
	}
	// Если timeout не задан, используем 0 (git.Git использует свой default)

	g := &git.Git{
		RepURL:  repURL,
		RepPath: cfg.RepPath,
		WorkDir: cfg.WorkDir,
		Token:   cfg.AccessToken,
		Branch:  cfg.BaseBranch,
		Timeout: gitTimeout,
	}
	return &gitWrapper{git: g}, nil
}

// convertConfigWrapper — обёртка для convert.Config, реализующая ConvertConfigOperator.
type convertConfigWrapper struct {
	cc *convert.Config
}

// Load загружает конфигурацию конвертации.
func (c *convertConfigWrapper) Load(ctx *context.Context, l *slog.Logger, cfg *config.Config, infobaseName string) error {
	return c.cc.Load(ctx, l, cfg, infobaseName)
}

// InitDb инициализирует базу данных.
func (c *convertConfigWrapper) InitDb(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.InitDb(ctx, l, cfg)
}

// StoreUnBind отвязывает от хранилища.
func (c *convertConfigWrapper) StoreUnBind(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.StoreUnBind(ctx, l, cfg)
}

// LoadDb загружает конфигурацию в БД.
func (c *convertConfigWrapper) LoadDb(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.LoadDb(ctx, l, cfg)
}

// DbUpdate обновляет БД.
func (c *convertConfigWrapper) DbUpdate(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.DbUpdate(ctx, l, cfg)
}

// DumpDb выгружает БД.
func (c *convertConfigWrapper) DumpDb(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.DumpDb(ctx, l, cfg)
}

// StoreBind привязывает к хранилищу.
func (c *convertConfigWrapper) StoreBind(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.StoreBind(ctx, l, cfg)
}

// StoreLock блокирует объекты в хранилище.
func (c *convertConfigWrapper) StoreLock(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.StoreLock(ctx, l, cfg)
}

// Merge выполняет слияние.
func (c *convertConfigWrapper) Merge(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.Merge(ctx, l, cfg)
}

// StoreCommit фиксирует изменения в хранилище.
func (c *convertConfigWrapper) StoreCommit(ctx *context.Context, l *slog.Logger, cfg *config.Config) error {
	return c.cc.StoreCommit(ctx, l, cfg)
}

// SetOneDB устанавливает параметры временной БД.
func (c *convertConfigWrapper) SetOneDB(dbConnectString, user, pass string) {
	c.cc.OneDB = designer.OneDb{
		DbConnectString:   dbConnectString,
		User:              user,
		Pass:              pass,
		FullConnectString: dbConnectString,
		ServerDb:          false,
		DbExist:           true,
	}
	// Формируем полную строку подключения с пользователем и паролем
	if user != "" {
		c.cc.OneDB.FullConnectString = dbConnectString + " /N " + user
		if pass != "" {
			c.cc.OneDB.FullConnectString = c.cc.OneDB.FullConnectString + " /P " + pass
		}
	}
}

// createConvertConfigProduction — production реализация создания ConvertConfigOperator.
func createConvertConfigProduction() ConvertConfigOperator {
	return &convertConfigWrapper{cc: &convert.Config{}}
}

// createTempDbProduction — production реализация создания временной БД.
func createTempDbProduction(ctx *context.Context, l *slog.Logger, cfg *config.Config) (string, error) {
	// Генерируем путь для временной базы данных
	dbPath := filepath.Join(cfg.TmpDir, "temp_db_"+time.Now().Format("20060102_150405"))

	// Вызов функции CreateTempDb с правильными параметрами
	oneDb, err := designer.CreateTempDb(*ctx, l, cfg, dbPath, cfg.AddArray)
	if err != nil {
		return "", err
	}

	l.Debug("Временная база данных создана успешно",
		slog.String("Строка соединения", oneDb.DbConnectString),
		slog.Bool("Существует", oneDb.DbExist),
	)

	return oneDb.DbConnectString, nil
}
