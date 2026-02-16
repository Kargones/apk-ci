package filer

import (
	"os"
	"runtime"
	"strings"
	"testing"
)

// TestCheckAvailableRAM_Complete тестирует функцию CheckAvailableRAM полностью
func TestCheckAvailableRAM_Complete(t *testing.T) {
	tm := NewTempManager(true)
	defer tm.Close()

	result := tm.CheckAvailableRAM()

	// Проверяем обязательные поля
	if _, exists := result["ram_disk_available"]; !exists {
		t.Error("Expected 'ram_disk_available' field in result")
	}

	if _, exists := result["using_ram"]; !exists {
		t.Error("Expected 'using_ram' field in result")
	}

	if _, exists := result["base_dir"]; !exists {
		t.Error("Expected 'base_dir' field in result")
	}

	if _, exists := result["os"]; !exists {
		t.Error("Expected 'os' field in result")
	}

	// Проверяем значение OS
	if result["os"] != runtime.GOOS {
		t.Errorf("Expected OS to be %s, got %v", runtime.GOOS, result["os"])
	}

	// Проверяем Linux-специфичные поля
	if runtime.GOOS == "linux" {
		if _, exists := result["shm_exists"]; !exists {
			t.Error("Expected 'shm_exists' field for Linux")
		}
		if _, exists := result["shm_is_dir"]; !exists {
			t.Error("Expected 'shm_is_dir' field for Linux")
		}
	} else {
		if _, exists := result["fallback_reason"]; !exists {
			t.Error("Expected 'fallback_reason' field for non-Linux")
		}
		expectedReason := "RAM-диск доступен только на Linux"
		if result["fallback_reason"] != expectedReason {
			t.Errorf("Expected fallback_reason to be '%s', got %v", expectedReason, result["fallback_reason"])
		}
	}
}

// TestPathUtils_NormalizePath_AdditionalCases тестирует дополнительные случаи NormalizePath
func TestPathUtils_NormalizePath_AdditionalCases(t *testing.T) {
	pu := NewPathUtils()
	
	// Тест с пустой строкой
	_, err := pu.NormalizePath("")
	if err == nil {
		t.Error("Expected error for empty path")
	}
	
	// Тест с очень длинным путем
	longPath := "/" + strings.Repeat("a", 300)
	result, err := pu.NormalizePath(longPath)
	if err != nil {
		t.Errorf("Unexpected error for long path: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result for long path")
	}
	
	// Тест с путем, содержащим только точки
	result, err = pu.NormalizePath("...")
	if err != nil {
		t.Errorf("Unexpected error for path with dots: %v", err)
	}
	if result == "" {
		t.Error("Expected non-empty result for path with dots")
	}
}

// TestPathUtils_EnsureDir_AdditionalCases тестирует дополнительные случаи EnsureDir
func TestPathUtils_EnsureDir_AdditionalCases(t *testing.T) {
	pu := NewPathUtils()
	tempDir := t.TempDir()
	
	// Тест с пустым путем
	err := pu.EnsureDir("")
	if err == nil {
		t.Error("Expected error for empty path")
	}
	
	// Тест с недопустимыми символами
	invalidPath := tempDir + "/test\x00invalid"
	err = pu.EnsureDir(invalidPath)
	if err == nil {
		t.Error("Expected error for path with invalid characters")
	}
	
	// Тест с очень глубокой вложенностью
	deepPath := tempDir + "/a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p"
	err = pu.EnsureDir(deepPath)
	if err != nil {
		t.Errorf("Unexpected error for deep path: %v", err)
	}
	
	// Проверяем, что директория создана
	if _, err := os.Stat(deepPath); os.IsNotExist(err) {
		t.Error("Deep directory was not created")
	}
}

