// Package sonarqube provides resource management for SonarScanner operations.
// This package contains resource management logic for handling temporary files,
// processes, network connections, and cleanup operations.
package sonarqube

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ResourceType represents the type of resource being managed.
type ResourceType string

const (
	// ResourceTypeFile represents file resources
	ResourceTypeFile ResourceType = "file"
	// ResourceTypeDirectory represents directory resources
	ResourceTypeDirectory ResourceType = "directory"
	// ResourceTypeProcess represents process resources
	ResourceTypeProcess ResourceType = "process"
	// ResourceTypeNetwork represents network resources
	ResourceTypeNetwork ResourceType = "network"
	// ResourceTypeGeneric represents generic resources
	ResourceTypeGeneric ResourceType = "generic"
)

// Resource represents a managed resource.
type Resource struct {
	// ID is the unique identifier for the resource
	ID string `json:"id"`
	
	// Type is the type of resource
	Type ResourceType `json:"type"`
	
	// Path is the file system path (for file/directory resources)
	Path string `json:"path,omitempty"`
	
	// PID is the process ID (for process resources)
	PID int `json:"pid,omitempty"`
	
	// Description is a human-readable description of the resource
	Description string `json:"description"`
	
	// CreatedAt is the timestamp when the resource was created
	CreatedAt time.Time `json:"created_at"`
	
	// CleanupFunc is the function to call for cleanup
	CleanupFunc func() error `json:"-"`
	
	// IsTemporary indicates if the resource should be cleaned up automatically
	IsTemporary bool `json:"is_temporary"`
	
	// Priority defines the cleanup priority (higher numbers cleaned up first)
	Priority int `json:"priority"`
}

// ResourceManager manages resources for SonarScanner operations.
type ResourceManager struct {
	// mu protects concurrent access to resources
	mu sync.RWMutex
	
	// resources holds all managed resources
	resources map[string]*Resource
	
	// logger for resource management operations
	logger *slog.Logger
	
	// tempDir is the base directory for temporary files
	tempDir string
	
	// isShutdown indicates if the manager is shutting down
	isShutdown bool
	
	// cleanupTimeout is the timeout for cleanup operations
	cleanupTimeout time.Duration
	
	// autoCleanup indicates if automatic cleanup is enabled
	autoCleanup bool
	
	// cleanupInterval is the interval for automatic cleanup
	cleanupInterval time.Duration
	
	// cleanupTicker is the ticker for automatic cleanup
	cleanupTicker *time.Ticker
	
	// cleanupDone is the channel to signal cleanup goroutine to stop
	cleanupDone chan struct{}
}

// NewResourceManager creates a new resource manager.
//
// Parameters:
//   - logger: logger for resource management operations
//   - tempDir: base directory for temporary files
//
// Returns:
//   - *ResourceManager: new resource manager instance
func NewResourceManager(logger *slog.Logger, tempDir string) *ResourceManager {
	return &ResourceManager{
		resources:       make(map[string]*Resource),
		logger:          logger,
		tempDir:         tempDir,
		isShutdown:      false,
		cleanupTimeout:  30 * time.Second,
		autoCleanup:     true,
		cleanupInterval: 5 * time.Minute,
		cleanupDone:     make(chan struct{}),
	}
}

// Start starts the resource manager and begins automatic cleanup if enabled.
func (rm *ResourceManager) Start() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.autoCleanup && rm.cleanupTicker == nil {
		rm.cleanupTicker = time.NewTicker(rm.cleanupInterval)
		go rm.autoCleanupWorker()
		rm.logger.Debug("Resource manager started with automatic cleanup", 
			"interval", rm.cleanupInterval)
	} else {
		rm.logger.Debug("Resource manager started without automatic cleanup")
	}
}

// Stop stops the resource manager and performs final cleanup.
func (rm *ResourceManager) Stop() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.isShutdown {
		return nil
	}
	
	rm.isShutdown = true
	
	// Stop automatic cleanup
	if rm.cleanupTicker != nil {
		rm.cleanupTicker.Stop()
		close(rm.cleanupDone)
		rm.cleanupTicker = nil
	}
	
	rm.logger.Debug("Resource manager stopping, performing final cleanup")
	
	// Perform final cleanup
	return rm.cleanupAllResources()
}

