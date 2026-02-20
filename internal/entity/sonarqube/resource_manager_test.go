package sonarqube

import (
	"log/slog"
	"strings"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestNewResourceManager проверяет создание нового менеджера ресурсов
func TestNewResourceManager(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir := "/tmp/test"
	
	rm := NewResourceManager(logger, tempDir)
	
	if rm == nil {
		t.Fatal("Expected non-nil resource manager")
	}
	
	if rm.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
	
	if rm.tempDir != tempDir {
		t.Error("Expected tempDir to be set correctly")
	}
	
	if rm.resources == nil {
		t.Error("Expected resources map to be initialized")
	}
	
	if rm.cleanupTimeout != 30*time.Second {
		t.Error("Expected default cleanup timeout to be 30 seconds")
	}
}

// TestResourceManagerRegisterFile проверяет регистрацию файловых ресурсов
func TestResourceManagerRegisterFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	// Создаем временный файл для тестирования
	tempFile, err := os.CreateTemp("", "test_file_*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())
	tempFile.Close()
	
	err = rm.RegisterFile("test_file", tempFile.Name(), "Test file", true)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что ресурс зарегистрирован
	resources := rm.GetResources()
	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
	
	resource, exists := resources["test_file"]
	if !exists {
		t.Error("Expected resource to be registered")
	}
	
	if resource.Type != ResourceTypeFile {
		t.Errorf("Expected resource type %s, got %s", ResourceTypeFile, resource.Type)
	}
	
	if resource.Path != tempFile.Name() {
		t.Errorf("Expected path %s, got %s", tempFile.Name(), resource.Path)
	}
}

// TestResourceManagerRegisterDirectory проверяет регистрацию директорий
func TestResourceManagerRegisterDirectory(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	// Создаем временную директорию для тестирования
	tempDir, err := os.MkdirTemp("", "test_dir_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	err = rm.RegisterDirectory("test_dir", tempDir, "Test directory", true)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что ресурс зарегистрирован
	resources := rm.GetResources()
	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
	
	resource, exists := resources["test_dir"]
	if !exists {
		t.Error("Expected resource to be registered")
	}
	
	if resource.Type != ResourceTypeDirectory {
		t.Errorf("Expected resource type %s, got %s", ResourceTypeDirectory, resource.Type)
	}
}

// TestResourceManagerRegisterProcess проверяет регистрацию процессов
func TestResourceManagerRegisterProcess(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	killFunc := func() error {
		return nil
	}
	
	err := rm.RegisterProcess("test_process", 12345, "Test process", killFunc)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что ресурс зарегистрирован
	resources := rm.GetResources()
	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
	
	resource, exists := resources["test_process"]
	if !exists {
		t.Error("Expected resource to be registered")
	}
	
	if resource.Type != ResourceTypeProcess {
		t.Errorf("Expected resource type %s, got %s", ResourceTypeProcess, resource.Type)
	}
	
	if resource.PID != 12345 {
		t.Errorf("Expected PID 12345, got %d", resource.PID)
	}
}

// TestResourceManagerRegisterGeneric проверяет регистрацию общих ресурсов
func TestResourceManagerRegisterGeneric(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	cleanupFunc := func() error {
		return nil
	}
	
	err := rm.RegisterGeneric("test_generic", "Test generic resource", cleanupFunc, true, 10)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что ресурс зарегистрирован
	resources := rm.GetResources()
	if len(resources) != 1 {
		t.Errorf("Expected 1 resource, got %d", len(resources))
	}
	
	resource, exists := resources["test_generic"]
	if !exists {
		t.Error("Expected resource to be registered")
	}
	
	if resource.Type != ResourceTypeGeneric {
		t.Errorf("Expected resource type %s, got %s", ResourceTypeGeneric, resource.Type)
	}
	
	if resource.Priority != 10 {
		t.Errorf("Expected priority 10, got %d", resource.Priority)
	}
}

// TestResourceManagerUnregisterResource проверяет отмену регистрации ресурсов
func TestResourceManagerUnregisterResource(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	// Регистрируем ресурс
	cleanupFunc := func() error { return nil }
	err := rm.RegisterGeneric("test_resource", "Test resource", cleanupFunc, false, 0)
	if err != nil {
		t.Fatalf("Failed to register resource: %v", err)
	}
	
	// Проверяем, что ресурс зарегистрирован
	if rm.GetResourceCount() != 1 {
		t.Error("Expected 1 resource to be registered")
	}
	
	// Отменяем регистрацию
	err = rm.UnregisterResource("test_resource")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что ресурс удален
	if rm.GetResourceCount() != 0 {
		t.Error("Expected 0 resources after unregistering")
	}
}

// TestResourceManagerCleanupResource проверяет очистку ресурсов
func TestResourceManagerCleanupResource(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	cleanupCalled := false
	cleanupFunc := func() error {
		cleanupCalled = true
		return nil
	}
	
	// Регистрируем ресурс
	err := rm.RegisterGeneric("test_cleanup", "Test cleanup resource", cleanupFunc, false, 0)
	if err != nil {
		t.Fatalf("Failed to register resource: %v", err)
	}
	
	// Очищаем ресурс
	err = rm.CleanupResource("test_cleanup")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что функция очистки была вызвана
	if !cleanupCalled {
		t.Error("Expected cleanup function to be called")
	}
	
	// Проверяем, что ресурс удален
	if rm.GetResourceCount() != 0 {
		t.Error("Expected 0 resources after cleanup")
	}
}

// TestResourceManagerCreateTempFile проверяет создание временных файлов
func TestResourceManagerCreateTempFile(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir, err := os.MkdirTemp("", "rm_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	rm := NewResourceManager(logger, tempDir)
	
	filePath, err := rm.CreateTempFile("test_", ".txt")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что файл создан
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Error("Expected temp file to be created")
	}
	
	// Проверяем, что файл в правильной директории
	if !strings.HasPrefix(filePath, tempDir) {
		t.Errorf("Expected file to be in temp dir %s, got %s", tempDir, filePath)
	}
	
	// Очищаем
	os.Remove(filePath)
}

// TestResourceManagerCreateTempDir проверяет создание временных директорий
func TestResourceManagerCreateTempDir(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	tempDir, err := os.MkdirTemp("", "rm_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	
	rm := NewResourceManager(logger, tempDir)
	
	dirPath, err := rm.CreateTempDir("test_dir_")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	
	// Проверяем, что директория создана
	if stat, err := os.Stat(dirPath); os.IsNotExist(err) || !stat.IsDir() {
		t.Error("Expected temp directory to be created")
	}
	
	// Проверяем, что директория в правильном месте
	if !strings.HasPrefix(dirPath, tempDir) {
		t.Errorf("Expected directory to be in temp dir %s, got %s", tempDir, dirPath)
	}
	
	// Очищаем
	os.RemoveAll(dirPath)
}

// TestResourceManagerSetCleanupTimeout проверяет установку таймаута очистки
func TestResourceManagerSetCleanupTimeout(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	newTimeout := 60 * time.Second
	rm.SetCleanupTimeout(newTimeout)
	
	if rm.cleanupTimeout != newTimeout {
		t.Errorf("Expected cleanup timeout %v, got %v", newTimeout, rm.cleanupTimeout)
	}
}

// TestResourceManagerSetAutoCleanup проверяет настройку автоматической очистки
func TestResourceManagerSetAutoCleanup(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	interval := 5 * time.Minute
	rm.SetAutoCleanup(true, interval)
	
	if !rm.autoCleanup {
		t.Error("Expected auto cleanup to be enabled")
	}
	
	if rm.cleanupInterval != interval {
		t.Errorf("Expected cleanup interval %v, got %v", interval, rm.cleanupInterval)
	}
}

// TestResourceManagerDuplicateRegistration проверяет обработку дублирующихся регистраций
func TestResourceManagerDuplicateRegistration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	rm := NewResourceManager(logger, "/tmp")
	
	cleanupFunc := func() error { return nil }
	
	// Первая регистрация
	err := rm.RegisterGeneric("duplicate_id", "First resource", cleanupFunc, false, 0)
	if err != nil {
		t.Fatalf("Failed to register first resource: %v", err)
	}
	
	// Вторая регистрация с тем же ID
	err = rm.RegisterGeneric("duplicate_id", "Second resource", cleanupFunc, false, 0)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}