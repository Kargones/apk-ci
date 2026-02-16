package alerting

import "strings"

// RulesConfig содержит конфигурацию правил фильтрации для factory.
type RulesConfig struct {
	// MinSeverity — минимальный уровень severity ("INFO", "WARNING", "CRITICAL").
	MinSeverity string

	// ExcludeErrorCodes — коды ошибок, для которых НЕ отправляются алерты.
	ExcludeErrorCodes []string

	// IncludeErrorCodes — если задан, алерты отправляются ТОЛЬКО для этих кодов.
	IncludeErrorCodes []string

	// ExcludeCommands — команды, для которых НЕ отправляются алерты.
	ExcludeCommands []string

	// IncludeCommands — если задан, алерты отправляются ТОЛЬКО для этих команд.
	IncludeCommands []string

	// Channels — правила для конкретных каналов (переопределяют глобальные).
	Channels map[string]ChannelRulesConfig
}

// ChannelRulesConfig — правила для конкретного канала алертинга.
type ChannelRulesConfig struct {
	MinSeverity       string
	ExcludeErrorCodes []string
	IncludeErrorCodes []string
	ExcludeCommands   []string
	IncludeCommands   []string
}

// ruleConfig определяет внутренний набор правил фильтрации.
type ruleConfig struct {
	minSeverity       Severity
	excludeErrorCodes map[string]struct{}
	includeErrorCodes map[string]struct{}
	excludeCommands   map[string]struct{}
	includeCommands   map[string]struct{}
}

// RulesEngine оценивает алерты по правилам фильтрации.
type RulesEngine struct {
	global   ruleConfig
	channels map[string]ruleConfig
}

// NewRulesEngine создаёт RulesEngine из конфигурации.
func NewRulesEngine(config RulesConfig) *RulesEngine {
	engine := &RulesEngine{
		global:   buildRuleConfig(config.MinSeverity, config.ExcludeErrorCodes, config.IncludeErrorCodes, config.ExcludeCommands, config.IncludeCommands),
		channels: make(map[string]ruleConfig),
	}

	for name, ch := range config.Channels {
		engine.channels[name] = buildRuleConfig(ch.MinSeverity, ch.ExcludeErrorCodes, ch.IncludeErrorCodes, ch.ExcludeCommands, ch.IncludeCommands)
	}

	return engine
}

// Evaluate проверяет, должен ли алерт быть отправлен в указанный канал.
// Возвращает true если алерт разрешён.
//
// Если для канала задан channel override — используется ТОЛЬКО override правило,
// глобальные правила НЕ мержатся. Это означает, что channel override с
// ExcludeErrorCodes но без MinSeverity будет использовать MinSeverity=INFO (default),
// а не глобальный MinSeverity.
func (e *RulesEngine) Evaluate(alert Alert, channel string) bool {
	rule := e.global
	if channelRule, ok := e.channels[channel]; ok {
		rule = channelRule
	}

	return evaluateRule(rule, alert)
}

// evaluateRule применяет правило к алерту.
func evaluateRule(rule ruleConfig, alert Alert) bool {
	// Проверка severity: alert.Severity >= rule.minSeverity
	if alert.Severity < rule.minSeverity {
		return false
	}

	// L-9/Review #15: Проверка error code: include имеет приоритет над exclude.
	// При одновременном задании обоих include побеждает — алерт для кода из include
	// будет отправлен даже если он также в exclude. Аналогично для commands.
	if len(rule.includeErrorCodes) > 0 {
		if _, ok := rule.includeErrorCodes[alert.ErrorCode]; !ok {
			return false
		}
	} else if len(rule.excludeErrorCodes) > 0 {
		if _, ok := rule.excludeErrorCodes[alert.ErrorCode]; ok {
			return false
		}
	}

	// Проверка command: include имеет приоритет над exclude
	if len(rule.includeCommands) > 0 {
		if _, ok := rule.includeCommands[alert.Command]; !ok {
			return false
		}
	} else if len(rule.excludeCommands) > 0 {
		if _, ok := rule.excludeCommands[alert.Command]; ok {
			return false
		}
	}

	return true
}

// parseSeverity конвертирует строковое представление severity в Severity.
func parseSeverity(s string) Severity {
	switch strings.ToUpper(s) {
	case "WARNING":
		return SeverityWarning
	case "CRITICAL":
		return SeverityCritical
	default:
		return SeverityInfo
	}
}

// buildRuleConfig создаёт ruleConfig из строковых параметров.
func buildRuleConfig(minSeverity string, excludeErrors, includeErrors, excludeCommands, includeCommands []string) ruleConfig {
	return ruleConfig{
		minSeverity:       parseSeverity(minSeverity),
		excludeErrorCodes: toSet(excludeErrors),
		includeErrorCodes: toSet(includeErrors),
		excludeCommands:   toSet(excludeCommands),
		includeCommands:   toSet(includeCommands),
	}
}

// toSet конвертирует slice строк в map для быстрого lookup.
func toSet(items []string) map[string]struct{} {
	if len(items) == 0 {
		return nil
	}
	s := make(map[string]struct{}, len(items))
	for _, item := range items {
		s[item] = struct{}{}
	}
	return s
}