// autoCleanupWorker runs the automatic cleanup process.
func (rm *ResourceManager) autoCleanupWorker() {
	for {
		select {
		case <-rm.cleanupTicker.C:
			rm.cleanupTemporaryResources()
		case <-rm.cleanupDone:
			return
		}
	}
}

// RegisterFile registers a file resource for management.
//
// Parameters:
//   - id: unique identifier for the resource
//   - path: file path
//   - description: human-readable description
//   - isTemporary: whether the file should be cleaned up automatically
//
// Returns:
//   - error: error if registration fails
func (rm *ResourceManager) RegisterFile(id, path, description string, isTemporary bool) error {
	return rm.RegisterResource(&Resource{
		ID:          id,
		Type:        ResourceTypeFile,
		Path:        path,
		Description: description,
		CreatedAt:   time.Now(),
		IsTemporary: isTemporary,
		Priority:    1,
		CleanupFunc: func() error {
			return rm.removeFile(path)
		},
	})
}

// RegisterDirectory registers a directory resource for management.
//
// Parameters:
//   - id: unique identifier for the resource
//   - path: directory path
//   - description: human-readable description
//   - isTemporary: whether the directory should be cleaned up automatically
//
// Returns:
//   - error: error if registration fails
func (rm *ResourceManager) RegisterDirectory(id, path, description string, isTemporary bool) error {
	return rm.RegisterResource(&Resource{
		ID:          id,
		Type:        ResourceTypeDirectory,
		Path:        path,
		Description: description,
		CreatedAt:   time.Now(),
		IsTemporary: isTemporary,
		Priority:    2,
		CleanupFunc: func() error {
			return rm.removeDirectory(path)
		},
	})
}

// RegisterProcess registers a process resource for management.
//
// Parameters:
//   - id: unique identifier for the resource
//   - pid: process ID
//   - description: human-readable description
//   - killFunc: function to kill the process
//
// Returns:
//   - error: error if registration fails
func (rm *ResourceManager) RegisterProcess(id string, pid int, description string, killFunc func() error) error {
	return rm.RegisterResource(&Resource{
		ID:          id,
		Type:        ResourceTypeProcess,
		PID:         pid,
		Description: description,
		CreatedAt:   time.Now(),
		IsTemporary: true, // Processes are always temporary
		Priority:    10,   // High priority for cleanup
		CleanupFunc: killFunc,
	})
}

// RegisterGeneric registers a generic resource for management.
//
// Parameters:
//   - id: unique identifier for the resource
//   - description: human-readable description
//   - cleanupFunc: function to clean up the resource
//   - isTemporary: whether the resource should be cleaned up automatically
//   - priority: cleanup priority
//
// Returns:
//   - error: error if registration fails
func (rm *ResourceManager) RegisterGeneric(id, description string, cleanupFunc func() error, isTemporary bool, priority int) error {
	return rm.RegisterResource(&Resource{
		ID:          id,
		Type:        ResourceTypeGeneric,
		Description: description,
		CreatedAt:   time.Now(),
		IsTemporary: isTemporary,
		Priority:    priority,
		CleanupFunc: cleanupFunc,
	})
}

// RegisterResource registers a resource for management.
//
// Parameters:
//   - resource: resource to register
//
// Returns:
//   - error: error if registration fails
func (rm *ResourceManager) RegisterResource(resource *Resource) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.isShutdown {
		return fmt.Errorf("resource manager is shutting down")
	}
	
	if resource.ID == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	
	if _, exists := rm.resources[resource.ID]; exists {
		return fmt.Errorf("resource with ID '%s' already exists", resource.ID)
	}
	
	rm.resources[resource.ID] = resource
	rm.logger.Debug("Resource registered", 
		"id", resource.ID,
		"type", resource.Type,
		"description", resource.Description,
		"temporary", resource.IsTemporary)
	
	return nil
}

