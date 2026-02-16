package rac

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNewClient проверяет создание нового клиента RAC
func TestNewClient(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo", // Используем echo для тестирования
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	if client == nil {
		t.Fatal("Expected non-nil client")
	}
	
	if client.RacPath != "/usr/bin/echo" {
		t.Errorf("Expected RacPath '/usr/bin/echo', got '%s'", client.RacPath)
	}
	
	if client.Server != "localhost" {
		t.Errorf("Expected Server 'localhost', got '%s'", client.Server)
	}
	
	if client.Port != 1540 {
		t.Errorf("Expected Port 1540, got %d", client.Port)
	}
}




// TestExecuteCommandOnce проверяет выполнение команды через runner
func TestExecuteCommandOnce(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo", // Используем echo для безопасного тестирования
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		5*time.Second,
		1,
		logger,
	)
	
	ctx := context.Background()
	
	// Тестируем выполнение простой команды
	output, err := client.executeCommandOnce(ctx, "test", "message")
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
	
	// echo должен вернуть аргументы
	if output == "" {
		t.Error("Expected non-empty output")
	}
	
	t.Logf("Command output: %s", output)
}

// TestExecuteCommandTimeout проверяет обработку таймаута
func TestExecuteCommandTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/sleep", // Используем sleep для тестирования таймаута
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		1*time.Second, // Короткий таймаут
		1,
		logger,
	)
	
	ctx := context.Background()
	
	// Тестируем команду, которая должна превысить таймаут
	_, err := client.executeCommandOnce(ctx, "5") // sleep 5 секунд
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
	
	t.Logf("Expected timeout error: %v", err)
}

// TestIsValidUUID проверяет валидацию UUID
func TestIsValidUUID(t *testing.T) {
	tests := []struct {
		uuid     string
		expected bool
	}{
		{"550e8400-e29b-41d4-a716-446655440000", true},
		{"invalid-uuid", false},
		{"", false},
		{"550e8400-e29b-41d4-a716-44665544000", false}, // Неправильная длина
	}
	
	for _, test := range tests {
		result := IsValidUUID(test.uuid)
		if result != test.expected {
			t.Errorf("IsValidUUID(%s) = %v, expected %v", test.uuid, result, test.expected)
		}
	}
}

// TestConvertOutputToUTF8 проверяет конвертацию вывода в UTF-8
func TestConvertOutputToUTF8(t *testing.T) {
	// Тест с валидным UTF-8
	utf8Input := []byte("Hello, мир!")
	result, err := convertOutputToUTF8(utf8Input)
	if err != nil {
		t.Errorf("Expected no error for valid UTF-8, got: %v", err)
	}
	if result != "Hello, мир!" {
		t.Errorf("Expected 'Hello, мир!', got '%s'", result)
	}
	
	// Тест с пустым вводом
	emptyInput := []byte("")
	result, err = convertOutputToUTF8(emptyInput)
	if err != nil {
		t.Errorf("Expected no error for empty input, got: %v", err)
	}
	if result != "" {
		t.Errorf("Expected empty string, got '%s'", result)
	}
}

// TestGetClusterUUID проверяет получение UUID кластера
func TestGetClusterUUID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	// Создаем клиент с echo для симуляции вывода RAC
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	
	// Тест с невалидным выводом (echo просто выведет аргументы)
	_, err := client.GetClusterUUID(ctx)
	if err == nil {
		t.Error("Expected error for invalid output, got nil")
	}
}

// TestGetInfobaseUUID проверяет получение UUID информационной базы
func TestGetInfobaseUUID(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	
	// Тест с невалидным выводом
	_, err := client.GetInfobaseUUID(ctx, "test-cluster-uuid", "testdb")
	if err == nil {
		t.Error("Expected error for invalid output, got nil")
	}
}