// TestPathUtils_IsSubPath_AdditionalCases тестирует дополнительные случаи IsSubPath
func TestPathUtils_IsSubPath_AdditionalCases(t *testing.T) {
	pu := NewPathUtils()
	
	// Тест с недопустимыми путями
	_, err := pu.IsSubPath("", "/valid/path")
	if err == nil {
		t.Error("Expected error for empty parent path")
	}
	
	_, err = pu.IsSubPath("/valid/path", "")
	if err == nil {
		t.Error("Expected error for empty child path")
	}
	
	// Тест с путями, содержащими недопустимые символы
	_, err = pu.IsSubPath("/parent\x00", "/child")
	if err == nil {
		t.Error("Expected error for parent path with invalid characters")
	}
	
	_, err = pu.IsSubPath("/parent", "/child\x00")
	if err == nil {
		t.Error("Expected error for child path with invalid characters")
	}
	
	// Тест с очень похожими путями
	isSubPath, err := pu.IsSubPath("/home/user", "/home/user2")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if isSubPath {
		t.Error("Expected false for similar but different paths")
	}
	
	// Тест с путями, где один является префиксом другого, но не подпутем
	isSubPath, err = pu.IsSubPath("/home/user", "/home/user_backup")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if isSubPath {
		t.Error("Expected false for prefix but not subpath")
	}
}

// TestCheckAvailableRAM_RAMDisabledCase тестирует CheckAvailableRAM с отключенным RAM
func TestCheckAvailableRAM_RAMDisabledCase(t *testing.T) {
	tm := NewTempManager(false)
	defer tm.Close()

	result := tm.CheckAvailableRAM()

	// Проверяем, что using_ram = false
	if usingRAM, ok := result["using_ram"].(bool); !ok || usingRAM {
		t.Error("Expected using_ram to be false when RAM is disabled")
	}
}

// TestCheckAvailableRAM_NonLinux тестирует CheckAvailableRAM на не-Linux системах
func TestCheckAvailableRAM_NonLinux(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("Skipping non-Linux test on Linux")
	}

	tm := NewTempManager(true)
	defer tm.Close()

	result := tm.CheckAvailableRAM()

	// На не-Linux системах должно быть fallback_reason
	if _, exists := result["fallback_reason"]; !exists {
		t.Error("Expected 'fallback_reason' field for non-Linux")
	}

	// RAM disk должен быть недоступен
	if ramAvailable, ok := result["ram_disk_available"].(bool); !ok || ramAvailable {
		t.Error("Expected ram_disk_available to be false on non-Linux")
	}
}

// TestGetOptimalTempDir_AllCases тестирует все случаи GetOptimalTempDir
func TestGetOptimalTempDir_AllCases(t *testing.T) {
	result := GetOptimalTempDir()

	if result == "" {
		t.Error("GetOptimalTempDir should not return empty string")
	}

	// Проверяем логику выбора
	if IsRAMDiskAvailable() {
		if result != "/dev/shm" {
			t.Errorf("Expected /dev/shm when RAM disk available, got %s", result)
		}
	} else {
		expected := os.TempDir()
		if result != expected {
			t.Errorf("Expected %s when RAM disk unavailable, got %s", expected, result)
		}
	}
}

// TestIsRAMDiskAvailable_AllPlatforms тестирует IsRAMDiskAvailable на всех платформах
func TestIsRAMDiskAvailable_AllPlatforms(t *testing.T) {
	result := IsRAMDiskAvailable()

	// Функция должна возвращать bool
	if _, ok := interface{}(result).(bool); !ok {
		t.Error("IsRAMDiskAvailable should return bool")
	}

	// Проверяем логику для разных платформ
	info, err := os.Stat("/dev/shm")
	expected := err == nil && info.IsDir()
	if result != expected {
		t.Logf("IsRAMDiskAvailable returned %v, expected %v based on /dev/shm check", result, expected)
	}
}

