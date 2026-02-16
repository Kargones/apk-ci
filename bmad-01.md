Шаг 1: Архитектурная валидация

  /bmad:bmm:agents:architect

  Задача для Winston:
  Проанализируй Epic-0 (extension-publish) из bdocs/epics/epic-0-extension-publish.md.
  Создай или дополни архитектурный документ, включающий:
  - Диаграмму компонентов
  - API контракты для новых методов Gitea
  - Схему данных (структура Release)
  - Точки интеграции с существующим кодом

  ---
  Шаг 2: Валидация готовности к реализации (опционально)

  /bmad:bmm:agents:pm

  Затем выбрать:
  4  (или *implementation-readiness)

  Цель: Проверить выравнивание PRD ↔ Architecture ↔ Epic Stories

  ---
  Шаг 3: Подготовка Story-файлов для разработки

  /bmad:bmm:agents:sm

  Задача для Bob:
  Создай developer-ready story файлы для Epic-0 из bdocs/epics/epic-0-extension-publish.md.
  Для каждой из 8 stories создай файл в bdocs/stories/ с:
  - Story Context XML
  - Детальными Acceptance Criteria
  - Tasks и Subtasks
  - Technical Notes
  - Definition of Done

  Ожидаемый результат — 8 файлов:
  bdocs/stories/epic-0/
  ├── story-0.1-gitea-releases-api.md
  ├── story-0.2-gitea-search-repos.md
  ├── story-0.3-find-subscribed-repos.md
  ├── story-0.4-sync-extension-dir.md
  ├── story-0.5-create-pr-with-info.md
  ├── story-0.6-integrate-command.md
  ├── story-0.7-error-handling.md
  └── story-0.8-tests.md

  ---
  Шаг 4: Разработка (итеративно для каждой Story)

  /bmad:bmm:agents:dev

  Последовательность реализации:

  | Порядок | Story | Файл      | Задача                                           |
  |---------|-------|-----------|--------------------------------------------------|
  | 1       | 0.1   | story-0.1 | Gitea API: GetLatestRelease(), GetReleaseByTag() |
  | 2       | 0.2   | story-0.2 | Gitea API: SearchAllRepos() с пагинацией         |
  | 3       | 0.3   | story-0.3 | FindSubscribedRepositories()                     |
  | 4       | 0.4   | story-0.4 | SyncExtensionDirectory()                         |
  | 5       | 0.5   | story-0.5 | Расширение CreatePR()                            |
  | 6       | 0.6   | story-0.6 | Константа + main.go + ExtensionPublish()         |
  | 7       | 0.7   | story-0.7 | Error handling + отчётность                      |
  | 8       | 0.8   | story-0.8 | Unit + Integration тесты                         |

  Для каждой Story вызываем Dev:
  /bmad:bmm:agents:dev

  Команда Amelia:
  Реализуй Story 0.X из файла bdocs/stories/epic-0/story-0.X-....md
  Следуй Story Context XML и Acceptance Criteria.

  ---
  Шаг 5: Тестирование

  /bmad:bmm:agents:tea

  Задача для Murat:
  Проверь тестовое покрытие для Epic-0 (extension-publish).
  Убедись, что:
  - Покрытие >80% для internal/app/extension_publish.go
  - Есть unit-тесты для новых методов Gitea API
  - Есть integration test с httptest
  - Все тесты проходят: make test

  ---
  Шаг 6: Финальная валидация

  /bmad:bmm:agents:pm

  Затем:
  4  (*implementation-readiness)

  Цель: Подтвердить Definition of Done для Epic-0

  ---
  📝 Краткая шпаргалка команд

  # 1. Архитектура
  /bmad:bmm:agents:architect
  # → Создать архитектуру для Epic-0

  # 2. Story-файлы  
  /bmad:bmm:agents:sm
  # → Создать 8 story-файлов в bdocs/stories/epic-0/

  # 3. Разработка (повторить для каждой story)
  /bmad:bmm:agents:dev
  # → Реализовать Story 0.1
  # → Реализовать Story 0.2
  # → ...
  # → Реализовать Story 0.8

  # 4. Тесты
  /bmad:bmm:agents:tea
  # → Валидация покрытия и качества тестов

  # 5. Финальная проверка
  /bmad:bmm:agents:pm → 4
  # → Implementation Readiness Check

  ---
  ⚡ Альтернатива: Quick Flow (для опытных)

  Если хочешь ускорить процесс и работать в режиме "один разработчик делает всё":

  /bmad:bmm:agents:quick-flow-solo-dev

  Barry (Quick Flow Solo Dev) объединит планирование и реализацию в один поток.

  Доступные действия:

  1. **[M] menu* — Показать меню
  2. **[S] develop-story* — Выполнить Dev Story workflow
  3. **[R] code-review* — Code Review (рекомендуется в свежем контексте)
  4. **[D] dismiss* — Завершить сессию

  /bmad:bmm:agents:dev *develop-story story-0-
  /bmad:bmm:agents:dev *code-review story-0-