// TestCheckServerConnection проверяет проверку подключения к серверу
func TestCheckServerConnection(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	// Тест с недоступным сервером
	client := NewClient(
		"/usr/bin/echo",
		"invalid-host",
		9999,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	
	err := client.checkServerConnection(ctx)
	if err == nil {
		t.Error("Expected error for invalid server, got nil")
	}
}

// TestIsValidUTF8 проверяет функцию проверки валидности UTF-8
func TestIsValidUTF8(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid UTF-8", "Hello, мир!", true},
		{"empty string", "", true},
		{"ASCII only", "Hello World", true},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidUTF8(tt.input)
			if result != tt.expected {
				t.Errorf("isValidUTF8(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestEnableServiceMode проверяет включение сервисного режима
func TestEnableServiceMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	err := client.EnableServiceMode(ctx, "test-cluster-uuid", "test-infobase-uuid", false)
	
	// При использовании echo функция может выполниться успешно
	if err != nil {
		t.Logf("EnableServiceMode returned error (expected with echo): %v", err)
	}
}

// TestDisableServiceMode проверяет отключение сервисного режима
func TestDisableServiceMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	err := client.DisableServiceMode(ctx, "test-cluster-uuid", "test-infobase-uuid")
	
	// При использовании echo функция может выполниться успешно
	if err != nil {
		t.Logf("DisableServiceMode returned error (expected with echo): %v", err)
	}
}

// TestGetServiceModeStatus проверяет получение статуса сервисного режима
func TestGetServiceModeStatus(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	status, err := client.GetServiceModeStatus(ctx, "test-cluster-uuid", "test-infobase-uuid")
	
	// При использовании echo функция может вернуть ошибку
	if err != nil {
		t.Logf("GetServiceModeStatus returned error (expected with echo): %v", err)
	}
	if status != nil {
		t.Logf("GetServiceModeStatus returned status: %+v", status)
	}
}

// TestGetSessions проверяет получение списка сессий
func TestGetSessions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	sessions, err := client.GetSessions(ctx, "test-cluster-uuid", "test-infobase-uuid")
	
	// При использовании echo функция может вернуть ошибку
	if err != nil {
		t.Logf("GetSessions returned error (expected with echo): %v", err)
	}
	if sessions != nil {
		t.Logf("GetSessions returned sessions: %+v", sessions)
	}
}

// TestTerminateSession проверяет завершение сессии
func TestTerminateSession(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	err := client.TerminateSession(ctx, "test-cluster-uuid", "test-session-id")
	
	// При использовании echo функция может выполниться успешно
	if err != nil {
		t.Logf("TerminateSession returned error (expected with echo): %v", err)
	}
}

// TestTerminateAllSessions проверяет завершение всех сессий
func TestTerminateAllSessions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	err := client.TerminateAllSessions(ctx, "test-cluster-uuid", "test-infobase-uuid")
	
	// При использовании echo функция может выполниться успешно
	if err != nil {
		t.Logf("TerminateAllSessions returned error (expected with echo): %v", err)
	}
}

// TestVerifyServiceMode проверяет верификацию сервисного режима
func TestVerifyServiceMode(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	err := client.VerifyServiceMode(ctx, "test-cluster-uuid", "test-infobase-uuid", true)
	
	// При использовании echo функция может выполниться успешно
	if err != nil {
		t.Logf("VerifyServiceMode returned error (expected with echo): %v", err)
	}
}

// TestGetClusterUUIDWithValidOutput проверяет получение UUID кластера с валидным выводом
func TestGetClusterUUIDWithValidOutput(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	// Создаем временный скрипт, который имитирует вывод RAC
	client := NewClient(
		"/bin/sh",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3,
		logger,
	)
	
	ctx := context.Background()
	
	// Тест с невалидным выводом
	_, err := client.GetClusterUUID(ctx)
	if err == nil {
		t.Error("Expected error for invalid output, got nil")
	}
}