// UnregisterResource unregisters a resource without cleanup.
//
// Parameters:
//   - id: resource ID to unregister
//
// Returns:
//   - error: error if unregistration fails
func (rm *ResourceManager) UnregisterResource(id string) error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if _, exists := rm.resources[id]; !exists {
		return fmt.Errorf("resource with ID '%s' not found", id)
	}
	
	delete(rm.resources, id)
	rm.logger.Debug("Resource unregistered", "id", id)
	
	return nil
}

// CleanupResource cleans up a specific resource.
//
// Parameters:
//   - id: resource ID to clean up
//
// Returns:
//   - error: error if cleanup fails
func (rm *ResourceManager) CleanupResource(id string) error {
	rm.mu.Lock()
	resource, exists := rm.resources[id]
	if !exists {
		rm.mu.Unlock()
		return fmt.Errorf("resource with ID '%s' not found", id)
	}
	delete(rm.resources, id)
	rm.mu.Unlock()
	
	rm.logger.Debug("Cleaning up resource", 
		"id", resource.ID,
		"type", resource.Type,
		"description", resource.Description)
	
	if resource.CleanupFunc != nil {
		ctx, cancel := context.WithTimeout(context.Background(), rm.cleanupTimeout)
		defer cancel()
		
		done := make(chan error, 1)
		go func() {
			done <- resource.CleanupFunc()
		}()
		
		select {
		case err := <-done:
			if err != nil {
				rm.logger.Warn("Resource cleanup failed", 
					"id", resource.ID,
					"error", err)
				return fmt.Errorf("cleanup failed for resource '%s': %w", id, err)
			}
		case <-ctx.Done():
			rm.logger.Warn("Resource cleanup timed out", 
				"id", resource.ID,
				"timeout", rm.cleanupTimeout)
			return fmt.Errorf("cleanup timed out for resource '%s'", id)
		}
	}
	
	rm.logger.Debug("Resource cleaned up successfully", "id", resource.ID)
	return nil
}

// cleanupTemporaryResources cleans up all temporary resources.
func (rm *ResourceManager) cleanupTemporaryResources() {
	rm.mu.RLock()
	var tempResources []*Resource
	for _, resource := range rm.resources {
		if resource.IsTemporary {
			tempResources = append(tempResources, resource)
		}
	}
	rm.mu.RUnlock()
	
	if len(tempResources) == 0 {
		return
	}
	
	rm.logger.Debug("Cleaning up temporary resources", "count", len(tempResources))
	
	for _, resource := range tempResources {
		if err := rm.CleanupResource(resource.ID); err != nil {
			rm.logger.Warn("Failed to cleanup temporary resource", 
				"id", resource.ID,
				"error", err)
		}
	}
}

// cleanupAllResources cleans up all resources.
func (rm *ResourceManager) cleanupAllResources() error {
	// Get all resources sorted by priority (highest first)
	resources := make([]*Resource, 0, len(rm.resources))
	for _, resource := range rm.resources {
		resources = append(resources, resource)
	}
	
	// Sort by priority (highest first)
	for i := 0; i < len(resources)-1; i++ {
		for j := i + 1; j < len(resources); j++ {
			if resources[i].Priority < resources[j].Priority {
				resources[i], resources[j] = resources[j], resources[i]
			}
		}
	}
	
	var errors []error
	for _, resource := range resources {
		if err := rm.CleanupResource(resource.ID); err != nil {
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("cleanup completed with %d errors: %v", len(errors), errors)
	}
	
	return nil
}

// GetResources returns a copy of all registered resources.
//
// Returns:
//   - map[string]*Resource: copy of all resources
func (rm *ResourceManager) GetResources() map[string]*Resource {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	resources := make(map[string]*Resource)
	for id, resource := range rm.resources {
		// Create a copy without the cleanup function
		resources[id] = &Resource{
			ID:          resource.ID,
			Type:        resource.Type,
			Path:        resource.Path,
			PID:         resource.PID,
			Description: resource.Description,
			CreatedAt:   resource.CreatedAt,
			IsTemporary: resource.IsTemporary,
			Priority:    resource.Priority,
		}
	}
	
	return resources
}

// GetResourceCount returns the number of registered resources.
//
// Returns:
//   - int: number of resources
func (rm *ResourceManager) GetResourceCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	
	return len(rm.resources)
}

