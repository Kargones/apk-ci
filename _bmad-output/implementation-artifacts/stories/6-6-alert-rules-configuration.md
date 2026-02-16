# Story 6.6: Alert Rules Configuration (FR40)

Status: done

<!-- Note: Validation is optional. Run validate-create-story for quality check before dev-story. -->

## Story

As a DevOps-инженер,
I want настраивать правила алертинга через конфигурацию,
so that могу контролировать когда и какие алерты отправляются, фильтруя по severity, error_code и command.

## Acceptance Criteria

1. [AC1] `alerting.rules` в YAML конфигурации и env переменных — правила фильтрации алертов
2. [AC2] Правило по `severity` — фильтрация алертов по минимальному уровню критичности (INFO/WARNING/CRITICAL)
3. [AC3] Правило по `error_code` — whitelist/blacklist конкретных кодов ошибок
4. [AC4] Правило по `command` — whitelist/blacklist команд, для которых отправляются/не отправляются алерты
5. [AC5] Правила per-channel — возможность настроить разные правила для email, telegram, webhook
6. [AC6] Default behavior: если правила не настроены — все алерты отправляются (backward compatibility)
7. [AC7] `RulesEngine.Evaluate(alert, channel)` возвращает allow/deny решение
8. [AC8] Unit-тесты покрывают: severity фильтрацию, error_code whitelist/blacklist, command фильтрацию, per-channel rules, default behavior, комбинации правил
9. [AC9] Интеграция в MultiChannelAlerter — правила применяются перед отправкой в каждый канал
10. [AC10] DI provider обновлён — `ProvideAlerter` передаёт rules в factory
11. [AC11] Env переменные `BR_ALERTING_RULES_*` для базовой конфигурации (global min severity)

## Tasks / Subtasks

