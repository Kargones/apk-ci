// Package giteatest предоставляет тестовые утилиты для пакета gitea.
//
// Пакет содержит мок-реализации интерфейсов для unit-тестирования компонентов,
// использующих Gitea API, без необходимости подключения к реальному серверу.
//
// # MockClient
//
// MockClient — основной инструмент для тестирования. Реализует интерфейс gitea.Client
// и все его составные интерфейсы (PRReader, CommitReader, FileReader и др.).
// Использует паттерн функциональных полей для гибкой настройки поведения.
//
// Пример использования:
//
//	mock := giteatest.NewMockClient()
//	mock.GetPRFunc = func(ctx context.Context, prNumber int64) (*gitea.PRResponse, error) {
//	    return &gitea.PRResponse{Number: prNumber, State: "open"}, nil
//	}
//
// # Конструкторы
//
// Пакет предоставляет удобные конструкторы для типовых сценариев:
//   - NewMockClient() — пустой mock с дефолтными ответами
//   - NewMockClientWithPR(pr) — mock с предзаданным PR
//   - NewMockClientWithCommits(commits) — mock с предзаданными коммитами
//   - NewMockClientWithIssue(issue) — mock с предзаданной задачей
//   - NewMockClientWithRelease(release) — mock с предзаданным релизом
//
// # Тестовые данные
//
// Функции *Data() возвращают реалистичные тестовые данные:
//   - PRData() — тестовый Pull Request
//   - CommitData() — список тестовых коммитов
//   - BranchData() — список тестовых веток
//   - IssueData() — тестовая задача
//   - ReleaseData() — тестовый релиз
//   - RepositoryData() — тестовый репозиторий
//
// # ISP (Interface Segregation Principle)
//
// MockClient реализует все role-based интерфейсы из пакета gitea:
//   - PRReader — чтение Pull Requests
//   - CommitReader — чтение коммитов
//   - FileReader — чтение файлов
//   - BranchManager — управление ветками
//   - ReleaseReader — чтение релизов
//   - IssueManager — управление задачами
//   - PRManager — управление Pull Requests
//   - RepositoryWriter — запись в репозиторий
//   - TeamReader — чтение информации о командах
//   - OrgReader — чтение информации об организациях
//
// Это позволяет использовать mock в тестах, требующих только подмножество методов,
// следуя принципу Interface Segregation.
package giteatest