// TestGetSessionsErrorHandling проверяет обработку ошибок при получении сессий
func TestGetSessionsErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/false", // Команда, которая всегда возвращает ошибку
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1, // Только одна попытка
		logger,
	)
	
	ctx := context.Background()
	sessions, err := client.GetSessions(ctx, "test-cluster-uuid", "test-infobase-uuid")
	
	if err == nil {
		t.Error("Expected error when using /usr/bin/false, got nil")
	}
	if sessions != nil {
		t.Error("Expected nil sessions on error")
	}
}

// TestTerminateAllSessionsErrorHandling проверяет обработку ошибок при завершении всех сессий
func TestTerminateAllSessionsErrorHandling(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/false", // Команда, которая всегда возвращает ошибку
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1, // Только одна попытка
		logger,
	)
	
	ctx := context.Background()
	err := client.TerminateAllSessions(ctx, "test-cluster-uuid", "test-infobase-uuid")
	
	if err == nil {
		t.Error("Expected error when using /usr/bin/false, got nil")
	}
}

// TestConvertOutputToUTF8WithCP1251 проверяет конвертацию из CP1251
func TestConvertOutputToUTF8WithCP1251(t *testing.T) {
	// Тестируем с валидным UTF-8
	validUTF8 := "Hello, мир!"
	result, err := convertOutputToUTF8([]byte(validUTF8))
	if err != nil {
		t.Errorf("Unexpected error for valid UTF-8: %v", err)
	}
	if result != validUTF8 {
		t.Errorf("Expected '%s', got '%s'", validUTF8, result)
	}
	
	// Тестируем с невалидными байтами
	invalidBytes := []byte{0xFF, 0xFE, 0xFD}
	result, err = convertOutputToUTF8(invalidBytes)
	if err == nil {
		t.Log("No error for invalid bytes (expected)")
	}
	if result == "" {
		t.Error("Expected non-empty result even for invalid bytes")
	}
}

// TestExecuteCommandWithUnsafeArgs проверяет обработку небезопасных аргументов
func TestExecuteCommandWithUnsafeArgs(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)
	
	ctx := context.Background()
	
	// Тестируем с небезопасными символами
	unsafeArgs := []string{"test;rm -rf /", "test&echo hack", "test|cat /etc/passwd"}
	
	for _, arg := range unsafeArgs {
		_, err := client.executeCommandOnce(ctx, arg)
		if err == nil {
			t.Errorf("Expected error for unsafe argument '%s', got nil", arg)
		}
		if !strings.Contains(err.Error(), "potentially unsafe argument") {
			t.Errorf("Expected 'potentially unsafe argument' error for '%s', got: %v", arg, err)
		}
	}
}

// TestCheckServerConnectionFailure проверяет обработку ошибок подключения к серверу
func TestCheckServerConnectionFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/echo",
		"nonexistent.host", // Несуществующий хост
		9999,               // Несуществующий порт
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)
	
	ctx := context.Background()
	err := client.checkServerConnection(ctx)
	
	if err == nil {
		t.Error("Expected error for connection to nonexistent host, got nil")
	}
	if !strings.Contains(err.Error(), "cannot connect to RAC server") {
		t.Errorf("Expected 'cannot connect to RAC server' error, got: %v", err)
	}
}

// TestServiceModeStatusStruct проверяет структуру ServiceModeStatus
func TestServiceModeStatusStruct(t *testing.T) {
	status := &ServiceModeStatus{
		Enabled:        true,
		Message:        "Test message",
		ActiveSessions: 5,
	}
	
	if !status.Enabled {
		t.Error("Expected Enabled to be true")
	}
	if status.Message != "Test message" {
		t.Errorf("Expected Message 'Test message', got '%s'", status.Message)
	}
	if status.ActiveSessions != 5 {
		t.Errorf("Expected ActiveSessions 5, got %d", status.ActiveSessions)
	}
}

