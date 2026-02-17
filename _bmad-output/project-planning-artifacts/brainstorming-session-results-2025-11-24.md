# Brainstorming Session Results

**Session Date:** 2025-11-24
**Facilitator:** Brainstorming Coach
**Participant:** BMad

## Session Start

**Выбранный подход:** AI-рекомендованные техники

**Техники сессии:**
1. First Principles Thinking (creative) - возврат к фундаментальным требованиям
2. Assumption Reversal (deep) - проверка архитектурных предположений
3. SCAMPER Method (structured) - систематическое улучшение

## Executive Summary

**Topic:** Технические подходы для apk-ci

**Session Goals:** Рефакторинг архитектуры

**Techniques Used:** First Principles Thinking, Assumption Reversal, SCAMPER Method

**Total Ideas Generated:** 25+

### Key Themes Identified:

1. **Pipeline-архитектура** — переход от монолитных команд к композиции атомарных шагов
2. **Явный контракт** — Input/Output/Context как основа взаимодействия
3. **Персистентность контекста** — хранение состояния в git для независимости шагов
4. **Обратная совместимость** — постепенная миграция без поломки существующего функционала
5. **Референсные паттерны** — GitHub Actions и Tekton как проверенные модели

## Technique Sessions

### Техника 1: First Principles Thinking

#### Фундаментальные истины:

1. **Миссия:** Сквозная цепочка от заявки на изменение до готового решения в продуктиве
2. **Механизм:** Композиция команд через pipeline с переиспользованием инфраструктуры (логирование, создание баз, управление ресурсами)

#### Идеальный контракт команды:

**Вход:**
- Ресурс (репозиторий, база данных)
- Правила преобразования (например, конвертация формата)
- Данные аутентификации (токены, пароли)

**Выход:**
- Преобразованный ресурс (коммит в ветке, обновлённая база)
- Данные о результате операции

**Передача контекста:**
- Персистентный контекст (желательно в git)
- Максимальная независимость команд (возможен разрыв по времени)
- Оптимизация для гарантированных последовательностей (не клонировать повторно)

#### Текущие блокеры:
- Команды слишком связаны друг с другом
- Нет явного контракта между командами
- Конфигурация слишком жёсткая

---

### Техника 2: Assumption Reversal

#### Текущие предположения и их инверсии:

| Предположение | Инверсия | Инсайт |
|--------------|----------|--------|
| Команды — монолитные функции в `app.go` | Команды — композиция мелких шагов | **Pipeline-архитектура** |
| Конфигурация загружается из Gitea целиком | Конфигурация собирается по частям | **Локальные оверрайды + удалённые defaults** |
| Каждая команда знает о своих зависимостях | Команды не знают друг о друге | **Dependency injection через контекст** |
| Результат команды — side effect | Результат — явный артефакт | **Immutable outputs** |
| Ошибки обрабатываются внутри команды | Ошибки — часть выходного контракта | **Явная обработка на уровне pipeline** |

#### Идеи из инверсий:

1. **Step-based architecture** — разбить команды на атомарные шаги
2. **Context as first-class citizen** — структура контекста как основа pipeline
3. **Artifact registry** — хранение промежуточных результатов в git
4. **Declarative pipelines** — описание цепочек в YAML

---

### Техника 3: SCAMPER Method

#### S - Substitute (Заменить)
- Заменить прямые вызовы функций на message passing
- Заменить глобальный Config на scoped configuration per step
- Заменить entity layer на plugin system

#### C - Combine (Комбинировать)
- Объединить `app.go` функции в единый executor
- Комбинировать конфигурации: gitea + local + env + CLI args
- Совместить logging + metrics + tracing в observability layer

#### A - Adapt (Адаптировать)
- Адаптировать паттерн GitHub Actions (jobs → steps → actions)
- Адаптировать Tekton Pipelines (tasks → steps → results)
- Адаптировать Unix pipes (stdin → stdout)

#### M - Modify (Модифицировать)
- Увеличить гранулярность: 1 команда → 5-10 шагов
- Уменьшить связанность: убрать прямые импорты между командами
- Изменить формат конфига: добавить pipeline definitions

#### P - Put to other uses (Применить иначе)
- Использовать существующие entity как reusable steps
- Применить servicemode client как generic resource locker
- Использовать git2store workflow как reference pipeline

#### E - Eliminate (Убрать)
- Убрать дублирование логики между командами
- Убрать жёсткую привязку к Gitea (абстракция source)
- Убрать implicit state между шагами

#### R - Reverse (Обратить)
- Вместо push → pull (orchestrator запрашивает)
- Вместо императивного кода → декларативное описание pipeline
- Вместо monorepo команд → registry of steps

