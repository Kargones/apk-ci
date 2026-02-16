package sonarqubetest

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/Kargones/apk-ci/internal/adapter/sonarqube"
)

// TestMockClientImplementsInterfaces проверяет, что MockClient реализует все интерфейсы.
func TestMockClientImplementsInterfaces(t *testing.T) {
	mock := NewMockClient()

	// Проверка реализации Client (композитный интерфейс)
	var client sonarqube.Client = mock
	if client == nil {
		t.Fatal("MockClient должен реализовывать sonarqube.Client")
	}

	// Проверка реализации отдельных интерфейсов (ISP)
	var _ sonarqube.ProjectsAPI = mock
	var _ sonarqube.AnalysesAPI = mock
	var _ sonarqube.IssuesAPI = mock
	var _ sonarqube.QualityGatesAPI = mock
	var _ sonarqube.MetricsAPI = mock
}

// TestMockClientDefaultBehavior проверяет поведение по умолчанию.
func TestMockClientDefaultBehavior(t *testing.T) {
	ctx := context.Background()
	mock := NewMockClient()

	// CreateProject по умолчанию возвращает проект с переданными данными
	project, err := mock.CreateProject(ctx, sonarqube.CreateProjectOptions{
		Key:        "my-project",
		Name:       "My Project",
		Visibility: "private",
	})
	if err != nil {
		t.Fatalf("CreateProject не должен возвращать ошибку: %v", err)
	}
	if project.Key != "my-project" {
		t.Errorf("Ожидался Key 'my-project', получен '%s'", project.Key)
	}

	// GetQualityGateStatus по умолчанию возвращает OK
	qgStatus, err := mock.GetQualityGateStatus(ctx, "test-project")
	if err != nil {
		t.Fatalf("GetQualityGateStatus не должен возвращать ошибку: %v", err)
	}
	if qgStatus.Status != "OK" {
		t.Errorf("Ожидался статус 'OK', получен '%s'", qgStatus.Status)
	}
}

// TestMockClientCustomBehavior проверяет переопределение поведения через функциональные поля.
func TestMockClientCustomBehavior(t *testing.T) {
	ctx := context.Background()

	expectedErr := errors.New("project not found")
	mock := &MockClient{
		GetProjectFunc: func(_ context.Context, projectKey string) (*sonarqube.Project, error) {
			if projectKey == "non-existent" {
				return nil, expectedErr
			}
			return &sonarqube.Project{Key: projectKey}, nil
		},
	}

	// Проверка кастомного поведения — ошибка для несуществующего проекта
	_, err := mock.GetProject(ctx, "non-existent")
	if !errors.Is(err, expectedErr) {
		t.Errorf("Ожидалась ошибка '%v', получена '%v'", expectedErr, err)
	}

	// Проверка кастомного поведения — успех для существующего проекта
	project, err := mock.GetProject(ctx, "existing-project")
	if err != nil {
		t.Fatalf("Не ожидалась ошибка: %v", err)
	}
	if project.Key != "existing-project" {
		t.Errorf("Ожидался Key 'existing-project', получен '%s'", project.Key)
	}
}

// TestMockClientWithProject проверяет конструктор с предзаданным проектом.
func TestMockClientWithProject(t *testing.T) {
	ctx := context.Background()
	expectedProject := &sonarqube.Project{
		Key:         "preset-project",
		Name:        "Preset Project",
		Description: "Проект для теста",
		Visibility:  "public",
	}

	mock := NewMockClientWithProject(expectedProject)

	project, err := mock.GetProject(ctx, "any-key")
	if err != nil {
		t.Fatalf("GetProject не должен возвращать ошибку: %v", err)
	}
	if project.Key != expectedProject.Key {
		t.Errorf("Ожидался Key '%s', получен '%s'", expectedProject.Key, project.Key)
	}
}

// TestMockClientWithQualityGateStatus проверяет конструктор с предзаданным статусом QG.
func TestMockClientWithQualityGateStatus(t *testing.T) {
	ctx := context.Background()
	conditions := []sonarqube.QualityCondition{
		{
			Metric:         "coverage",
			Operator:       "LT",
			ErrorThreshold: "80",
			ActualValue:    "75",
			Status:         "ERROR",
		},
	}

	mock := NewMockClientWithQualityGateStatus("ERROR", conditions)

	qgStatus, err := mock.GetQualityGateStatus(ctx, "test-project")
	if err != nil {
		t.Fatalf("GetQualityGateStatus не должен возвращать ошибку: %v", err)
	}
	if qgStatus.Status != "ERROR" {
		t.Errorf("Ожидался статус 'ERROR', получен '%s'", qgStatus.Status)
	}
	if len(qgStatus.Conditions) != 1 {
		t.Errorf("Ожидалось 1 условие, получено %d", len(qgStatus.Conditions))
	}
}

