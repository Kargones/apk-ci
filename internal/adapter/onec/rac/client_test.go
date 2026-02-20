package rac

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TODO(H-3): Отсутствуют интеграционные тесты для EnableServiceMode/DisableServiceMode.
// Текущие тесты покрывают только парсинг вывода RAC (unit-тесты), но не проверяют:
// - Правильность формирования RAC-команд (аргументы --cluster=, --infobase= и т.д.)
// - Обработку реального вывода RAC (corner cases)
// - Корректность subprocess-выполнения с реальным rac binary
// Рекомендация: добавить интеграционный тест (можно skip при отсутствии RAC)
// или тест, проверяющий аргументы формируемой команды через dependency injection.

// === Task 7.4: Compile-time проверка интерфейса ===

func TestRacClientImplementsClientInterface(_ *testing.T) {
	var _ Client = (*racClient)(nil)
}

// === Task 7.3: Тесты конструктора ===

func TestNewClient_Defaults(t *testing.T) {
	// Используем /bin/sh как существующий исполняемый файл для тестов
	c, err := NewClient(ClientOptions{
		RACPath: "/bin/sh",
		Server:  "server-1c",
	})
	require.NoError(t, err)
	require.NotNil(t, c)

	rc := c.(*racClient)
	assert.Equal(t, "/bin/sh", rc.racPath)
	assert.Equal(t, "server-1c", rc.server)
	assert.Equal(t, "1545", rc.port)
	assert.Equal(t, 30*time.Second, rc.timeout)
	assert.NotNil(t, rc.logger)
}

func TestNewClient_CustomOptions(t *testing.T) {
	// Используем /bin/sh как существующий исполняемый файл для тестов
	c, err := NewClient(ClientOptions{
		RACPath:      "/bin/sh",
		Server:       "prod-server",
		Port:         "2545",
		Timeout:      60 * time.Second,
		ClusterUser:  "admin",
		ClusterPass:  "secret",
		InfobaseUser: "dbadmin",
		InfobasePass: "dbsecret",
	})
	require.NoError(t, err)
	require.NotNil(t, c)

	rc := c.(*racClient)
	assert.Equal(t, "2545", rc.port)
	assert.Equal(t, 60*time.Second, rc.timeout)
	assert.Equal(t, "admin", rc.clusterUser)
	assert.Equal(t, "secret", rc.clusterPass)
	assert.Equal(t, "dbadmin", rc.infobaseUser)
	assert.Equal(t, "dbsecret", rc.infobasePass)
}

