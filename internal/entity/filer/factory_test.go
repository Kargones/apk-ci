package filer

import (
	"testing"
)

func TestFactory_CreateFileSystem(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Cleanup()
	
	tests := []struct {
		name   string
		fsType FSType
		config Config
		wantErr bool
	}{
		{
			name:   "DiskFS with valid config",
			fsType: DiskFS,
			config: Config{
				Type:     DiskFS,
				BasePath: "/tmp",
			},
			wantErr: false,
		},
		{
			name:   "MemoryFS with valid config",
			fsType: MemoryFS,
			config: Config{
				Type:     MemoryFS,
				BasePath: "/tmp",
			},
			wantErr: false,
		},
		{
			name:   "DiskFS with empty path",
			fsType: DiskFS,
			config: Config{
				Type:     DiskFS,
				BasePath: "",
			},
			wantErr: true,
		},
		{
			name:   "Invalid FS type",
			fsType: FSType(999),
			config: Config{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := factory.CreateFileSystem(tt.fsType, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateFileSystem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && fs == nil {
				t.Error("CreateFileSystem() returned nil filesystem")
			}
		})
	}
}

func TestFactory_CreateDiskFileSystem(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Cleanup()
	
	config := Config{
		Type:     DiskFS,
		BasePath: t.TempDir(),
	}
	
	fs, err := factory.CreateDiskFileSystem(config)
	if err != nil {
		t.Errorf("CreateDiskFileSystem() failed: %v", err)
	}
	if fs == nil {
		t.Error("CreateDiskFileSystem() returned nil")
	}
}

func TestFactory_CreateMemoryFileSystem(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Cleanup()
	
	config := Config{
		Type:     MemoryFS,
		BasePath: "/tmp",
	}
	
	fs, err := factory.CreateMemoryFileSystem(config)
	if err != nil {
		t.Errorf("CreateMemoryFileSystem() failed: %v", err)
	}
	if fs == nil {
		t.Error("CreateMemoryFileSystem() returned nil")
	}
}

func TestFactory_CreateTempFileSystem(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Cleanup()
	
	tests := []struct {
		name   string
		fsType FSType
		wantErr bool
	}{
		{
			name:   "DiskFS temp",
			fsType: DiskFS,
			wantErr: false,
		},
		{
			name:   "MemoryFS temp",
			fsType: MemoryFS,
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs, err := factory.CreateTempFileSystem(tt.fsType)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateTempFileSystem() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && fs == nil {
				t.Error("CreateTempFileSystem() returned nil filesystem")
			}
		})
	}
}

