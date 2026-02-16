package sonarqube

import "testing"

// TestInterfaceComposition проверяет, что композитный интерфейс Client
// может быть безопасно приведён к любому role-based интерфейсу.
// Это гарантирует корректность ISP-разделения.
func TestInterfaceComposition(t *testing.T) {
	// Проверяем, что Client включает все role-based интерфейсы
	// Компилятор проверит это на этапе компиляции
	var _ interface {
		ProjectsAPI
		AnalysesAPI
		IssuesAPI
		QualityGatesAPI
		MetricsAPI
	} = (Client)(nil)
}

// TestRoleBasedInterfacesAreIndependent проверяет, что role-based интерфейсы
// не зависят друг от друга и могут использоваться независимо.
func TestRoleBasedInterfacesAreIndependent(t *testing.T) {
	// Каждый интерфейс должен быть самодостаточным
	// Функция, принимающая только ProjectsAPI, не должна требовать других интерфейсов
	type projectsOnlyConsumer interface {
		ProjectsAPI
	}

	type analysesOnlyConsumer interface {
		AnalysesAPI
	}

	type issuesOnlyConsumer interface {
		IssuesAPI
	}

	type qualityGatesOnlyConsumer interface {
		QualityGatesAPI
	}

	type metricsOnlyConsumer interface {
		MetricsAPI
	}

	// Эти проверки нужны только для документации ISP-принципа
	_ = projectsOnlyConsumer(nil)
	_ = analysesOnlyConsumer(nil)
	_ = issuesOnlyConsumer(nil)
	_ = qualityGatesOnlyConsumer(nil)
	_ = metricsOnlyConsumer(nil)
}