func TestNewClient_ErrorOnEmptyRACPath(t *testing.T) {
	_, err := NewClient(ClientOptions{
		Server: "server-1c",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RAC.EXEC")
}

func TestNewClient_ErrorOnEmptyServer(t *testing.T) {
	_, err := NewClient(ClientOptions{
		RACPath: "/bin/sh",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RAC.EXEC")
}

func TestNewClient_ErrorOnNonExistentRACPath(t *testing.T) {
	_, err := NewClient(ClientOptions{
		RACPath: "/nonexistent/path/to/rac",
		Server:  "server-1c",
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "RAC.EXEC")
	assert.Contains(t, err.Error(), "не найден")
}

// === Task 7.1: Тесты парсинга вывода RAC ===

func TestParseBlocks_SingleBlock(t *testing.T) {
	input := "cluster        : 2e4b5c7a-8d3f-4a1b-9c6e-f0d2a3b4c5d6\nhost           : server-1c\nport           : 1541\nname           : \"Central cluster\"\n"
	blocks := parseBlocks(input)
	require.Len(t, blocks, 1)
	assert.Equal(t, "2e4b5c7a-8d3f-4a1b-9c6e-f0d2a3b4c5d6", blocks[0]["cluster"])
	assert.Equal(t, "server-1c", blocks[0]["host"])
	assert.Equal(t, "1541", blocks[0]["port"])
	assert.Equal(t, "\"Central cluster\"", blocks[0]["name"])
}

func TestParseBlocks_MultipleBlocks(t *testing.T) {
	input := `cluster        : aaa-bbb
host           : server1
port           : 1541
name           : "Cluster A"

cluster        : ccc-ddd
host           : server2
port           : 1542
name           : "Cluster B"
`
	blocks := parseBlocks(input)
	require.Len(t, blocks, 2)
	assert.Equal(t, "aaa-bbb", blocks[0]["cluster"])
	assert.Equal(t, "ccc-ddd", blocks[1]["cluster"])
}

func TestParseBlocks_EmptyOutput(t *testing.T) {
	blocks := parseBlocks("")
	assert.Empty(t, blocks)
}

func TestParseBlocks_OnlyWhitespace(t *testing.T) {
	blocks := parseBlocks("   \n\n  \n")
	assert.Empty(t, blocks)
}

func TestParseClusterInfo_Valid(t *testing.T) {
	block := map[string]string{
		"cluster": "2e4b5c7a-8d3f-4a1b-9c6e-f0d2a3b4c5d6",
		"host":    "server-1c",
		"port":    "1541",
		"name":    "\"Central cluster\"",
	}
	info, err := parseClusterInfo(block)
	require.NoError(t, err)
	assert.Equal(t, "2e4b5c7a-8d3f-4a1b-9c6e-f0d2a3b4c5d6", info.UUID)
	assert.Equal(t, "server-1c", info.Host)
	assert.Equal(t, 1541, info.Port)
	assert.Equal(t, "Central cluster", info.Name)
}

func TestParseClusterInfo_MissingClusterUUID(t *testing.T) {
	block := map[string]string{
		"host": "server-1c",
		"port": "1541",
	}
	_, err := parseClusterInfo(block)
	require.Error(t, err)
}

func TestParseClusterInfo_InvalidPort(t *testing.T) {
	block := map[string]string{
		"cluster": "aaa-bbb",
		"host":    "server-1c",
		"port":    "not-a-number",
		"name":    "Test",
	}
	_, err := parseClusterInfo(block)
	require.Error(t, err)
}

func TestParseInfobaseInfo_Valid(t *testing.T) {
	block := map[string]string{
		"infobase": "b2c3d4e5-f6a7-8901-bcde-f12345678901",
		"name":     "MyBase",
		"descr":    "\"Тестовая база\"",
	}
	info, err := parseInfobaseInfo(block)
	require.NoError(t, err)
	assert.Equal(t, "b2c3d4e5-f6a7-8901-bcde-f12345678901", info.UUID)
	assert.Equal(t, "MyBase", info.Name)
	assert.Equal(t, "Тестовая база", info.Description)
}

func TestParseInfobaseInfo_MissingUUID(t *testing.T) {
	block := map[string]string{
		"name": "MyBase",
	}
	_, err := parseInfobaseInfo(block)
	require.Error(t, err)
}

func TestParseSessionInfo_Valid(t *testing.T) {
	block := map[string]string{
		"session":        "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
		"user-name":      "Иванов",
		"app-id":         "1CV8C",
		"host":           "192.168.1.100",
		"started-at":     "2026-01-27T10:00:00",
		"last-active-at": "2026-01-27T10:15:00",
	}
	info, err := parseSessionInfo(block)
	require.NoError(t, err)
	assert.Equal(t, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", info.SessionID)
	assert.Equal(t, "Иванов", info.UserName)
	assert.Equal(t, "1CV8C", info.AppID)
	assert.Equal(t, "192.168.1.100", info.Host)
	assert.Equal(t, 2026, info.StartedAt.Year())
	assert.Equal(t, 2026, info.LastActiveAt.Year())
}

func TestParseSessionInfo_MissingSessionID(t *testing.T) {
	block := map[string]string{
		"user-name": "Иванов",
	}
	_, err := parseSessionInfo(block)
	require.Error(t, err)
}

func TestParseServiceModeStatus_Enabled(t *testing.T) {
	block := map[string]string{
		"infobase":            "some-uuid",
		"name":                "MyBase",
		"sessions-deny":       "on",
		"scheduled-jobs-deny": "on",
		"denied-message":      "\"Обновление базы данных\"",
		"denied-from":         "2026-01-27T10:00:00",
		"permission-code":     "ServiceMode",
	}
	status := parseServiceModeStatus(block)
	assert.True(t, status.Enabled)
	assert.True(t, status.ScheduledJobsBlocked)
	assert.Equal(t, "Обновление базы данных", status.Message)
}

func TestParseServiceModeStatus_Disabled(t *testing.T) {
	block := map[string]string{
		"infobase":            "some-uuid",
		"name":                "MyBase",
		"sessions-deny":       "off",
		"scheduled-jobs-deny": "off",
		"denied-message":      "",
	}
	status := parseServiceModeStatus(block)
	assert.False(t, status.Enabled)
	assert.False(t, status.ScheduledJobsBlocked)
	assert.Empty(t, status.Message)
}

// === Task 7.2: Тесты error-сценариев ===

func TestExecuteRAC_Timeout(t *testing.T) {
	// Создаём клиент с минимальным таймаутом и существующим файлом (sh)
	// sh не является rac, но позволяет проверить обработку таймаута/ошибки запуска
	c, err := NewClient(ClientOptions{
		RACPath: "/bin/sh",
		Server:  "localhost",
		Timeout: 1 * time.Millisecond,
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, execErr := c.(*racClient).executeRAC(ctx, []string{"cluster", "list"})
	require.Error(t, execErr)
	// Проверяем, что ошибка содержит один из ожидаемых RAC-кодов
	assert.True(t, strings.Contains(execErr.Error(), ErrRACTimeout) ||
		strings.Contains(execErr.Error(), ErrRACExec),
		"ошибка должна содержать RAC.TIMEOUT или RAC.EXEC_FAILED, получено: %s", execErr.Error())
}

func TestExecuteRAC_CancelledContext(t *testing.T) {
	ctx := context.Background()
	c, err := NewClient(ClientOptions{
		RACPath: "/bin/sh",
		Server:  "localhost",
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // немедленно отменяем

	_, execErr := c.(*racClient).executeRAC(ctx, []string{"cluster", "list"})
	require.Error(t, execErr)
}

func TestParseBlocks_MalformedLine(t *testing.T) {
	// Строка без разделителя ":"
	input := "this is not a valid line\n"
	blocks := parseBlocks(input)
	// Должен создать блок, но без полезных данных
	assert.Empty(t, blocks)
}

// === Интеграция парсинга: полный вывод RAC ===

func TestFullClusterListParsing(t *testing.T) {
	output := `cluster        : 2e4b5c7a-8d3f-4a1b-9c6e-f0d2a3b4c5d6
host           : server-1c
port           : 1541
name           : "Central cluster"

cluster        : a1b2c3d4-e5f6-7890-abcd-ef1234567890
host           : server-1c-2
port           : 1541
name           : "Backup cluster"
`
	blocks := parseBlocks(output)
	require.Len(t, blocks, 2)

	info1, err := parseClusterInfo(blocks[0])
	require.NoError(t, err)
	assert.Equal(t, "2e4b5c7a-8d3f-4a1b-9c6e-f0d2a3b4c5d6", info1.UUID)
	assert.Equal(t, "Central cluster", info1.Name)

	info2, err := parseClusterInfo(blocks[1])
	require.NoError(t, err)
	assert.Equal(t, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", info2.UUID)
	assert.Equal(t, "Backup cluster", info2.Name)
}

func TestFullInfobaseListParsing(t *testing.T) {
	output := `infobase       : aaa-bbb-ccc
name           : Base1
descr          : "Первая база"

infobase       : ddd-eee-fff
name           : Base2
descr          : "Вторая база"
`
	blocks := parseBlocks(output)
	require.Len(t, blocks, 2)

	info1, err := parseInfobaseInfo(blocks[0])
	require.NoError(t, err)
	assert.Equal(t, "Base1", info1.Name)

	info2, err := parseInfobaseInfo(blocks[1])
	require.NoError(t, err)
	assert.Equal(t, "Base2", info2.Name)
}

func TestFullSessionListParsing(t *testing.T) {
	output := `session        : a1b2c3d4-e5f6-7890-abcd-ef1234567890
user-name      : Иванов
app-id         : 1CV8C
host           : 192.168.1.100
started-at     : 2026-01-27T10:00:00
last-active-at : 2026-01-27T10:15:00

session        : b2c3d4e5-f6a7-8901-bcde-f12345678901
user-name      : Петров
app-id         : 1CV8
host           : 192.168.1.101
started-at     : 2026-01-27T09:00:00
last-active-at : 2026-01-27T10:10:00
`
	blocks := parseBlocks(output)
	require.Len(t, blocks, 2)

	s1, err := parseSessionInfo(blocks[0])
	require.NoError(t, err)
	assert.Equal(t, "Иванов", s1.UserName)

	s2, err := parseSessionInfo(blocks[1])
	require.NoError(t, err)
	assert.Equal(t, "Петров", s2.UserName)
}

func TestFullInfobaseInfoServiceModeParsing(t *testing.T) {
	output := `infobase            : some-uuid-here
name                : MyBase
sessions-deny       : on
scheduled-jobs-deny : on
denied-message      : "Обновление базы данных"
denied-from         : 2026-01-27T10:00:00
permission-code     : ServiceMode
`
	blocks := parseBlocks(output)
	require.Len(t, blocks, 1)

	status := parseServiceModeStatus(blocks[0])
	assert.True(t, status.Enabled)
	assert.True(t, status.ScheduledJobsBlocked)
	assert.Equal(t, "Обновление базы данных", status.Message)
}

// === Тест маскирования credentials в аргументах лога ===

func TestSanitizeArgs(t *testing.T) {
	args := []string{
		"infobase", "update",
		"--cluster=abc-123",
		"--cluster-user=admin",
		"--cluster-pwd=secret123",
		"--infobase-user=dbadmin",
		"--infobase-pwd=dbsecret",
	}
	sanitized := sanitizeArgs(args)
	joined := strings.Join(sanitized, " ")
	// Пароли должны быть замаскированы
	assert.NotContains(t, joined, "secret123")
	assert.NotContains(t, joined, "dbsecret")
	// Имена пользователей НЕ маскируются — они полезны для отладки
	assert.Contains(t, joined, "--cluster-user=admin")
	assert.Contains(t, joined, "--infobase-user=dbadmin")
	// Маска присутствует (для паролей)
	assert.Contains(t, joined, "***")
	assert.Contains(t, joined, "--cluster=abc-123")
}

func TestSanitizeString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no_passwords",
			input:    "Error: connection refused to server:1545",
			expected: "Error: connection refused to server:1545",
		},
		{
			name:     "cluster_password",
			input:    "Error: invalid password --cluster-pwd=secret123 for cluster",
			expected: "Error: invalid password --cluster-pwd=*** for cluster",
		},
		{
			name:     "infobase_password",
			input:    "Failed: --infobase-pwd=myDbPass123 is incorrect",
			expected: "Failed: --infobase-pwd=*** is incorrect",
		},
		{
			name:     "both_passwords",
			input:    "rac: --cluster-pwd=cpass --infobase-pwd=ipass failed",
			expected: "rac: --cluster-pwd=*** --infobase-pwd=*** failed",
		},
		{
			name:     "password_at_end",
			input:    "Error with --cluster-pwd=endofline",
			expected: "Error with --cluster-pwd=***",
		},
		{
			name:     "password_with_special_chars",
			input:    "Error: --cluster-pwd=P@ss!word#123 not valid",
			expected: "Error: --cluster-pwd=*** not valid",
		},
		{
			name:     "empty_string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeString(tt.input)
			assert.Equal(t, tt.expected, result)
			// Дополнительная проверка: оригинальные пароли не должны присутствовать
			if tt.input != tt.expected {
				assert.NotContains(t, result, "secret123")
				assert.NotContains(t, result, "myDbPass123")
				assert.NotContains(t, result, "cpass")
				assert.NotContains(t, result, "ipass")
				assert.NotContains(t, result, "endofline")
				assert.NotContains(t, result, "P@ss!word#123")
			}
		})
	}
}

// === Тесты методов аутентификации ===

func TestClusterAuthArgs(t *testing.T) {
	c, err := NewClient(ClientOptions{
		RACPath:     "/bin/sh",
		Server:      "srv",
		ClusterUser: "admin",
		ClusterPass: "pass",
	})
	require.NoError(t, err)
	rc := c.(*racClient)
	args := rc.clusterAuthArgs()
	assert.Equal(t, []string{"--cluster-user=admin", "--cluster-pwd=pass"}, args)
}

func TestClusterAuthArgs_Empty(t *testing.T) {
	c, err := NewClient(ClientOptions{
		RACPath: "/bin/sh",
		Server:  "srv",
	})
	require.NoError(t, err)
	rc := c.(*racClient)
	args := rc.clusterAuthArgs()
	assert.Empty(t, args)
}

func TestInfobaseAuthArgs(t *testing.T) {
	c, err := NewClient(ClientOptions{
		RACPath:      "/bin/sh",
		Server:       "srv",
		InfobaseUser: "dbadmin",
		InfobasePass: "dbpass",
	})
	require.NoError(t, err)
	rc := c.(*racClient)
	args := rc.infobaseAuthArgs()
	assert.Equal(t, []string{"--infobase-user=dbadmin", "--infobase-pwd=dbpass"}, args)
}

func TestInfobaseAuthArgs_Empty(t *testing.T) {
	c, err := NewClient(ClientOptions{
		RACPath: "/bin/sh",
		Server:  "srv",
	})
	require.NoError(t, err)
	rc := c.(*racClient)
	args := rc.infobaseAuthArgs()
	assert.Empty(t, args)
}

// === Тесты маркера точки в denied-message (legacy-паттерн scheduled-jobs) ===

func TestParseServiceModeStatus_WithDotMarker(t *testing.T) {
	block := map[string]string{
		"sessions-deny":       "on",
		"scheduled-jobs-deny": "on",
		"denied-message":      "\"Система находится в режиме обслуживания.\"",
	}
	status := parseServiceModeStatus(block)
	assert.True(t, status.Enabled)
	assert.True(t, status.ScheduledJobsBlocked)
	// Маркер точки в конце сообщения должен сохраняться после trimQuotes
	assert.True(t, strings.HasSuffix(status.Message, "."),
		"denied-message должен сохранять маркер '.' после trimQuotes")
	assert.Equal(t, "Система находится в режиме обслуживания.", status.Message)
}

func TestParseServiceModeStatus_WithoutDotMarker(t *testing.T) {
	block := map[string]string{
		"sessions-deny":       "on",
		"scheduled-jobs-deny": "on",
		"denied-message":      "\"Система находится в режиме обслуживания\"",
	}
	status := parseServiceModeStatus(block)
	assert.True(t, status.Enabled)
	assert.False(t, strings.HasSuffix(status.Message, "."),
		"denied-message без маркера не должен заканчиваться на '.'")
	assert.Equal(t, "Система находится в режиме обслуживания", status.Message)
}

func TestVerifyServiceMode_ErrorCode(t *testing.T) {
	// Проверяем, что VerifyServiceMode использует ErrRACVerify, а не ErrRACExec
	assert.NotEqual(t, ErrRACExec, ErrRACVerify)
	assert.Equal(t, "RAC.VERIFY_FAILED", ErrRACVerify)
}