- [x] Task 1: Добавить RulesConfig в конфигурацию (AC: #1, #5, #6, #11)
  - [x] Subtask 1.1: Добавить `AlertRulesConfig` struct в `internal/config/config.go`
  - [x] Subtask 1.2: Добавить поле `Rules AlertRulesConfig` в `AlertingConfig`
  - [x] Subtask 1.3: Добавить env tags для `BR_ALERTING_RULES_*` переменных
  - [x] Subtask 1.4: Добавить `getDefaultAlertRulesConfig()` с defaults (все алерты проходят)

- [x] Task 2: Создать rules пакет с интерфейсами (AC: #2, #3, #4, #7)
  - [x] Subtask 2.1: Создать `internal/pkg/alerting/rules.go` с `RulesEngine` struct
  - [x] Subtask 2.2: Определить `Rule` struct с полями: MinSeverity, ErrorCodes (include/exclude), Commands (include/exclude)
  - [x] Subtask 2.3: Реализовать `Evaluate(alert Alert, channel string) bool` метод
  - [x] Subtask 2.4: Реализовать логику: global rules + per-channel override

- [x] Task 3: Реализовать логику фильтрации (AC: #2, #3, #4, #6)
  - [x] Subtask 3.1: Severity filter — alert.Severity >= rule.MinSeverity
  - [x] Subtask 3.2: ErrorCode filter — whitelist (include) имеет приоритет, blacklist (exclude) отдельно
  - [x] Subtask 3.3: Command filter — whitelist/blacklist команд
  - [x] Subtask 3.4: Default behavior — пустые правила = всё разрешено

- [x] Task 4: Интегрировать rules в alerting pipeline (AC: #9, #10)
  - [x] Subtask 4.1: Обновить `factory.go` — принимать RulesConfig, создавать RulesEngine
  - [x] Subtask 4.2: Обновить `MultiChannelAlerter` — вызывать `rules.Evaluate()` перед `Send()`
  - [x] Subtask 4.3: Обновить `ProvideAlerter` в `internal/di/providers.go` — передавать rules config

- [x] Task 5: Написать unit-тесты (AC: #8)
  - [x] Subtask 5.1: TestRulesEngine_DefaultAllowAll — пустые правила пропускают всё
  - [x] Subtask 5.2: TestRulesEngine_SeverityFilter — фильтрация по минимальному severity
  - [x] Subtask 5.3: TestRulesEngine_ErrorCodeInclude — whitelist кодов ошибок
  - [x] Subtask 5.4: TestRulesEngine_ErrorCodeExclude — blacklist кодов ошибок
  - [x] Subtask 5.5: TestRulesEngine_CommandInclude — whitelist команд
  - [x] Subtask 5.6: TestRulesEngine_CommandExclude — blacklist команд
  - [x] Subtask 5.7: TestRulesEngine_PerChannelOverride — разные правила для каналов
  - [x] Subtask 5.8: TestRulesEngine_CombinedRules — комбинация severity + error_code + command
  - [x] Subtask 5.9: TestMultiChannelAlerter_WithRules — интеграционный тест rules + multi-channel

- [x] Task 6: Валидация и регрессионное тестирование
  - [x] Subtask 6.1: Запустить все существующие тесты (`go test ./...`)
  - [x] Subtask 6.2: Запустить lint (`make lint`) или `go vet`
  - [x] Subtask 6.3: Проверить backward compatibility — без rules config всё работает как раньше

### Review Follow-ups (AI)

- [ ] [AI-Review][HIGH] DEAD CODE: RulesEngine никогда не вызывается — Alerter.Send() не интегрирован в handlers, rules engine мёртвый код [di/providers.go:127-130]
- [ ] [AI-Review][MEDIUM] Channel override полностью заменяет global rules (не merge) — при задании channel-specific minSeverity теряются global excludeCommands, может быть неинтуитивно [alerting/rules.go:Evaluate]
- [ ] [AI-Review][MEDIUM] Include/exclude приоритет — include имеет приоритет над exclude, но при одновременном задании include и exclude для разных полей (error_codes include + commands exclude) поведение может быть непредсказуемым [alerting/rules.go]
- [ ] [AI-Review][LOW] parseSeverity для unknown values возвращает SeverityInfo (0) — тихий fallback без warning может маскировать ошибки конфигурации [alerting/rules.go:parseSeverity]

### Review Follow-ups (AI Code Review #34)

- [ ] [AI-Review][HIGH] RulesEngine dead code — Alerter не интегрирован в handlers [alerting/rules.go]
- [ ] [AI-Review][MEDIUM] Channel override полностью заменяет global rules — нет merge [alerting/rules.go:Evaluate]
- [ ] [AI-Review][MEDIUM] Include/exclude priority неоднозначна для complex scenarios [alerting/rules.go]

## Dev Notes

### Архитектурные паттерны и ограничения

**Следуй паттернам из Story 6-2/6-3/6-4 (Alerting channels)** [Source: internal/pkg/alerting/]
- Rules — дополнительный слой фильтрации ПЕРЕД отправкой
- RulesEngine не заменяет RateLimiter — работает в дополнение к нему
- Цепочка: `rules.Evaluate()` → `rateLimiter.Allow()` → `channel.Send()`
- Design decision: rules НЕ возвращают ошибку — только allow/deny boolean

**Alerting НЕ интегрирован в handlers** [Source: cmd/benadis-runner/main.go]
- Alerter инициализируется через DI (App.Alerter) но НЕ вызывается в текущих handlers
- Story 6-6 НЕ решает эту проблему — только добавляет rules engine
- Интеграция alerter.Send() в handlers — отдельная задача (TODO)

### Существующие структуры (НЕ ДУБЛИРОВАТЬ)

**Alert struct** (уже существует) [Source: internal/pkg/alerting/alerter.go:36-58]

```go
type Alert struct {
    ErrorCode string
    Message   string
    TraceID   string
    Timestamp time.Time
    Command   string
    Infobase  string
    Severity  Severity // SeverityInfo=0, SeverityWarning=1, SeverityCritical=2
}
```

**Severity enum** (уже существует) [Source: internal/pkg/alerting/alerter.go:20-34]

```go
const (
    SeverityInfo     Severity = iota // "INFO"
    SeverityWarning                  // "WARNING"
    SeverityCritical                 // "CRITICAL"
)
```

**Alerter interface** (уже существует, НЕ менять) [Source: internal/pkg/alerting/alerter.go:60-82]

```go
type Alerter interface {
    Send(ctx context.Context, alert Alert) error
}
```

**MultiChannelAlerter** (уже существует, НУЖНО ИЗМЕНИТЬ) [Source: internal/pkg/alerting/multi.go]
- Сейчас отправляет во все каналы безусловно
- Нужно добавить rules check перед отправкой в каждый канал

### Новая структура AlertRulesConfig (в config.go)

```go
// AlertRulesConfig содержит настройки правил фильтрации алертов.
type AlertRulesConfig struct {
    // MinSeverity — минимальный уровень severity для отправки алерта.
    // Значения: "INFO", "WARNING", "CRITICAL". По умолчанию: "INFO" (все алерты).
    MinSeverity string `yaml:"minSeverity" env:"BR_ALERTING_RULES_MIN_SEVERITY" env-default:"INFO"`

    // ExcludeErrorCodes — коды ошибок, для которых НЕ отправляются алерты.
    ExcludeErrorCodes []string `yaml:"excludeErrorCodes" env:"BR_ALERTING_RULES_EXCLUDE_ERRORS" env-separator:","`

    // IncludeErrorCodes — если задан, алерты отправляются ТОЛЬКО для этих кодов.
    // Имеет приоритет над ExcludeErrorCodes.
    IncludeErrorCodes []string `yaml:"includeErrorCodes" env:"BR_ALERTING_RULES_INCLUDE_ERRORS" env-separator:","`

    // ExcludeCommands — команды, для которых НЕ отправляются алерты.
    ExcludeCommands []string `yaml:"excludeCommands" env:"BR_ALERTING_RULES_EXCLUDE_COMMANDS" env-separator:","`

    // IncludeCommands — если задан, алерты отправляются ТОЛЬКО для этих команд.
    // Имеет приоритет над ExcludeCommands.
    IncludeCommands []string `yaml:"includeCommands" env:"BR_ALERTING_RULES_INCLUDE_COMMANDS" env-separator:","`

    // ChannelOverrides — правила для конкретных каналов (переопределяют глобальные).
    ChannelOverrides map[string]ChannelRuleConfig `yaml:"channels"`
}

// ChannelRuleConfig — правила для конкретного канала алертинга.
type ChannelRuleConfig struct {
    MinSeverity       string   `yaml:"minSeverity"`
    ExcludeErrorCodes []string `yaml:"excludeErrorCodes"`
    IncludeErrorCodes []string `yaml:"includeErrorCodes"`
    ExcludeCommands   []string `yaml:"excludeCommands"`
    IncludeCommands   []string `yaml:"includeCommands"`
}
```

### RulesEngine реализация

```go
// internal/pkg/alerting/rules.go

// RuleConfig определяет набор правил фильтрации алертов.
type RuleConfig struct {
    MinSeverity       Severity
    ExcludeErrorCodes map[string]struct{}
    IncludeErrorCodes map[string]struct{}
    ExcludeCommands   map[string]struct{}
    IncludeCommands   map[string]struct{}
}

// RulesEngine оценивает алерты по правилам фильтрации.
type RulesEngine struct {
    global   RuleConfig
    channels map[string]RuleConfig // "email", "telegram", "webhook"
}

// NewRulesEngine создаёт RulesEngine из конфигурации.
func NewRulesEngine(config RulesConfig) *RulesEngine

// Evaluate проверяет, должен ли алерт быть отправлен в указанный канал.
// Возвращает true если алерт разрешён.
func (e *RulesEngine) Evaluate(alert Alert, channel string) bool
```

**Логика Evaluate:**
1. Определить правило: channel override если есть, иначе global
2. Проверить MinSeverity: `alert.Severity >= rule.MinSeverity`
3. Проверить ErrorCode:
   - Если IncludeErrorCodes задан — `alert.ErrorCode` должен быть в списке
   - Если ExcludeErrorCodes задан — `alert.ErrorCode` НЕ должен быть в списке
4. Проверить Command:
   - Если IncludeCommands задан — `alert.Command` должен быть в списке
   - Если ExcludeCommands задан — `alert.Command` НЕ должен быть в списке
5. Все проверки AND — все должны пройти для allow

### Обновление MultiChannelAlerter

```go
// internal/pkg/alerting/multi.go — ИЗМЕНИТЬ

type MultiChannelAlerter struct {
    channels map[string]Alerter // "email" → EmailAlerter, ...
    rules    *RulesEngine       // ДОБАВИТЬ
    logger   logging.Logger
}

func (m *MultiChannelAlerter) Send(ctx context.Context, alert Alert) error {
    for name, ch := range m.channels {
        // ДОБАВИТЬ: rules check
        if m.rules != nil && !m.rules.Evaluate(alert, name) {
            m.logger.Debug("алерт отклонён правилами",
                "channel", name,
                "error_code", alert.ErrorCode,
                "command", alert.Command,
                "severity", alert.Severity.String(),
            )
            continue
        }
        // Существующая логика отправки
        _ = ch.Send(ctx, alert)
    }
    return nil
}
```

### Обновление Factory

```go
// internal/pkg/alerting/factory.go — ИЗМЕНИТЬ

// RulesConfig содержит конфигурацию правил для factory.
type RulesConfig struct {
    MinSeverity       string
    ExcludeErrorCodes []string
    IncludeErrorCodes []string
    ExcludeCommands   []string
    IncludeCommands   []string
    Channels          map[string]ChannelRulesConfig
}

type ChannelRulesConfig struct {
    MinSeverity       string
    ExcludeErrorCodes []string
    IncludeErrorCodes []string
    ExcludeCommands   []string
    IncludeCommands   []string
}

// NewAlerter — обновить сигнатуру
func NewAlerter(config Config, rules RulesConfig, logger logging.Logger) (Alerter, error) {
    // ... существующая логика ...
    rulesEngine := NewRulesEngine(rules)
    // Передать rulesEngine в MultiChannelAlerter
}
```

### Env переменные

| Переменная | Значение по умолчанию | Описание |
|------------|----------------------|----------|
| BR_ALERTING_RULES_MIN_SEVERITY | INFO | Минимальный уровень severity (INFO/WARNING/CRITICAL) |
| BR_ALERTING_RULES_EXCLUDE_ERRORS | "" | Коды ошибок для исключения (через запятую) |
| BR_ALERTING_RULES_INCLUDE_ERRORS | "" | Только эти коды ошибок (через запятую) |
| BR_ALERTING_RULES_EXCLUDE_COMMANDS | "" | Команды для исключения (через запятую) |
| BR_ALERTING_RULES_INCLUDE_COMMANDS | "" | Только эти команды (через запятую) |

### Пример YAML конфигурации

```yaml
# app.yaml
alerting:
  enabled: true
  rules:
    minSeverity: "WARNING"  # глобально: только WARNING и CRITICAL
    excludeCommands:
      - "nr-version"        # не алертить для версии
    channels:
      email:
        minSeverity: "CRITICAL"  # email только для CRITICAL
      telegram:
        excludeErrorCodes:
          - "ERR_SONARQUBE_API"  # не спамить в telegram ошибками SonarQube
      webhook:
        includeCommands:         # webhook только для этих команд
          - "nr-dbrestore"
          - "nr-service-mode-enable"
          - "nr-service-mode-disable"
  email:
    enabled: true
    # ...
  telegram:
    enabled: true
    # ...
  webhook:
    enabled: true
    # ...
```

### Project Structure Notes

**Новые файлы:**
- `internal/pkg/alerting/rules.go` — RulesEngine struct, Evaluate(), NewRulesEngine()
- `internal/pkg/alerting/rules_test.go` — unit-тесты для rules engine

**Изменяемые файлы:**
- `internal/config/config.go` — добавить AlertRulesConfig, ChannelRuleConfig structs, обновить AlertingConfig
- `internal/pkg/alerting/factory.go` — обновить NewAlerter() для принятия RulesConfig
- `internal/pkg/alerting/multi.go` — добавить rules field, вызов Evaluate() перед Send()
- `internal/di/providers.go` — обновить ProvideAlerter() для передачи rules config

**НЕ СОЗДАВАТЬ:**
- Отдельный пакет `internal/pkg/alerting/rules/` — rules является частью alerting
- Новые interfaces — RulesEngine это struct, не interface (YAGNI)

### Dependencies

**Нет новых зависимостей** — используются только stdlib и существующие пакеты.

### Testing Strategy

**Unit Tests:**
- Table-driven тесты для всех комбинаций правил
- Test default (пустые правила) → всё проходит
- Test severity: INFO алерт при minSeverity=WARNING → deny
- Test error_code include: только указанные коды → allow/deny
- Test error_code exclude: исключённые коды → deny
- Test command include/exclude: аналогично
- Test channel override: разные правила для разных каналов
- Test combined: severity + error_code + command одновременно
- Test integration: MultiChannelAlerter с rules → правильная фильтрация

```go
// Пример теста
func TestRulesEngine_SeverityFilter(t *testing.T) {
    tests := []struct {
        name        string
        minSeverity Severity
        alert       Alert
        want        bool
    }{
        {
            name:        "critical проходит при minSeverity=WARNING",
            minSeverity: SeverityWarning,
            alert:       Alert{Severity: SeverityCritical},
            want:        true,
        },
        {
            name:        "info не проходит при minSeverity=WARNING",
            minSeverity: SeverityWarning,
            alert:       Alert{Severity: SeverityInfo},
            want:        false,
        },
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            engine := NewRulesEngine(RulesConfig{
                MinSeverity: tt.minSeverity.String(),
            })
            got := engine.Evaluate(tt.alert, "email")
            assert.Equal(t, tt.want, got)
        })
    }
}
```

### Git Intelligence (Previous Stories Learnings)

**Story 6-5 (Prometheus Metrics):**
- Config struct добавляется в AppConfig с yaml/env tags
- Factory pattern: NewCollector() возвращает enabled или nop реализацию
- DI Provider: ProvideMetricsCollector конвертирует config types
- Тесты: table-driven, httptest.NewServer для mock HTTP

**Story 6-4 (Webhook Alerting):**
- Send() всегда возвращает nil — ошибки логируются
- ctx.Done() check перед операцией
- URL маскирование в логах

**Story 6-3 (Telegram Alerting):**
- MultiChannelAlerter отправляет во все каналы
- Partial failure handling — продолжает при ошибке одного канала

**Story 6-2 (Email Alerting):**
- RateLimiter shared между каналами
- Factory создаёт MultiChannelAlerter если 2+ каналов

**Patterns to follow:**
- RulesEngine struct (не interface — YAGNI)
- Evaluate() возвращает bool, не error
- Map-based lookups для быстрой проверки error_code и command
- Default = allow all (backward compatibility)
- Per-channel overrides через map[string]RuleConfig

### Recent Commits (Git Intelligence)

```
cbd8f7d feat(metrics): add Prometheus metrics with Pushgateway support (Story 6-5)
cc18b64 feat(alerting): add webhook alerting with retry support (Story 6-4)
ba73e07 feat(alerting): add telegram alerting with multi-channel support (Story 6-3)
befd489 feat(alerting): add email alerting with SMTP support (Story 6-2)
0170888 feat(logging): add log file rotation with lumberjack (Story 6-1)
```

### Backward Compatibility

- Без rules config → все алерты проходят (как сейчас)
- NewAlerter() signature меняется — нужно обновить все вызовы
- RulesEngine == nil → пропускать rules check (multi.go)
- Нет breaking changes для существующих env переменных

### Security Considerations

- Rules не содержат секретов — нет необходимости маскировать
- ErrorCode и Command — не sensitive данные
- Логировать deny decisions на DEBUG уровне (не INFO)

### Known Limitations

- Per-channel rules в YAML но НЕ через env переменные (map не поддерживается cleanenv)
- Rate limiting и rules — независимые механизмы
- Rules evaluation in-process only (нет персистенции)
- ChannelOverrides перезаписывают global rules полностью (не merge)

### References

- [Source: internal/pkg/alerting/alerter.go] — Alert struct, Severity, Alerter interface
- [Source: internal/pkg/alerting/multi.go] — MultiChannelAlerter (нужно изменить)
- [Source: internal/pkg/alerting/factory.go] — NewAlerter factory (нужно изменить)
- [Source: internal/pkg/alerting/ratelimit.go] — RateLimiter (не менять, дополняет rules)
- [Source: internal/config/config.go:374-464] — AlertingConfig (нужно расширить)
- [Source: internal/di/providers.go:117-180] — ProvideAlerter (нужно обновить)
- [Source: _bmad-output/project-planning-artifacts/epics/epic-6-observability.md#Story-6.6] — исходные требования
- [Source: _bmad-output/project-planning-artifacts/prd.md] — FR40: правила алертинга
- [Source: _bmad-output/project-planning-artifacts/architecture.md] — архитектурные паттерны

## Dev Agent Record

### Agent Model Used

Claude Opus 4.6

### Debug Log References

- Все тесты проекта (`go test ./...`) — PASS, 0 регрессий
- `go vet ./...` — чисто

### Completion Notes List

- Реализован `RulesEngine` с `Evaluate(alert, channel)` — фильтрация по severity, error_code, command
- `AlertRulesConfig` и `ChannelRuleConfig` добавлены в `config.go` с YAML/env tags
- `MultiChannelAlerter` переведён с `[]Alerter` на `map[string]Alerter` для поддержки per-channel rules
- `NewAlerter()` factory обновлён: принимает `RulesConfig`, создаёт именованные каналы
- `ProvideAlerter` в DI обновлён — конвертирует `config.AlertRulesConfig` → `alerting.RulesConfig`
- Backward compatibility: пустые rules / nil rules = все алерты проходят
- 12 тестовых функций для rules engine: default, severity, error_code include/exclude, command include/exclude, per-channel override, combined rules, include priority, integration с MultiChannelAlerter, nil rules, parseSeverity
- Существующие тесты multi_test.go и webhook_test.go обновлены для новой сигнатуры

### Change Log

- 2026-02-05: Реализована Story 6-6 Alert Rules Configuration (FR40) — RulesEngine, per-channel rules, integration в alerting pipeline
- 2026-02-06: Code Review — исправлены H-1 (хрупкая индексация в factory), H-2 (docstring), H-3 (документация breaking change), M-1 (atomic read), M-2 (комментарий override), M-3 (defaults), L-1 (тест пустого channel)
- 2026-02-06: **[Code Review Epic-6]** H-3: добавлен TODO для интеграции Alerter в command handlers (alerter.Send() нигде не вызывается)
- 2026-02-06: **[Code Review Epic-6 #2]** M-2: улучшена документация ChannelOverrides в config.go — предупреждение о полном override (не merge).

### File List

**Новые файлы:**
- `internal/pkg/alerting/rules.go` — RulesEngine, RulesConfig, ChannelRulesConfig, Evaluate(), NewRulesEngine(), parseSeverity()
- `internal/pkg/alerting/rules_test.go` — 13 тестовых функций (44+ test cases)

**Изменённые файлы:**
- `internal/config/config.go` — добавлены AlertRulesConfig, ChannelRuleConfig structs; поле Rules в AlertingConfig; Rules default в getDefaultAlertingConfig()
- `internal/pkg/alerting/factory.go` — NewAlerter() принимает RulesConfig; именованные каналы строятся напрямую в map (без промежуточного slice); обновлён docstring
- `internal/pkg/alerting/multi.go` — channels: map[string]Alerter; rules *RulesEngine; Evaluate() перед Send()
- `internal/di/providers.go` — ProvideAlerter() конвертирует и передаёт rules config
- `internal/pkg/alerting/alerter_test.go` — обновлены вызовы NewAlerter() с RulesConfig{}; тест EmailAlerter → MultiChannelAlerter
- `internal/pkg/alerting/multi_test.go` — обновлены вызовы NewMultiChannelAlerter() на map[string]Alerter + rules; убран лишний atomic в mockAlerter
- `internal/pkg/alerting/webhook_test.go` — обновлён TestMultiChannelAlerter_AllChannels на map[string]Alerter

### Code Review #3
- **MEDIUM-3**: `MultiChannelAlerter.Send` теперь итерирует каналы в алфавитном порядке (детерминистичное поведение)

### Code Review #4
- **MEDIUM-2**: Расширена документация ChannelOverrides в config.go — добавлены примеры правильной/неправильной конфигурации с override

### Code Review #5
- **M-1**: Fail-fast валидация AlertingConfig добавлена в config.go при загрузке (validateAlertingConfig)

### Code Review #6
- **H-4**: Добавлены константы ChannelEmail, ChannelTelegram, ChannelWebhook — magic strings заменены в factory.go и log messages (alerter.go, factory.go, telegram.go, webhook.go)
- **M-2**: Добавлена проверка ctx.Done() в MultiChannelAlerter.Send() перед отправкой в каждый канал (multi.go:39-44)

### Code Review #7 (adversarial)
- **H-2**: Перенос RateLimiter из индивидуальных каналов в MultiChannelAlerter — rate limiting теперь один раз для всех каналов

### Code Review #8 (adversarial)
- Нет замечаний по Story 6-6. Код прошёл ревью без изменений.

### Review #9 — 2026-02-06 (Adversarial)

**Reviewer**: Claude Code (AI, adversarial Senior Dev review)

**Findings**: 3 HIGH, 4 MEDIUM, 3 LOW

**Issues fixed**:
- **H-1**: CRLF injection в email From/To — добавлена валидация control characters в EmailConfig.Validate() (config.go, errors.go) + тесты
- **H-2**: Отсутствие warning log при полном отказе доставки — добавлен logger.Warn() в telegram.go и webhook.go когда successCount==0 + тесты
- **H-3**: os.Hostname() не кэшировался в WebhookAlerter — hostname теперь кэшируется в конструкторе (webhook.go) + тест
- **M-1**: Magic numbers в getDefaultAlertingConfig() — заменены на alerting.DefaultXxx константы (config.go)
- **M-2**: Комментарий добавлен к validateAlertingConfig() — defense-in-depth документирована (config.go)
- **M-3**: TODO добавлен к encodeRFC2047 о RFC 2047 75-char limit (email.go)
- **M-4**: TODO добавлен для bool YAML zero-value issue (Compress, UseTLS) в config.go
- **L-1**: Success log добавлен в ActStore2db case (main.go)
- **L-2**: Комментарий о triple validation (defense-in-depth) добавлен в providers.go
- **L-3**: TODO добавлен к encodeRFC2047 о =? marker в ASCII строках (email.go)

**Decision**: All findings fixed ✅

### Adversarial Code Review #10
- M-3 fix: `multi.go` — добавлен итоговый debug лог с channels_sent/channels_skipped/channels_total
- Добавлен тест: `TestMultiChannelAlerter_SummaryLog`

### Adversarial Code Review #13
- H-3 confirmed: Alerter dead code (providers.go:127-130 TODO H-3 уже задокументирован)
- Без изменений в story 6-6

### Adversarial Code Review #15

**Findings**: 1 HIGH (shared), 1 MEDIUM, 1 LOW

**Issues fixed (code)**:
- **M-7**: `factory.go` — добавлен warning log при channel override без minSeverity (footgun prevention)
- **L-9**: `rules.go` — добавлен комментарий о приоритете include над exclude

**Issues documented (not code)**:
- **H-1** (shared): Alerter не интегрирован — rules engine мёртвый код. См. Story 6-2

### Adversarial Code Review #16

**Findings**: 1 HIGH (shared)

**Issues documented (not code)**:
- **H-9** (shared): Alerter dead code (~3000 строк) — см. Story 6-2

### Adversarial Code Review #17 (2026-02-07)

**Findings**: 1 HIGH (shared)

**Issues documented (not code)**:
- **H-1** (shared): Rules engine мёртвый код (часть alerter infrastructure). TODO H-9 в providers.go:127-130 подтверждён review #17

**Status**: done