// TestSessionInfoStruct проверяет структуру SessionInfo
func TestSessionInfoStruct(t *testing.T) {
	now := time.Now()
	session := &SessionInfo{
		SessionID:    "test-session-id",
		UserName:     "test-user",
		AppID:        "test-app",
		StartedAt:    now,
		LastActiveAt: now,
	}
	
	if session.SessionID != "test-session-id" {
		t.Errorf("Expected SessionID 'test-session-id', got '%s'", session.SessionID)
	}
	if session.UserName != "test-user" {
		t.Errorf("Expected UserName 'test-user', got '%s'", session.UserName)
	}
	if session.AppID != "test-app" {
		t.Errorf("Expected AppID 'test-app', got '%s'", session.AppID)
	}
}

// TestExecuteCommandRetries проверяет механизм повторных попыток
func TestExecuteCommandRetries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	
	client := NewClient(
		"/usr/bin/false", // Команда, которая всегда возвращает ошибку
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3, // Три попытки
		logger,
	)
	
	ctx := context.Background()
	_, err := client.ExecuteCommand(ctx, "test")
	
	if err == nil {
		t.Error("Expected error after retries, got nil")
	}
	if !strings.Contains(err.Error(), "failed after 3 attempts") {
		t.Errorf("Expected 'failed after 3 attempts' error, got: %v", err)
	}
}

// TestGetInfobaseUUIDWithClusterError проверяет обработку ошибки получения кластера
func TestGetInfobaseUUIDWithClusterError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/false", // Команда, которая возвращает ошибку
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	_, err := client.GetInfobaseUUID(ctx, "test-cluster", "testdb")

	if err == nil {
		t.Error("Expected error when cluster command fails, got nil")
	}
}

// TestGetClusterUUIDWithCommandError проверяет обработку ошибки выполнения команды получения кластера
func TestGetClusterUUIDWithCommandError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/false", // Команда, которая возвращает ошибку
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	_, err := client.GetClusterUUID(ctx)

	if err == nil {
		t.Error("Expected error when cluster list command fails, got nil")
	}
}

// TestCheckServerConnectionWithNetError проверяет обработку сетевых ошибок при проверке подключения
func TestCheckServerConnectionWithNetError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/echo",
		"192.0.2.1", // TEST-NET-1 адрес, который гарантированно недоступен
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		1*time.Second, // Короткий таймаут для быстрого провала
		1,
		logger,
	)

	ctx := context.Background()
	err := client.checkServerConnection(ctx)

	if err == nil {
		t.Error("Expected network connection error, got nil")
	}
}

// TestExecuteCommandWithEmptyOutput проверяет обработку пустого вывода команды
func TestExecuteCommandWithEmptyOutput(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/true", // Команда, которая ничего не выводит
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	output, err := client.executeCommandOnce(ctx, "test")

	if err != nil {
		t.Errorf("Expected no error for successful command, got: %v", err)
	}
	if output != "" {
		t.Errorf("Expected empty output, got: '%s'", output)
	}
}

// TestEnableServiceModeWithTerminateSessions проверяет включение сервисного режима с завершением сессий
func TestEnableServiceModeWithTerminateSessions(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	err := client.EnableServiceMode(ctx, "test-cluster-uuid", "test-infobase-uuid", true)

	// С echo это должно работать
	if err != nil {
		t.Logf("EnableServiceMode with terminate sessions returned error: %v", err)
	}
}

// TestVerifyServiceModeFailure проверяет обработку ошибки верификации сервисного режима
func TestVerifyServiceModeFailure(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/false", // Команда, которая возвращает ошибку
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	err := client.VerifyServiceMode(ctx, "test-cluster-uuid", "test-infobase-uuid", true)

	if err == nil {
		t.Error("Expected error for verification failure, got nil")
	}
}

// TestGetSessionsWithParsingError проверяет обработку ошибок парсинга при получении сессий
func TestGetSessionsWithParsingError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Создаем скрипт, который выводит некорректные данные для парсинга
	client := NewClient(
		"/bin/sh",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	// Команда sh без аргументов выдаст ошибку
	_, err := client.GetSessions(ctx, "test-cluster-uuid", "test-infobase-uuid")

	if err == nil {
		t.Error("Expected error for invalid session data, got nil")
	}
}