---

## Idea Categorization

### Immediate Opportunities

_Ideas ready to implement now_

1. **Определить интерфейс Step** — создать базовый контракт (Input/Output/Context)
2. **Выделить атомарные шаги из git2store** — это самый сложный workflow, идеальный кандидат
3. **Добавить структуру PipelineContext** — для передачи данных между шагами
4. **Создать слой абстракции над конфигурацией** — изолировать источник от потребителей

### Future Innovations

_Ideas requiring development/research_

1. **Declarative pipeline YAML** — описание цепочек в конфигурационных файлах
2. **Artifact registry в git** — версионирование промежуточных результатов
3. **Plugin system для steps** — динамическая загрузка шагов
4. **Observability layer** — единый logging/metrics/tracing
5. **Pipeline optimizer** — автоматическое объединение шагов для производительности

### Moonshots

_Ambitious, transformative concepts_

1. **Visual pipeline builder** — графический редактор цепочек
2. **Self-healing pipelines** — автоматическое восстановление при сбоях
3. **Distributed execution** — распределённое выполнение шагов на разных агентах
4. **AI-assisted pipeline composition** — автоматический подбор шагов под задачу

### Insights and Learnings

_Key realizations from the session_

1. **git2store — эталонный pipeline** — содержит все паттерны: clone → convert → bind → update → export → commit
2. **Контекст важнее кода** — персистентный контекст в git решает проблему разрыва по времени
3. **Существующие entity — готовые building blocks** — designer, convert, store уже атомарны
4. **GitHub Actions как референс** — проверенная модель jobs/steps/actions применима
5. **Оптимизация через гарантии** — если шаги гарантированно последовательны, можно оптимизировать

## Action Planning

### Top 3 Priority Ideas

#### #1 Priority: Определить интерфейс Step и PipelineContext

- **Rationale:** Это фундамент всей новой архитектуры — без явного контракта невозможно двигаться дальше
- **Next steps:**
  1. Создать `internal/pipeline/step.go` с интерфейсом Step
  2. Определить структуры Input, Output, Context
  3. Добавить базовую реализацию executor
  4. Написать тесты для контракта
- **Resources needed:** Изучить паттерны из Tekton и GitHub Actions

#### #2 Priority: Рефакторинг git2store в pipeline

- **Rationale:** Самый сложный workflow — если он работает как pipeline, остальные тем более
- **Next steps:**
  1. Декомпозировать git2store на атомарные шаги
  2. Реализовать каждый шаг с новым интерфейсом
  3. Создать pipeline definition для git2store
  4. Обеспечить обратную совместимость через wrapper
- **Resources needed:** Анализ текущей реализации в `internal/app/`

#### #3 Priority: Персистентный контекст в git

- **Rationale:** Решает проблему разрыва по времени между шагами и даёт аудит
- **Next steps:**
  1. Определить формат хранения контекста (YAML/JSON)
  2. Создать механизм save/load контекста
  3. Интегрировать с git (коммиты в служебную ветку)
  4. Добавить механизм resume для прерванных pipeline
- **Resources needed:** Исследовать best practices для pipeline state management

## Reflection and Follow-up

### What Worked Well

- **First Principles** выявил чёткий контракт команды (вход/выход/контекст)
- **Assumption Reversal** показал, что текущие entity уже близки к нужной атомарности
- **SCAMPER** дал конкретные направления: GitHub Actions и Tekton как референсы
- Фокус на git2store как эталонном pipeline — правильная точка входа

### Areas for Further Exploration

1. **Error handling в pipeline** — retry, fallback, compensation
2. **Параллельное выполнение шагов** — где это безопасно?
3. **Versioning контракта** — как эволюционировать Step interface
4. **Миграция существующих команд** — стратегия постепенного перехода

### Recommended Follow-up Techniques

- **Morphological Analysis** — для систематического исследования вариантов Step interface
- **Six Thinking Hats** — для оценки рисков миграции
- **Analogical Thinking** — глубже изучить Tekton, Argo Workflows, Temporal

### Questions That Emerged

1. Как обеспечить обратную совместимость с существующими Gitea Actions?
2. Нужен ли отдельный язык для pipeline definitions или достаточно YAML?
3. Как интегрировать pipeline executor с существующим main.go?
4. Какой уровень гранулярности шагов оптимален?

### Next Session Planning

- **Suggested topics:** Детальный дизайн Step interface и PipelineContext
- **Preparation needed:** Изучить исходники Tekton Pipeline и GitHub Actions runner

---

_Session facilitated using the BMAD CIS brainstorming framework_