// TestMockClientWithIssues проверяет конструктор с предзаданными проблемами.
func TestMockClientWithIssues(t *testing.T) {
	ctx := context.Background()
	issues := IssueData()

	mock := NewMockClientWithIssues(issues)

	gotIssues, err := mock.GetIssues(ctx, sonarqube.GetIssuesOptions{ProjectKey: "test"})
	if err != nil {
		t.Fatalf("GetIssues не должен возвращать ошибку: %v", err)
	}
	if len(gotIssues) != len(issues) {
		t.Errorf("Ожидалось %d проблем, получено %d", len(issues), len(gotIssues))
	}
}

// TestProjectData проверяет вспомогательную функцию для тестовых данных проекта.
func TestProjectData(t *testing.T) {
	project := ProjectData()

	if project.Key == "" {
		t.Error("Project.Key не должен быть пустым")
	}
	if project.Name == "" {
		t.Error("Project.Name не должен быть пустым")
	}
	if project.LastAnalysisDate == nil {
		t.Error("Project.LastAnalysisDate не должен быть nil")
	}
}

// TestIssueData проверяет вспомогательную функцию для тестовых данных проблем.
func TestIssueData(t *testing.T) {
	issues := IssueData()

	if len(issues) == 0 {
		t.Fatal("IssueData должен возвращать непустой срез")
	}
	for i, issue := range issues {
		if issue.Key == "" {
			t.Errorf("Issue[%d].Key не должен быть пустым", i)
		}
		if issue.Rule == "" {
			t.Errorf("Issue[%d].Rule не должен быть пустым", i)
		}
	}
}

// TestAnalysisData проверяет вспомогательную функцию для тестовых данных анализов.
func TestAnalysisData(t *testing.T) {
	analyses := AnalysisData()

	if len(analyses) == 0 {
		t.Fatal("AnalysisData должен возвращать непустой срез")
	}
	for i, analysis := range analyses {
		if analysis.ID == "" {
			t.Errorf("Analysis[%d].ID не должен быть пустым", i)
		}
		if analysis.ProjectKey == "" {
			t.Errorf("Analysis[%d].ProjectKey не должен быть пустым", i)
		}
	}
}

// ExampleMockClient демонстрирует базовое использование MockClient.
func ExampleMockClient() {
	ctx := context.Background()

	// Создаём mock с кастомным поведением
	mock := &MockClient{
		CreateProjectFunc: func(_ context.Context, opts sonarqube.CreateProjectOptions) (*sonarqube.Project, error) {
			return &sonarqube.Project{
				Key:        opts.Key,
				Name:       opts.Name,
				Visibility: opts.Visibility,
			}, nil
		},
	}

	// Используем mock как sonarqube.Client
	var client sonarqube.Client = mock

	project, _ := client.CreateProject(ctx, sonarqube.CreateProjectOptions{
		Key:  "example-project",
		Name: "Example Project",
	})
	fmt.Println(project.Key)
	// Output: example-project
}

// ExampleMockClient_isp демонстрирует ISP (Interface Segregation Principle):
// функции должны принимать только минимально необходимый интерфейс,
// а не композитный Client.
func ExampleMockClient_isp() {
	ctx := context.Background()

	// Функция принимает только ProjectsAPI, не весь Client.
	// Это позволяет использовать любую реализацию ProjectsAPI,
	// не требуя реализации других интерфейсов (AnalysesAPI, IssuesAPI и т.д.)
	createProject := func(api sonarqube.ProjectsAPI, key, name string) (*sonarqube.Project, error) {
		return api.CreateProject(ctx, sonarqube.CreateProjectOptions{
			Key:  key,
			Name: name,
		})
	}

	// MockClient реализует ProjectsAPI (и другие интерфейсы)
	mock := NewMockClient()

	// Передаём mock как ProjectsAPI
	project, _ := createProject(mock, "my-project", "My Project")
	fmt.Println(project.Key)
	// Output: my-project
}

// ExampleMockClient_qualityGateCheck демонстрирует проверку Quality Gate
// с использованием только QualityGatesAPI интерфейса.
func ExampleMockClient_qualityGateCheck() {
	ctx := context.Background()

	// Функция для проверки Quality Gate принимает только QualityGatesAPI
	checkQualityGate := func(api sonarqube.QualityGatesAPI, projectKey string) (bool, error) {
		status, err := api.GetQualityGateStatus(ctx, projectKey)
		if err != nil {
			return false, err
		}
		return status.Status == "OK", nil
	}

	// Создаём mock с предзаданным статусом
	mock := NewMockClientWithQualityGateStatus("OK", nil)

	passed, _ := checkQualityGate(mock, "test-project")
	fmt.Println(passed)
	// Output: true
}