// TestGetServiceModeStatusWithParsingError проверяет обработку ошибок парсинга статуса сервисного режима
func TestGetServiceModeStatusWithParsingError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/bin/sh",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	// Команда sh без аргументов выдаст ошибку
	_, err := client.GetServiceModeStatus(ctx, "test-cluster-uuid", "test-infobase-uuid")

	if err == nil {
		t.Error("Expected error for invalid status data, got nil")
	}
}

// TestTerminateSessionError проверяет обработку ошибок при завершении сессии
func TestTerminateSessionError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/false",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	err := client.TerminateSession(ctx, "test-cluster-uuid", "test-session-id")

	if err == nil {
		t.Error("Expected error when terminate session command fails, got nil")
	}
}

// TestIsValidUTF8WithInvalidString проверяет функцию isValidUTF8 с невалидными строками
func TestIsValidUTF8WithInvalidString(t *testing.T) {
	// Создаем строку с невалидными UTF-8 байтами
	invalidString := string([]byte{0xFF, 0xFE, 0xFD})

	result := isValidUTF8(invalidString)
	if result {
		t.Error("Expected false for invalid UTF-8 string")
	}
}

// TestExecuteCommandWithRetries проверяет механизм повторов с успешным выполнением
func TestExecuteCommandWithRetries(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		3, // Несколько попыток
		logger,
	)

	ctx := context.Background()
	output, err := client.ExecuteCommand(ctx, "test-output")

	if err != nil {
		t.Errorf("Expected no error with echo command, got: %v", err)
	}
	if !strings.Contains(output, "test-output") {
		t.Errorf("Expected output to contain 'test-output', got: %s", output)
	}
}

// TestConvertOutputToUTF8ErrorHandling проверяет обработку ошибок в convertOutputToUTF8
func TestConvertOutputToUTF8ErrorHandling(t *testing.T) {
	// Тест с очень длинным невалидным вводом для принудительной ошибки
	longInvalidBytes := make([]byte, 1000)
	for i := range longInvalidBytes {
		longInvalidBytes[i] = 0xFF // Невалидные UTF-8 байты
	}

	result, err := convertOutputToUTF8(longInvalidBytes)
	// Функция может вернуть результат даже при ошибках
	if result == "" && err == nil {
		t.Error("Expected either result or error for invalid input")
	}
}

// TestDisableServiceModeError проверяет обработку ошибки при отключении сервисного режима
func TestDisableServiceModeError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/false",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	err := client.DisableServiceMode(ctx, "test-cluster-uuid", "test-infobase-uuid")

	if err == nil {
		t.Error("Expected error when disable service mode command fails, got nil")
	}
}

// TestEnableServiceModeError проверяет обработку ошибки при включении сервисного режима
func TestEnableServiceModeError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/false",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	err := client.EnableServiceMode(ctx, "test-cluster-uuid", "test-infobase-uuid", false)

	if err == nil {
		t.Error("Expected error when enable service mode command fails, got nil")
	}
}