func TestFactory_ValidateConfig(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	
	tests := []struct {
		name    string
		fsType  FSType
		config  Config
		wantErr bool
	}{
		{
			name:   "Valid DiskFS config",
			fsType: DiskFS,
			config: Config{
				Type:     DiskFS,
				BasePath: "tmp",
			},
			wantErr: false,
		},
		{
			name:   "Invalid DiskFS config - empty path",
			fsType: DiskFS,
			config: Config{
				Type:     DiskFS,
				BasePath: "",
			},
			wantErr: true,
		},
		{
			name:   "Valid MemoryFS config",
			fsType: MemoryFS,
			config: Config{
				Type:     MemoryFS,
				BasePath: "tmp",
			},
			wantErr: false,
		},
		{
			name:   "Unsupported FS type",
			fsType: FSType(999),
			config: Config{},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.ValidateConfig(tt.fsType, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFactory_GetSupportedTypes(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	
	supportedTypes := factory.GetSupportedTypes()
	if len(supportedTypes) != 2 {
		t.Errorf("Expected 2 supported types, got %d", len(supportedTypes))
	}
	
	// Проверяем, что поддерживаются DiskFS и MemoryFS
	found := make(map[FSType]bool)
	for _, fsType := range supportedTypes {
		found[fsType] = true
	}
	
	if !found[DiskFS] {
		t.Error("DiskFS not found in supported types")
	}
	if !found[MemoryFS] {
		t.Error("MemoryFS not found in supported types")
	}
}

func TestFactory_IsTypeSupported(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	
	tests := []struct {
		name      string
		fsType    FSType
		supported bool
	}{
		{
			name:      "DiskFS supported",
			fsType:    DiskFS,
			supported: true,
		},
		{
			name:      "MemoryFS supported",
			fsType:    MemoryFS,
			supported: true,
		},
		{
			name:      "Unknown type not supported",
			fsType:    FSType(999),
			supported: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := factory.IsTypeSupported(tt.fsType)
			if result != tt.supported {
				t.Errorf("IsTypeSupported() = %v, want %v", result, tt.supported)
			}
		})
	}
}

func TestFactory_CreateTempFileSystem_Extended(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	defer factory.Cleanup()
	
	testCases := []struct {
		name        string
		fsType      FSType
		expectError bool
	}{
		{
			name:        "DiskFS temp with prefix",
			fsType:      DiskFS,
			expectError: false,
		},
		{
			name:        "MemoryFS temp with prefix",
			fsType:      MemoryFS,
			expectError: false,
		},
		{
			name:        "Invalid type",
			fsType:      FSType(999),
			expectError: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			fs, err := factory.CreateTempFileSystem(tc.fsType)
			if tc.expectError {
				if err == nil {
					t.Errorf("Expected error for fsType %v, but got none", tc.fsType)
				}
				return
			}
			
			if err != nil {
				t.Errorf("Unexpected error for fsType %v: %v", tc.fsType, err)
				return
			}
			
			if fs == nil {
				t.Errorf("Expected non-nil filesystem for fsType %v", tc.fsType)
				return
			}
			
			// Test that filesystem works
			testFile := "test.txt"
			testContent := []byte("test content")
			
			err = fs.WriteFile(testFile, testContent, 0644)
			if err != nil {
				t.Errorf("Failed to write test file: %v", err)
			}
			
			readContent, err := fs.ReadFile(testFile)
			if err != nil {
				t.Errorf("Failed to read test file: %v", err)
			}
			
			if string(readContent) != string(testContent) {
				t.Errorf("Content mismatch: got %q, expected %q", string(readContent), string(testContent))
			}
		})
	}
}

func TestFactory_ValidateConfig_Extended(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	
	tests := []struct {
		name    string
		fsType  FSType
		config  Config
		wantErr bool
	}{
		{
			name:   "Invalid DiskFS config with absolute path",
			fsType: DiskFS,
			config: Config{
				Type:     DiskFS,
				BasePath: "/tmp/test",
			},
			wantErr: true,
		},
		{
			name:   "Valid DiskFS config with relative path",
			fsType: DiskFS,
			config: Config{
				Type:     DiskFS,
				BasePath: "./tmp",
			},
			wantErr: false,
		},
		{
			name:   "Invalid DiskFS config - empty path",
			fsType: DiskFS,
			config: Config{
				Type:     DiskFS,
				BasePath: "",
			},
			wantErr: true,
		},
		{
			name:   "Valid MemoryFS config",
			fsType: MemoryFS,
			config: Config{
				Type:     MemoryFS,
				BasePath: "/memory",
			},
			wantErr: false,
		},
		{
			name:   "MemoryFS with empty path (should be valid)",
			fsType: MemoryFS,
			config: Config{
				Type:     MemoryFS,
				BasePath: "",
			},
			wantErr: false,
		},
		{
			name:   "Unsupported FS type",
			fsType: FSType(999),
			config: Config{},
			wantErr: true,
		},
		{
			name:   "Config type mismatch",
			fsType: DiskFS,
			config: Config{
				Type:     MemoryFS,
				BasePath: "/tmp",
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := factory.ValidateConfig(tt.fsType, tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFactory_GetSupportedTypes_Extended(t *testing.T) {
	factory, err := NewFactory()
	if err != nil {
		t.Fatalf("Failed to create factory: %v", err)
	}
	
	supportedTypes := factory.GetSupportedTypes()
	if len(supportedTypes) < 2 {
		t.Errorf("Expected at least 2 supported types, got %d", len(supportedTypes))
	}
	
	// Check that supported types are unique
	typeMap := make(map[FSType]bool)
	for _, fsType := range supportedTypes {
		if typeMap[fsType] {
			t.Errorf("Duplicate type found in supported types: %v", fsType)
		}
		typeMap[fsType] = true
	}
	
	// Check that all returned types are actually supported
	for _, fsType := range supportedTypes {
		if !factory.IsTypeSupported(fsType) {
			t.Errorf("Type %v is in supported list but IsTypeSupported returns false", fsType)
		}
	}
	
	// Check that essential types are supported
	if !typeMap[DiskFS] {
		t.Error("DiskFS not found in supported types")
	}
	if !typeMap[MemoryFS] {
		t.Error("MemoryFS not found in supported types")
	}
}