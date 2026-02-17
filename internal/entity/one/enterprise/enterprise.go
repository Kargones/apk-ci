// Package enterprise provides functionality for executing external 1C processing files (.epf)
package enterprise

import (
	"github.com/Kargones/apk-ci/internal/constants"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
	"github.com/Kargones/apk-ci/internal/util/runner"
)

// EpfExecutor представляет исполнитель внешних обработок 1С
type EpfExecutor struct {
	logger *slog.Logger
	runner *runner.Runner
}

// NewEpfExecutor создает новый экземпляр исполнителя внешних обработок
func NewEpfExecutor(logger *slog.Logger, workDir string) *EpfExecutor {
	r := &runner.Runner{
		TmpDir:  workDir,
		WorkDir: workDir,
	}

	return &EpfExecutor{
		logger: logger,
		runner: r,
	}
}

// Execute выполняет внешнюю обработку по указанному URL
func (e *EpfExecutor) Execute(ctx context.Context, cfg *config.Config) error {
	if cfg == nil {
		return fmt.Errorf("конфигурация не может быть nil")
	}

	// Валидация URL
	if err := e.validateEpfURL(cfg.StartEpf); err != nil {
		return err
	}

	e.logger.Info("Начало выполнения внешней обработки",
		slog.String("epf_url", cfg.StartEpf),
		slog.String("infobase", cfg.InfobaseName),
	)

	// Создание временной директории если необходимо
	if err := e.ensureTempDirectory(cfg); err != nil {
		return err
	}

	// Скачивание .epf файла
	tempEpfPath, cleanup, err := e.downloadEpfFile(cfg)
	if err != nil {
		return err
	}
	defer cleanup()

	// Подготовка параметров подключения
	connectString, err := e.prepareConnectionString(cfg)
	if err != nil {
		return err
	}

	// Запуск внешней обработки
	if err := e.executeEpfInEnterprise(ctx, cfg, tempEpfPath, connectString); err != nil {
		return err
	}

	e.logger.Info("Внешняя обработка успешно выполнена",
		slog.String("epf_url", cfg.StartEpf),
		slog.String("infobase", cfg.InfobaseName),
	)

	return nil
}

// validateEpfURL проверяет корректность URL для .epf файла
func (e *EpfExecutor) validateEpfURL(url string) error {
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		return fmt.Errorf("некорректный URL для StartEpf: %s", url)
	}
	return nil
}

// ensureTempDirectory создает временную директорию если необходимо
func (e *EpfExecutor) ensureTempDirectory(cfg *config.Config) error {
	if cfg.AppConfig != nil && cfg.AppConfig.TmpDir != "" {
		if err := os.MkdirAll(cfg.AppConfig.TmpDir, constants.DirPermPrivate); err != nil {
			e.logger.Error("Ошибка создания временной директории",
				slog.String("tmpDir", cfg.AppConfig.TmpDir),
				slog.String("error", err.Error()),
			)
			return fmt.Errorf("ошибка создания временной директории: %w", err)
		}
	}
	return nil
}