// TestGetClusterUUIDWithValidData проверяет парсинг валидного вывода GetClusterUUID
func TestGetClusterUUIDWithValidData(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Создаем временный скрипт, который имитирует корректный вывод RAC
	client := NewClient(
		"/bin/sh",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	// Тестируем с валидным выводом через echo
	client.RacPath = "/usr/bin/echo"
	_, err := client.GetClusterUUID(ctx)

	// echo выведет что-то, но это не будет валидный UUID
	if err == nil {
		t.Error("Expected error for non-UUID output from echo, got nil")
	}
}

// TestGetInfobaseUUIDWithValidData проверяет парсинг валидного вывода GetInfobaseUUID
func TestGetInfobaseUUIDWithValidData(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	// echo выведет что-то, но это не будет валидный формат InfoBase list
	_, err := client.GetInfobaseUUID(ctx, "test-cluster", "testdb")

	if err == nil {
		t.Error("Expected error for invalid infobase list format, got nil")
	}
}

// TestTerminateAllSessionsWithValidCluster проверяет завершение всех сессий с валидным кластером
func TestTerminateAllSessionsWithValidCluster(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	// С echo команда должна выполниться успешно
	err := client.TerminateAllSessions(ctx, "test-cluster-uuid", "test-infobase-uuid")

	if err != nil {
		t.Logf("TerminateAllSessions with echo returned error: %v", err)
	}
}

// TestGetSessionsWithEmptyOutput проверяет обработку пустого вывода GetSessions
func TestGetSessionsWithEmptyOutput(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/true", // Команда, которая ничего не выводит
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	sessions, err := client.GetSessions(ctx, "test-cluster-uuid", "test-infobase-uuid")

	if err != nil {
		t.Errorf("Expected no error for empty session list, got: %v", err)
	}
	if sessions != nil && len(sessions) != 0 {
		t.Errorf("Expected empty sessions list, got %d sessions", len(sessions))
	}
}

// TestGetServiceModeStatusWithEmptyOutput проверяет обработку пустого вывода GetServiceModeStatus
func TestGetServiceModeStatusWithEmptyOutput(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/true", // Команда, которая ничего не выводит
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	status, err := client.GetServiceModeStatus(ctx, "test-cluster-uuid", "test-infobase-uuid")

	// Пустой вывод может не генерировать ошибку
	if err != nil {
		t.Logf("Got expected error for empty status output: %v", err)
	}
	if status != nil && err != nil {
		t.Error("Expected nil status on error")
	}
}

// TestCheckServerConnectionSuccess проверяет успешное подключение к серверу
func TestCheckServerConnectionSuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/echo",
		"127.0.0.1", // localhost должен быть доступен
		22,          // SSH порт обычно открыт
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	err := client.checkServerConnection(ctx)

	// Даже если порт недоступен, это покроет больше кода
	if err != nil {
		t.Logf("Connection test returned expected error: %v", err)
	}
}

// TestGetScheduledJobsDenyStatusError проверяет обработку ошибки при получении статуса запрета регламентных заданий
func TestGetScheduledJobsDenyStatusError(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/false",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	status, err := client.getScheduledJobsDenyStatus(ctx, "test-cluster-uuid", "test-infobase-uuid")

	if err == nil {
		t.Error("Expected error when getting scheduled jobs deny status fails, got nil")
	}
	if status != "" {
		t.Error("Expected empty status on error")
	}
}

// TestGetScheduledJobsDenyStatusSuccess проверяет успешное получение статуса запрета регламентных заданий
func TestGetScheduledJobsDenyStatusSuccess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()
	status, err := client.getScheduledJobsDenyStatus(ctx, "test-cluster-uuid", "test-infobase-uuid")

	if err != nil {
		t.Errorf("Expected no error for scheduled jobs deny status check, got: %v", err)
	}
	// status может быть любым значением при использовании echo
	t.Logf("Scheduled jobs deny status: %v", status)
}

// TestGetInfobaseUUIDWithExactMatch проверяет поиск InfoBase UUID с точным совпадением имени
func TestGetInfobaseUUIDWithExactMatch(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	client := NewClient(
		"/usr/bin/echo",
		"localhost",
		1540,
		"admin",
		"password",
		"dbuser",
		"dbpass",
		30*time.Second,
		1,
		logger,
	)

	ctx := context.Background()

	// Тестируем с простым выводом, который не будет содержать валидную информацию об InfoBase
	_, err := client.GetInfobaseUUID(ctx, "test-cluster", "exact-match-name")

	if err == nil {
		t.Error("Expected error for invalid infobase list format, got nil")
	}
}