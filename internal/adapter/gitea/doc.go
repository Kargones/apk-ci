// Package gitea определяет интерфейсы и типы данных для работы с Gitea API.
//
// Пакет предоставляет абстракцию над Gitea операциями, разделённую по принципу ISP
// (Interface Segregation Principle) на сфокусированные интерфейсы:
//   - PRReader — чтение информации о Pull Requests
//   - CommitReader — чтение информации о коммитах
//   - FileReader — чтение файлов из репозитория
//   - BranchManager — управление ветками
//   - ReleaseReader — чтение информации о релизах
//   - IssueManager — управление задачами
//   - PRManager — управление Pull Requests
//   - RepositoryWriter — запись в репозиторий
//   - TeamReader — чтение информации о командах
//   - OrgReader — чтение информации об организациях
//
// Композитный интерфейс Client объединяет все вышеперечисленные интерфейсы.
//
// # Типы ошибок
//
// Пакет определяет типизированные ошибки с кодами:
//   - ErrGiteaConnect — ошибка подключения к серверу
//   - ErrGiteaAPI — общая ошибка API
//   - ErrGiteaAuth — ошибка аутентификации
//   - ErrGiteaTimeout — превышение времени ожидания
//   - ErrGiteaNotFound — ресурс не найден
//   - ErrGiteaValidation — ошибка валидации
//
// Для проверки типа ошибки используйте helper функции:
// IsNotFoundError, IsAuthError, IsTimeoutError, IsConnectionError, IsAPIError.
//
// # Тестирование
//
// Для тестирования используйте пакет giteatest, который предоставляет
// MockClient с функциональными полями для гибкой настройки поведения.
//
// # Миграция с legacy
//
// TODO(#42): После реализации Client adapter удалить дублирующиеся структуры
// из internal/entity/gitea/gitea.go и использовать типы из этого пакета.
// Текущие структуры (Repository, Branch, Commit, Issue, etc.) скопированы
// из legacy для совместимости JSON-тегов с Gitea API.
package gitea