// downloadEpfFile скачивает .epf файл и возвращает путь к нему и функцию очистки
func (e *EpfExecutor) downloadEpfFile(cfg *config.Config) (string, func(), error) {
	e.logger.Debug("Скачивание .epf файла", slog.String("url", cfg.StartEpf))

	// Создаем временный файл для .epf
	tempEpfFile, err := os.CreateTemp(cfg.WorkDir, "*.epf")
	if err != nil {
		e.logger.Error("Ошибка создания временного файла для .epf",
			slog.String("error", err.Error()),
		)
		return "", nil, fmt.Errorf("ошибка создания временного файла: %w", err)
	}

	cleanup := func() {
		if closeErr := tempEpfFile.Close(); closeErr != nil {
			e.logger.Warn("Ошибка закрытия временного файла", slog.String("error", closeErr.Error()))
		}
		if removeErr := os.Remove(tempEpfFile.Name()); removeErr != nil {
			e.logger.Warn("Ошибка удаления временного файла", slog.String("error", removeErr.Error()))
		}
	}

	// Скачиваем файл с помощью Gitea API
	giteaConfig := gitea.Config{
		GiteaURL:    cfg.GiteaURL,
		Owner:       cfg.Owner,
		Repo:        cfg.Repo,
		AccessToken: cfg.AccessToken,
		BaseBranch:  cfg.BaseBranch,
		NewBranch:   cfg.NewBranch,
		Command:     cfg.Command,
	}
	giteaAPI := gitea.NewGiteaAPI(giteaConfig)
	data, err := giteaAPI.GetConfigData(e.logger, cfg.StartEpf)
	if err != nil {
		e.logger.Error("Ошибка получения данных .epf файла",
			slog.String("url", cfg.StartEpf),
			slog.String("error", err.Error()),
		)
		cleanup()
		return "", nil, fmt.Errorf("ошибка получения данных .epf файла: %w", err)
	}

	// Записываем данные в временный файл
	_, err = tempEpfFile.Write(data)
	if err != nil {
		e.logger.Error("Ошибка записи данных в временный файл",
			slog.String("error", err.Error()),
		)
		cleanup()
		return "", nil, fmt.Errorf("ошибка записи данных в файл: %w", err)
	}

	e.logger.Debug("Файл .epf успешно скачан", slog.String("path", tempEpfFile.Name()))
	return tempEpfFile.Name(), cleanup, nil
}

// prepareConnectionString подготавливает строку подключения к базе данных
func (e *EpfExecutor) prepareConnectionString(cfg *config.Config) (string, error) {
	// Получаем информацию о базе данных
	dbInfo := cfg.GetDatabaseInfo(cfg.InfobaseName)
	if dbInfo == nil {
		return "", fmt.Errorf("база данных %s не найдена в конфигурации", cfg.InfobaseName)
	}

	// Формируем строку подключения
	connectString := "/S " + dbInfo.OneServer + "\\" + cfg.InfobaseName
	if cfg.AppConfig.Users.Db != "" {
		connectString += " /N " + cfg.AppConfig.Users.Db
		if cfg.SecretConfig.Passwords.Db != "" {
			connectString += " /P " + cfg.SecretConfig.Passwords.Db
		}
	}

	e.logger.Debug("Подготовка к запуску внешней обработки",
		slog.String("connect_string", connectString),
	)

	return connectString, nil
}

// executeEpfInEnterprise запускает внешнюю обработку в 1С:Предприятие
func (e *EpfExecutor) executeEpfInEnterprise(ctx context.Context, cfg *config.Config, epfPath, connectString string) error {
	e.runner.ClearParams()
	e.runner.RunString = cfg.AppConfig.Paths.Bin1cv8
	e.runner.Params = append(e.runner.Params, "@")
	e.runner.Params = append(e.runner.Params, "ENTERPRISE")
	e.runner.Params = append(e.runner.Params, connectString)
	e.runner.Params = append(e.runner.Params, "/Execute")
	e.runner.Params = append(e.runner.Params, epfPath)
	addDisableParam(e.runner)
	e.logger.Info("Запуск внешней обработки в 1С:Предприятие")
	_, err := e.runner.RunCommand(ctx, e.logger)
	if err != nil {
		e.logger.Error("Ошибка выполнения внешней обработки",
			slog.String("epf_file", epfPath),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("ошибка выполнения внешней обработки: %w", err)
	}

	return nil
}

func addDisableParam(r *runner.Runner) {
	r.Params = append(r.Params, "/DisableStartupDialogs")
	r.Params = append(r.Params, "/DisableStartupMessages")
	r.Params = append(r.Params, "/DisableUnrecoverableErrorMessage")
	r.Params = append(r.Params, "/UC ServiceMode")
}
