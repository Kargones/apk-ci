package rac

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/Kargones/apk-ci/internal/pkg/apperrors"
)

// parseBlocks разбивает вывод RAC на блоки key-value, разделённые пустыми строками.
func parseBlocks(output string) []map[string]string {
	var blocks []map[string]string
	current := make(map[string]string)

	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(current) > 0 {
				blocks = append(blocks, current)
				current = make(map[string]string)
			}
			continue
		}
		idx := strings.Index(line, ":")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		current[key] = value
	}
	if len(current) > 0 {
		blocks = append(blocks, current)
	}
	return blocks
}

// trimQuotes удаляет обрамляющие кавычки из строки.
func trimQuotes(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

// parseClusterInfo парсит блок key-value в ClusterInfo.
func parseClusterInfo(block map[string]string) (*ClusterInfo, error) {
	uuid, ok := block["cluster"]
	if !ok || uuid == "" {
		return nil, apperrors.NewAppError(ErrRACParse, "UUID кластера отсутствует в выводе", nil)
	}
	portStr := block["port"]
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, apperrors.NewAppError(ErrRACParse,
			fmt.Sprintf("невалидный порт кластера: %s", portStr), err)
	}
	return &ClusterInfo{
		UUID: uuid,
		Host: block["host"],
		Port: port,
		Name: trimQuotes(block["name"]),
	}, nil
}

// parseInfobaseInfo парсит блок key-value в InfobaseInfo.
func parseInfobaseInfo(block map[string]string) (*InfobaseInfo, error) {
	uuid, ok := block["infobase"]
	if !ok || uuid == "" {
		return nil, apperrors.NewAppError(ErrRACParse, "UUID информационной базы отсутствует в выводе", nil)
	}
	return &InfobaseInfo{
		UUID:        uuid,
		Name:        block["name"],
		Description: trimQuotes(block["descr"]),
	}, nil
}

// parseSessionInfo парсит блок key-value в SessionInfo.
func parseSessionInfo(block map[string]string) (*SessionInfo, error) {
	sessionID, ok := block["session"]
	if !ok || sessionID == "" {
		return nil, apperrors.NewAppError(ErrRACParse, "ID сессии отсутствует в выводе", nil)
	}
	info := &SessionInfo{
		SessionID: sessionID,
		UserName:  block["user-name"],
		AppID:     block["app-id"],
		Host:      block["host"],
	}
	if v, ok := block["started-at"]; ok && v != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", v); err == nil {
			info.StartedAt = t
		}
	}
	if v, ok := block["last-active-at"]; ok && v != "" {
		if t, err := time.Parse("2006-01-02T15:04:05", v); err == nil {
			info.LastActiveAt = t
		}
	}
	return info, nil
}

// parseServiceModeStatus парсит блок key-value в ServiceModeStatus.
func parseServiceModeStatus(block map[string]string) *ServiceModeStatus {
	status := &ServiceModeStatus{}
	if v, ok := block["sessions-deny"]; ok {
		status.Enabled = v == "on"
	}
	if v, ok := block["scheduled-jobs-deny"]; ok {
		status.ScheduledJobsBlocked = v == "on"
	}
	if v, ok := block["denied-message"]; ok {
		status.Message = trimQuotes(v)
	}
	return status
}
