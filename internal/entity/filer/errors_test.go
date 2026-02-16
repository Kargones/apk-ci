package filer

import (
	"errors"
	"os"
	"testing"
)

func TestFileSystemError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *FileSystemError
		expected string
	}{
		{
			name: "Basic error",
			err: &FileSystemError{
				Op:   "read",
				Path: "/test/file.txt",
				Err:  errors.New("permission denied"),
			},
			expected: "операция: read, путь: /test/file.txt, ошибка: permission denied",
		},
		{
			name: "Error without path",
			err: &FileSystemError{
				Op:  "create",
				Err: errors.New("disk full"),
			},
			expected: "операция: create, ошибка: disk full",
		},
		{
			name: "Error without operation",
			err: &FileSystemError{
				Path: "/test/file.txt",
				Err:  errors.New("file not found"),
			},
			expected: "путь: /test/file.txt, ошибка: file not found",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			result := tt.err.Error()
			if result != tt.expected {
				t.Errorf("Error() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFileSystemError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	fsErr := &FileSystemError{
		Op:   "test",
		Path: "/test",
		Err:  originalErr,
	}
	
	unwrapped := fsErr.Unwrap()
	if unwrapped != originalErr {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, originalErr)
	}
}

func TestNewFileSystemError(t *testing.T) {
	originalErr := errors.New("test error")
	fsErr := NewFileSystemError("read", "/test/file.txt", originalErr, DiskFS, SeverityError)
	
	if fsErr.Op != "read" {
		t.Errorf("Op = %q, want %q", fsErr.Op, "read")
	}
	if fsErr.Path != "/test/file.txt" {
		t.Errorf("Path = %q, want %q", fsErr.Path, "/test/file.txt")
	}
	if fsErr.Err != originalErr {
		t.Errorf("Err = %v, want %v", fsErr.Err, originalErr)
	}
	if fsErr.FSType != DiskFS {
		t.Errorf("FSType = %v, want %v", fsErr.FSType, DiskFS)
	}
	if fsErr.Severity != SeverityError {
		t.Errorf("Severity = %v, want %v", fsErr.Severity, SeverityError)
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{
			name:    "Valid relative path",
			path:    "test/file.txt",
			wantErr: false,
		},
		{
			name:    "Valid single file",
			path:    "file.txt",
			wantErr: false,
		},
		{
			name:    "Empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "Path with parent directory traversal",
			path:    "../file.txt",
			wantErr: true,
		},
		{
			name:    "Path with parent directory in middle",
			path:    "test/../file.txt",
			wantErr: true,
		},
		{
			name:    "Absolute path",
			path:    "/tmp/file.txt",
			wantErr: true,
		},
		{
			name:    "Path with null byte",
			path:    "file\x00.txt",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "Valid disk config",
			config: &Config{
				Type:     DiskFS,
				BasePath: "test",
			},
			wantErr: false,
		},
		{
			name: "Valid memory config",
			config: &Config{
				Type: MemoryFS,
			},
			wantErr: false,
		},
		{
			name:    "Nil config",
			config:  nil,
			wantErr: true,
		},
		{
			name: "Invalid type",
			config: &Config{
				Type: FSType(999), // Недопустимое значение
			},
			wantErr: true,
		},
		{
			name: "Disk config without BasePath",
			config: &Config{
				Type: DiskFS,
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.config != nil {
				err = ValidateConfig(*tt.config)
			} else {
				err = errors.New("config is nil")
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSafeOperation(t *testing.T) {
	// Тест успешной операции
	err := SafeOperation("test_success", func() error {
		return nil
	})
	
	if err != nil {
		t.Errorf("SafeOperation should not return error for successful operation: %v", err)
	}
	
	// Тест операции с ошибкой
	err = SafeOperation("test_error", func() error {
		return errors.New("operation failed")
	})
	
	if err == nil {
		t.Error("SafeOperation should return error for failed operation")
	}
	
	// Тест операции с паникой
	err = SafeOperation("test_panic", func() error {
		panic("test panic")
	})
	
	if err == nil {
		t.Error("SafeOperation should return error for panicking operation")
	}
}

func TestDefaultErrorHandler_HandleError(t *testing.T) {
	handler := &DefaultErrorHandler{}
	
	tests := []struct {
		name string
		err  error
	}{
		{
			name: "Simple error",
			err:  errors.New("test error"),
		},
		{
			name: "FileSystemError",
			err: &FileSystemError{
				Op:   "read",
				Path: "/test/file.txt",
				Err:  errors.New("permission denied"),
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(_ *testing.T) {
			// HandleError не должен паниковать
			handler.HandleError(tt.err)
		})
	}
}

func TestDefaultErrorHandler_ShouldRetry(t *testing.T) {
	handler := &DefaultErrorHandler{}
	
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "Temporary error",
			err:      &temporaryError{msg: "temporary failure"},
			expected: true,
		},
		{
			name:     "Non-temporary error",
			err:      errors.New("permanent failure"),
			expected: false,
		},
		{
			name:     "Nil error",
			err:      nil,
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.ShouldRetry(tt.err)
			if result != tt.expected {
				t.Errorf("ShouldRetry() = %v, want %v", result, tt.expected)
			}
		})
	}
}

// temporaryError реализует интерфейс для временных ошибок
type temporaryError struct {
	msg string
}

func (e *temporaryError) Error() string {
	return e.msg
}

func (e *temporaryError) Temporary() bool {
	return true
}

func TestErrorSeverity_String(t *testing.T) {
	tests := []struct {
		name     string
		severity ErrorSeverity
		expected string
	}{
		{"Info", SeverityInfo, "INFO"},
		{"Warning", SeverityWarning, "WARNING"},
		{"Error", SeverityError, "ERROR"},
		{"Critical", SeverityCritical, "CRITICAL"},
		{"Unknown", ErrorSeverity(999), "UNKNOWN"},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.severity.String()
			if result != tt.expected {
				t.Errorf("String() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestDetermineSeverity(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected ErrorSeverity
	}{
		{"PathTraversal", ErrPathTraversal, SeverityError},
		{"InsufficientPermissions", ErrInsufficientPermissions, SeverityError},
		{"InvalidPath", ErrInvalidPath, SeverityError},
		{"FileSystemClosed", ErrFileSystemClosed, SeverityCritical},
		{"ResourceExhausted", ErrResourceExhausted, SeverityCritical},
		{"FileNotFound", os.ErrNotExist, SeverityWarning},
		{"PermissionDenied", os.ErrPermission, SeverityError},
		{"GenericError", errors.New("generic"), SeverityError},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := determineSeverity(tt.err)
			if result != tt.expected {
				t.Errorf("determineSeverity() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsSecurityError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"PathTraversal", ErrPathTraversal, true},
		{"InsufficientPermissions", ErrInsufficientPermissions, true},
		{"InvalidPath", ErrInvalidPath, true},
		{"GenericError", errors.New("generic"), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsSecurityError(tt.err)
			if result != tt.expected {
				t.Errorf("IsSecurityError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsCriticalError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"FileSystemClosed", ErrFileSystemClosed, true},
		{"ResourceExhausted", ErrResourceExhausted, true},
		{"CriticalFileSystemError", &FileSystemError{Severity: SeverityCritical}, true},
		{"ErrorFileSystemError", &FileSystemError{Severity: SeverityError}, false},
		{"InvalidPath", ErrInvalidPath, false},
		{"GenericError", errors.New("generic"), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsCriticalError(tt.err)
			if result != tt.expected {
				t.Errorf("IsCriticalError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"TemporaryError", &temporaryError{"temp"}, true},
		{"ResourceExhausted", ErrResourceExhausted, false}, // критическая ошибка
		{"PathTraversal", ErrPathTraversal, false}, // ошибка безопасности
		{"InvalidPath", ErrInvalidPath, false}, // ошибка безопасности
		{"GenericError", errors.New("generic"), false},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsRetryableError(tt.err)
			if result != tt.expected {
				t.Errorf("IsRetryableError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWrapError(t *testing.T) {
	originalErr := errors.New("original error")
	wrappedErr := WrapError("test_op", "/test/path", originalErr, DiskFS)
	
	if wrappedErr == nil {
		t.Fatal("WrapError should return non-nil error")
	}
	
	fsErr, ok := wrappedErr.(*FileSystemError)
	if !ok {
		t.Fatal("WrapError should return *FileSystemError")
	}
	
	if fsErr.Op != "test_op" {
		t.Errorf("Expected Op = 'test_op', got %q", fsErr.Op)
	}
	
	if fsErr.Path != "/test/path" {
		t.Errorf("Expected Path = '/test/path', got %q", fsErr.Path)
	}
	
	if fsErr.FSType != DiskFS {
		t.Errorf("Expected FSType = DiskFS, got %v", fsErr.FSType)
	}
	
	if !errors.Is(fsErr, originalErr) {
		t.Error("Wrapped error should contain original error")
	}
}