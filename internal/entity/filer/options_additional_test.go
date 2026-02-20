package filer

import (
	"testing"
)

// TestOptions_WithType тестирует WithType
func TestOptions_WithType(t *testing.T) {
	config := DefaultConfig()
	
	// Применяем WithType
	WithType(MemoryFS)(&config)
	
	if config.Type != MemoryFS {
		t.Errorf("Expected Type MemoryFS, got %v", config.Type)
	}
}

// TestOptions_WithBasePath тестирует WithBasePath
func TestOptions_WithBasePath(t *testing.T) {
	config := DefaultConfig()
	
	// Применяем WithBasePath
	WithBasePath("/tmp/test")(&config)
	
	if config.BasePath != "/tmp/test" {
		t.Errorf("Expected BasePath '/tmp/test', got %q", config.BasePath)
	}
}

// TestOptions_ApplyOptions тестирует ApplyOptions
func TestOptions_ApplyOptions(t *testing.T) {
	baseConfig := Config{
		Type:     DiskFS,
		BasePath: "/tmp",
		UseRAM:   false,
	}
	
	// Применяем опции
	newConfig := ApplyOptions(baseConfig, WithMemoryFS())
	
	if newConfig.Type != MemoryFS {
		t.Error("ApplyOptions failed to apply WithMemoryFS")
	}
	
	// Применяем несколько опций
	newConfig2 := ApplyOptions(baseConfig, WithDiskFS("/test"), WithRAMDisk())
	
	if newConfig2.Type != DiskFS {
		t.Error("ApplyOptions failed to apply WithDiskFS")
	}
	
	if newConfig2.BasePath != "/test" { //nolint:goconst // test value
		t.Error("ApplyOptions failed to set base path")
	}
	
	if !newConfig2.UseRAM {
		t.Error("ApplyOptions failed to set UseRAM")
	}
}

// TestOptions_NewConfig тестирует NewConfig
func TestOptions_NewConfig(t *testing.T) {
	// Создаем конфигурацию с опциями
	config := NewConfig(WithMemoryFS(), WithRAMDisk())
	
	if config.Type != MemoryFS {
		t.Error("NewConfig failed to apply WithMemoryFS")
	}
	
	if !config.UseRAM {
		t.Error("NewConfig failed to apply WithRAMDisk")
	}
	
	// Создаем конфигурацию с другими опциями
	config2 := NewConfig(WithDiskFS("/tmp/test"))
	
	if config2.Type != DiskFS {
		t.Error("NewConfig failed to apply WithDiskFS")
	}
	
	if config2.BasePath != "/tmp/test" {
		t.Error("NewConfig failed to set base path")
	}
}