package service

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Kargones/apk-ci/internal/config"
	"github.com/Kargones/apk-ci/internal/entity/gitea"
)

// GiteaService содержит бизнес-логику для работы с Gitea
type GiteaService struct {
	api             gitea.APIInterface
	config          *config.Config
	projectAnalyzer gitea.ProjectAnalyzer
}

// NewGiteaService создает новый экземпляр GiteaService.
// Инициализирует сервис для работы с Gitea API, включая
// конфигурацию и анализатор проектов.
// Параметры:
//   - api: интерфейс для работы с Gitea API
//   - cfg: конфигурация приложения
//   - analyzer: анализатор проектов
//
// Возвращает:
//   - *GiteaService: новый экземпляр сервиса
func NewGiteaService(api gitea.APIInterface, cfg *config.Config, analyzer gitea.ProjectAnalyzer) *GiteaService {
	return &GiteaService{
		api:             api,
		config:          cfg,
		projectAnalyzer: analyzer,
	}
}

// TestMerge выполняет тестирование слияния pull request'ов.
// Создает тестовую ветку, пытается слить активные PR и закрывает
// конфликтные или неуспешные PR.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//
// Возвращает:
//   - error: ошибка выполнения или nil при успехе
func (s *GiteaService) TestMerge(_ context.Context, l *slog.Logger) error {
	// Получаю список активных PR
	activePRs, err := s.api.ActivePR()
	if err != nil {
		l.Error("Ошибка получения активных PR",
			slog.String("BR_ACTION", "test-merge"),
			slog.String("constants.MsgErrProcessing", "constants.MsgAppExit"),
		)
		return err
	}
	l.Debug("Список активных PR",
		slog.String("BR_ACTION", "test-merge"),
		slog.String("BR_ACTIVE_PR", fmt.Sprintf("%v", activePRs)),
	)
	_ = s.api.DeleteTestBranch()
	err = s.api.CreateTestBranch()
	if err != nil {
		l.Error("Ошибка создания тестовой ветки",
			slog.String("BR_ACTION", "test-merge"),
			slog.String("constants.MsgErrProcessing", "constants.MsgAppExit"),
		)
		return err
	}
	l.Debug("Тестовая ветка успешно создана",
		slog.String("BR_ACTION", "test-merge"),
		slog.String("BR_TEST_BRANCH", "constants.TestBranch"),
		slog.String("BR_BASE_BRANCH", s.config.BaseBranch),
	)
	testPR := []gitea.PR{}
	var toClose bool

	for _, pr := range activePRs {
		l.Debug("Обрабатывается PR",
			slog.String("BR_ACTION", "test-merge"),
			slog.Int64("BR_PR_NUMBER", pr.Number),
			slog.String("BR_PR_HEAD", pr.Head),
			slog.String("BR_PR_BASE", pr.Base),
		)
		newPR, createErr := s.api.CreatePR(pr.Head)
		if createErr != nil {
			l.Error("Ошибка создания PR",
				slog.String("BR_ACTION", "test-merge"),
				slog.String("BR_PR_HEAD", pr.Head),
				slog.String("BR_PR_BASE", pr.Base),
				slog.String("Error", createErr.Error()),
			)
			return createErr
		}
		l.Debug("Создан PR",
			slog.String("BR_ACTION", "test-merge"),
			slog.Int64("BR_PR_NUMBER", newPR.Number),
			slog.String("BR_PR_HEAD", newPR.Head),
			slog.String("BR_PR_BASE", newPR.Base),
		)
		// Добавляю PR в список тестовых PR
		testPR = append(testPR, newPR)
	}

	for prArrayID, newPR := range testPR {
		toClose = false
		isConflict, conflictErr := s.api.ConflictPR(newPR.Number)
		if conflictErr != nil {
			l.Debug("Ошибка проверки конфликта PR",
				slog.String("BR_ACTION", "test-merge"),
				slog.Int64("BR_PR_NUMBER", newPR.Number),
				slog.String("BR_PR_HEAD", newPR.Head),
				slog.String("BR_PR_BASE", newPR.Base),
			)
			isConflict = true
		}
		if isConflict {
			toClose = true
			l.Debug("Конфликт PR",
				slog.String("BR_ACTION", "test-merge"),
				slog.Int64("BR_PR_NUMBER", newPR.Number),
				slog.String("BR_PR_HEAD", newPR.Head),
				slog.String("BR_PR_BASE", newPR.Base),
			)
		}

		err = s.api.MergePR(newPR.Number, l)
		if err != nil {
			l.Debug("Ошибка слияния PR",
				slog.String("BR_ACTION", "test-merge"),
				slog.Int64("BR_PR_NUMBER", newPR.Number),
				slog.String("BR_PR_HEAD", newPR.Head),
				slog.String("BR_PR_BASE", newPR.Base),
			)
			toClose = true
		}
		if toClose {
			pr := activePRs[prArrayID]
			err = s.api.ClosePR(pr.Number)
			if err == nil {
				l.Debug("Закрыт PR",
					slog.String("BR_ACTION", "test-merge"),
					slog.Int64("BR_PR_ID", pr.ID),
					slog.Int64("BR_PR_NUMBER", pr.Number),
					slog.String("BR_PR_HEAD", pr.Head),
					slog.String("BR_PR_BASE", pr.Base),
				)
			}
		}
	}
	l.Debug("testPR",
		slog.String("BR_ACTION", "test-merge"),
		slog.String("BR_TEST_PR", fmt.Sprintf("%v", testPR)),
	)
	err = s.api.DeleteTestBranch()
	if err != nil {
		l.Error("Ошибка удаления тестовой ветки",
			slog.String("BR_ACTION", "test-merge"),
			slog.String("constants.MsgErrProcessing", "constants.MsgAppExit"),
		)
		return err
	}
	l.Debug("Ветка успешно удалена",
		slog.String("BR_ACTION", "test-merge"),
		slog.String("BR_TEST_BRANCH", "constants.TestBranch"),
	)
	return nil
}

// AnalyzeProject выполняет анализ структуры проекта.
// Использует настроенный анализатор для изучения проекта
// в указанной ветке репозитория.
// Параметры:
//   - ctx: контекст выполнения операции
//   - l: логгер для записи сообщений
//   - branch: имя ветки для анализа
//
// Возвращает:
//   - error: ошибка анализа или nil при успехе
func (s *GiteaService) AnalyzeProject(_ context.Context, l *slog.Logger, branch string) error {
	// Используем ProjectAnalyzer для анализа проекта
	return s.projectAnalyzer.AnalyzeProject(l, branch)
}

// GetAPI возвращает интерфейс Gitea API.
// Предоставляет доступ к API для выполнения операций
// с репозиторием Gitea.
// Возвращает:
//   - gitea.APIInterface: интерфейс для работы с Gitea API
func (s *GiteaService) GetAPI() gitea.APIInterface {
	return s.api
}

// GetConfig возвращает конфигурацию сервиса.
// Предоставляет доступ к настройкам приложения,
// используемым сервисом.
// Возвращает:
//   - *config.Config: конфигурация приложения
func (s *GiteaService) GetConfig() *config.Config {
	return s.config
}
