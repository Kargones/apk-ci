package filer

import (
	"testing"
)

// TestConstants тестирует константы модуля
func TestConstants(t *testing.T) {
	// Тест FSType констант
	tests := []struct {
		name     string
		fsType   FSType
		expected string
	}{
		{
			name:     "DiskFS",
			fsType:   DiskFS,
			expected: "DiskFS",
		},
		{
			name:     "MemoryFS",
			fsType:   MemoryFS,
			expected: "MemoryFS",
		},
		{
			name:     "Unknown FSType",
			fsType:   FSType(999),
			expected: "Unknown",
		},
	}
	
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			result := test.fsType.String()
			if result != test.expected {
				t.Errorf("FSType.String() = %q, expected %q", result, test.expected)
			}
		})
	}
	
	// Тест констант строк
	if DefaultDir == "" {
		t.Error("DefaultDir should not be empty")
	}
	
	if RAMDiskPath == "" {
		t.Error("RAMDiskPath should not be empty")
	}
	
	if TempDirPrefix == "" {
		t.Error("TempDirPrefix should not be empty")
	}
	
	// Тест числовых констант
	if RAMDiskThreshold <= 0 || RAMDiskThreshold > 1 {
		t.Error("RAMDiskThreshold should be between 0 and 1")
	}
	
	if MinRAMSize <= 0 {
		t.Error("MinRAMSize should be positive")
	}
	
	if MaxFileSize <= 0 {
		t.Error("MaxFileSize should be positive")
	}
	
	// Тест FileMode констант
	if DirMode == 0 {
		t.Error("DirMode should not be zero")
	}
	
	if FileMode == 0 {
		t.Error("FileMode should not be zero")
	}
	
	if TempMode == 0 {
		t.Error("TempMode should not be zero")
	}
	
	// Проверяем, что FileMode константы имеют правильные значения
	if DirMode != 0755 {
		t.Errorf("DirMode = %o, expected 0755", DirMode)
	}
	
	if FileMode != 0644 {
		t.Errorf("FileMode = %o, expected 0644", FileMode)
	}
	
	if TempMode != 0700 {
		t.Errorf("TempMode = %o, expected 0700", TempMode)
	}
}

// TestDefaultConfigConstants тестирует конфигурацию по умолчанию
func TestDefaultConfigConstants(t *testing.T) {
	config := DefaultConfig()
	
	if config.Type != DiskFS {
		t.Errorf("DefaultConfig.Type = %v, expected DiskFS", config.Type)
	}
	
	if config.BasePath != "" {
		t.Errorf("DefaultConfig.BasePath = %q, expected empty string", config.BasePath)
	}
	
	if config.UseRAM != false {
		t.Errorf("DefaultConfig.UseRAM = %v, expected false", config.UseRAM)
	}
}

// TestConfigFields тестирует поля конфигурации
func TestConfigFields(t *testing.T) {
	config := Config{
		Type:     MemoryFS,
		BasePath: "/tmp/test",
		UseRAM:   true,
	}
	
	if config.Type != MemoryFS {
		t.Errorf("Config.Type = %v, expected MemoryFS", config.Type)
	}
	
	if config.BasePath != "/tmp/test" {
		t.Errorf("Config.BasePath = %q, expected '/tmp/test'", config.BasePath)
	}
	
	if config.UseRAM != true {
		t.Errorf("Config.UseRAM = %v, expected true", config.UseRAM)
	}
}