// CreateTempFile creates a temporary file and registers it for cleanup.
//
// Parameters:
//   - prefix: file name prefix
//   - suffix: file name suffix
//
// Returns:
//   - string: path to the created file
//   - error: error if creation fails
func (rm *ResourceManager) CreateTempFile(prefix, suffix string) (string, error) {
	tempFile, err := os.CreateTemp(rm.tempDir, prefix+"*"+suffix)
	if err != nil {
		return "", fmt.Errorf("failed to create temporary file: %w", err)
	}
	
	filePath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		rm.logger.Warn("Failed to close temporary file", "path", filePath, "error", err)
	}
	
	// Register the file for cleanup
	fileID := fmt.Sprintf("tempfile_%d", time.Now().UnixNano())
	if err := rm.RegisterFile(fileID, filePath, 
		fmt.Sprintf("Temporary file: %s", filepath.Base(filePath)), true); err != nil {
		// If registration fails, clean up the file manually
		if removeErr := os.Remove(filePath); removeErr != nil {
			rm.logger.Warn("Failed to remove temporary file after registration failure", "path", filePath, "error", removeErr)
		}
		return "", fmt.Errorf("failed to register temporary file: %w", err)
	}
	
	rm.logger.Debug("Temporary file created", "path", filePath, "id", fileID)
	return filePath, nil
}

// CreateTempDir creates a temporary directory and registers it for cleanup.
//
// Parameters:
//   - prefix: directory name prefix
//
// Returns:
//   - string: path to the created directory
//   - error: error if creation fails
func (rm *ResourceManager) CreateTempDir(prefix string) (string, error) {
	tempDir, err := os.MkdirTemp(rm.tempDir, prefix+"*")
	if err != nil {
		return "", fmt.Errorf("failed to create temporary directory: %w", err)
	}
	
	// Register the directory for cleanup
	dirID := fmt.Sprintf("tempdir_%d", time.Now().UnixNano())
	if err := rm.RegisterDirectory(dirID, tempDir, 
		fmt.Sprintf("Temporary directory: %s", filepath.Base(tempDir)), true); err != nil {
		// If registration fails, clean up the directory manually
		if removeErr := os.RemoveAll(tempDir); removeErr != nil {
			rm.logger.Warn("Failed to remove temporary directory after registration failure", "path", tempDir, "error", removeErr)
		}
		return "", fmt.Errorf("failed to register temporary directory: %w", err)
	}
	
	rm.logger.Debug("Temporary directory created", "path", tempDir, "id", dirID)
	return tempDir, nil
}

// removeFile removes a file from the filesystem.
func (rm *ResourceManager) removeFile(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// File doesn't exist, consider it cleaned up
		return nil
	}
	
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file '%s': %w", path, err)
	}
	
	rm.logger.Debug("File removed", "path", path)
	return nil
}

// removeDirectory removes a directory and all its contents from the filesystem.
func (rm *ResourceManager) removeDirectory(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Directory doesn't exist, consider it cleaned up
		return nil
	}
	
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("failed to remove directory '%s': %w", path, err)
	}
	
	rm.logger.Debug("Directory removed", "path", path)
	return nil
}

// SetCleanupTimeout sets the timeout for cleanup operations.
//
// Parameters:
//   - timeout: cleanup timeout duration
func (rm *ResourceManager) SetCleanupTimeout(timeout time.Duration) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.cleanupTimeout = timeout
	rm.logger.Debug("Cleanup timeout updated", "timeout", timeout)
}

// SetAutoCleanup enables or disables automatic cleanup.
//
// Parameters:
//   - enabled: whether to enable automatic cleanup
//   - interval: cleanup interval (only used if enabled is true)
func (rm *ResourceManager) SetAutoCleanup(enabled bool, interval time.Duration) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.autoCleanup = enabled
	if enabled {
		rm.cleanupInterval = interval
	}
	
	rm.logger.Debug("Auto cleanup settings updated", 
		"enabled", enabled,
		"interval", interval)
}