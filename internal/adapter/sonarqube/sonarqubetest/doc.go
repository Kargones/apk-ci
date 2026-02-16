// Package sonarqubetest предоставляет тестовые утилиты для пакета sonarqube.
//
// Основные компоненты:
//
// MockClient — мок-реализация sonarqube.Client с функциональными полями
// для гибкой настройки поведения в тестах. Реализует все ISP-интерфейсы:
// ProjectsAPI, AnalysesAPI, IssuesAPI, QualityGatesAPI, MetricsAPI.
//
// # Использование MockClient
//
// Базовое использование с дефолтным поведением:
//
//	mock := sonarqubetest.NewMockClient()
//	project, err := mock.GetProject(ctx, "my-project")
//	// project содержит тестовые данные, err == nil
//
// Кастомизация поведения через функциональные поля:
//
//	mock := &sonarqubetest.MockClient{
//	    GetProjectFunc: func(ctx context.Context, key string) (*sonarqube.Project, error) {
//	        if key == "not-found" {
//	            return nil, sonarqube.NewSonarQubeError(
//	                sonarqube.ErrSonarQubeNotFound,
//	                "Project not found",
//	                nil,
//	            )
//	        }
//	        return &sonarqube.Project{Key: key, Name: "Test"}, nil
//	    },
//	}
//
// # ISP (Interface Segregation Principle)
//
// MockClient реализует все role-based интерфейсы, но функции-потребители
// должны принимать только минимально необходимый интерфейс:
//
//	// Хорошо: функция требует только ProjectsAPI
//	func createProject(api sonarqube.ProjectsAPI, key string) error {
//	    _, err := api.CreateProject(ctx, sonarqube.CreateProjectOptions{Key: key})
//	    return err
//	}
//
//	// Плохо: функция требует весь Client, хотя использует только один метод
//	func createProjectBad(client sonarqube.Client, key string) error {
//	    _, err := client.CreateProject(ctx, sonarqube.CreateProjectOptions{Key: key})
//	    return err
//	}
//
// # Вспомогательные конструкторы
//
// Для частых сценариев предоставляются готовые конструкторы:
//
//   - NewMockClient() — пустой mock с дефолтным поведением
//   - NewMockClientWithProject(project) — mock с предзаданным проектом
//   - NewMockClientWithQualityGateStatus(status, conditions) — mock с предзаданным QG статусом
//   - NewMockClientWithIssues(issues) — mock с предзаданными проблемами
//
// # Тестовые данные
//
// Функции для получения готовых тестовых данных:
//
//   - ProjectData() — тестовый проект
//   - IssueData() — список тестовых проблем
//   - AnalysisData() — список тестовых анализов
package sonarqubetest