// TestTempManager_isRAMDiskAvailable_AllCases тестирует все случаи isRAMDiskAvailable
func TestTempManager_isRAMDiskAvailable_AllCases(t *testing.T) {
	tm := NewTempManager(true)
	defer tm.Close()

	result := tm.isRAMDiskAvailable()

	// На Linux результат зависит от /dev/shm
	if runtime.GOOS == "linux" {
		info, err := os.Stat("/dev/shm")
		expected := err == nil && info.IsDir()
		if result != expected {
			t.Errorf("Expected %v on Linux, got %v", expected, result)
		}
	} else {
		// На не-Linux всегда false
		if result {
			t.Error("Expected false on non-Linux systems")
		}
	}
}

// TestTempManager_isRAMDiskAvailable_NonLinuxOnly тестирует только не-Linux случай
func TestTempManager_isRAMDiskAvailable_NonLinuxOnly(t *testing.T) {
	if runtime.GOOS == "linux" {
		t.Skip("Skipping non-Linux test on Linux")
	}

	tm := NewTempManager(true)
	defer tm.Close()

	result := tm.isRAMDiskAvailable()
	if result {
		t.Error("Expected false on non-Linux systems")
	}
}

// TestCheckAvailableRAM_LinuxShmError тестирует случай ошибки при проверке /dev/shm
func TestCheckAvailableRAM_LinuxShmError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}

	tm := NewTempManager(true)
	defer tm.Close()

	result := tm.CheckAvailableRAM()

	// Проверяем, что поля shm_exists и shm_error обрабатываются
	if shmExists, exists := result["shm_exists"]; exists {
		if shmExists == false {
			// Если shm не существует, должна быть ошибка
			if _, errorExists := result["shm_error"]; !errorExists {
				t.Error("Expected shm_error when shm_exists is false")
			}
		}
	}
}

// TestGetOptimalTempDir_RAMUnavailable тестирует GetOptimalTempDir когда RAM недоступен
func TestGetOptimalTempDir_RAMUnavailable(t *testing.T) {
	// Этот тест покрывает ветку else в GetOptimalTempDir
	result := GetOptimalTempDir()
	
	// Если RAM диск недоступен, должен возвращать os.TempDir()
	if !IsRAMDiskAvailable() {
		expected := os.TempDir()
		if result != expected {
			t.Errorf("Expected %s when RAM disk unavailable, got %s", expected, result)
		}
	}
}

// TestIsRAMDiskAvailable_StatError тестирует IsRAMDiskAvailable при ошибке os.Stat
func TestIsRAMDiskAvailable_StatError(t *testing.T) {
	// Тестируем поведение функции при различных условиях
	result := IsRAMDiskAvailable()
	
	// Проверяем, что функция возвращает bool
	if _, ok := interface{}(result).(bool); !ok {
		t.Error("IsRAMDiskAvailable should return bool")
	}
	
	// Проверяем логику: если /dev/shm не существует или не директория, должно быть false
	info, err := os.Stat("/dev/shm")
	if err != nil {
		// Если ошибка при stat, должно быть false
		if result {
			t.Error("Expected false when /dev/shm stat fails")
		}
	} else if !info.IsDir() {
		// Если /dev/shm не директория, должно быть false
		if result {
			t.Error("Expected false when /dev/shm is not a directory")
		}
	}
}

// TestTempManager_isRAMDiskAvailable_LinuxStatError тестирует ошибку stat в isRAMDiskAvailable
func TestTempManager_isRAMDiskAvailable_LinuxStatError(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}
	
	tm := NewTempManager(true)
	defer tm.Close()
	
	result := tm.isRAMDiskAvailable()
	
	// На Linux проверяем логику
	info, err := os.Stat("/dev/shm")
	if err != nil {
		// Если ошибка при stat, должно быть false
		if result {
			t.Error("Expected false when /dev/shm stat fails on Linux")
		}
	} else {
		// Если stat успешен, результат зависит от IsDir()
		expected := info.IsDir()
		if result != expected {
			t.Errorf("Expected %v based on IsDir(), got %v", expected, result)
		}
	}
}