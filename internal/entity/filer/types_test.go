package filer

import "testing"

func TestFSType_String(t *testing.T) {
	tests := []struct {
		fsType   FSType
		expected string
	}{
		{DiskFS, "DiskFS"},
		{MemoryFS, "MemoryFS"},
	}
	
	for _, test := range tests {
		t.Run(test.expected, func(t *testing.T) {
			result := test.fsType.String()
			if result != test.expected {
				t.Errorf("Expected %s, got %s", test.expected, result)
			}
		})
	}
	
	// Тест для неизвестного типа
	unknownType := FSType(999)
	result := unknownType.String()
	if result != "Unknown" {
		t.Errorf("Expected 'Unknown' for invalid type, got %s", result)
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config.Type != DiskFS {
		t.Errorf("Expected default type to be DiskFS, got %v", config.Type)
	}
	
	if config.BasePath != "" {
		t.Errorf("Expected default BasePath to be empty, got %s", config.BasePath)
	}
	
	if config.UseRAM != false {
		t.Errorf("Expected default UseRAM to be false, got %v", config.UseRAM)
	}
}

func TestConfig_Fields(t *testing.T) {
	config := Config{
		Type:     MemoryFS,
		BasePath: "/tmp/test",
		UseRAM:   true,
	}
	
	if config.Type != MemoryFS {
		t.Errorf("Expected Type to be MemoryFS, got %v", config.Type)
	}
	
	if config.BasePath != "/tmp/test" {
		t.Errorf("Expected BasePath to be '/tmp/test', got %s", config.BasePath)
	}
	
	if config.UseRAM != true {
		t.Errorf("Expected UseRAM to be true, got %v", config.UseRAM)
	}
}