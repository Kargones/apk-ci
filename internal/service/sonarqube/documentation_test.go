package sonarqube

import (
	"context"
	"log/slog"
	"os"
	"testing"
)

func TestNewDocumentationService(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	service := NewDocumentationService(logger)

	if service == nil {
		t.Error("Expected service to be created, got nil")
	}
	if service.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
}

func TestGenerateAPIDocumentation(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewDocumentationService(logger)
	ctx := context.Background()

	doc, err := service.GenerateAPIDocumentation(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if doc == nil {
		t.Error("Expected documentation to be generated, got nil")
	}

	// Check documentation fields
	expectedTitle := "SonarQube Integration API Documentation"
	if doc.Title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, doc.Title)
	}

	expectedDescription := "API documentation for SonarQube integration components"
	if doc.Description != expectedDescription {
		t.Errorf("Expected description '%s', got '%s'", expectedDescription, doc.Description)
	}

	expectedVersion := "1.0.0"
	if doc.Version != expectedVersion {
		t.Errorf("Expected version '%s', got '%s'", expectedVersion, doc.Version)
	}

	if doc.Endpoints == nil {
		t.Error("Expected endpoints to be initialized, got nil")
	}
}

func TestGenerateUserGuide(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewDocumentationService(logger)
	ctx := context.Background()

	guide, err := service.GenerateUserGuide(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if guide == nil {
		t.Error("Expected user guide to be generated, got nil")
	}

	// Check user guide fields
	expectedTitle := "SonarQube Integration User Guide"
	if guide.Title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, guide.Title)
	}

	expectedDescription := "User guide for SonarQube integration commands"
	if guide.Description != expectedDescription {
		t.Errorf("Expected description '%s', got '%s'", expectedDescription, guide.Description)
	}

	expectedVersion := "1.0.0"
	if guide.Version != expectedVersion {
		t.Errorf("Expected version '%s', got '%s'", expectedVersion, guide.Version)
	}

	if guide.Sections == nil {
		t.Error("Expected sections to be initialized, got nil")
	}
}

func TestGenerateTroubleshootingGuide(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	service := NewDocumentationService(logger)
	ctx := context.Background()

	guide, err := service.GenerateTroubleshootingGuide(ctx)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if guide == nil {
		t.Error("Expected troubleshooting guide to be generated, got nil")
	}

	// Check troubleshooting guide fields
	expectedTitle := "SonarQube Integration Troubleshooting Guide"
	if guide.Title != expectedTitle {
		t.Errorf("Expected title '%s', got '%s'", expectedTitle, guide.Title)
	}

	expectedDescription := "Troubleshooting guide for SonarQube integration components"
	if guide.Description != expectedDescription {
		t.Errorf("Expected description '%s', got '%s'", expectedDescription, guide.Description)
	}

	expectedVersion := "1.0.0"
	if guide.Version != expectedVersion {
		t.Errorf("Expected version '%s', got '%s'", expectedVersion, guide.Version)
	}

	if guide.Issues == nil {
		t.Error("Expected issues to be initialized, got nil")
	}
}

func TestAPIDocumentationStructure(t *testing.T) {
	doc := &APIDocumentation{
		Title:       "Test API Documentation",
		Description: "Test description",
		Version:     "1.0.0",
		Endpoints: []Endpoint{
			{
				Path:        "/api/test",
				Method:      "GET",
				Description: "Test endpoint",
				Parameters: []Parameter{
					{
						Name:        "id",
						Type:        "string",
						Required:    true,
						Description: "Test parameter",
					},
				},
				Responses: []Response{
					{
						StatusCode:  200,
						Description: "Success response",
						Schema:      "TestSchema",
					},
				},
			},
		},
	}

	if doc.Title != "Test API Documentation" {
		t.Errorf("Expected title 'Test API Documentation', got '%s'", doc.Title)
	}
	if len(doc.Endpoints) != 1 {
		t.Errorf("Expected 1 endpoint, got %d", len(doc.Endpoints))
	}
	if doc.Endpoints[0].Path != "/api/test" {
		t.Errorf("Expected path '/api/test', got '%s'", doc.Endpoints[0].Path)
	}
	if len(doc.Endpoints[0].Parameters) != 1 {
		t.Errorf("Expected 1 parameter, got %d", len(doc.Endpoints[0].Parameters))
	}
	if doc.Endpoints[0].Parameters[0].Required != true {
		t.Error("Expected parameter to be required")
	}
}

func TestUserGuideStructure(t *testing.T) {
	guide := &UserGuide{
		Title:       "Test User Guide",
		Description: "Test description",
		Version:     "1.0.0",
		Sections: []Section{
			{
				Title:   "Test Section",
				Content: "Test content",
				Examples: []Example{
					{
						Description: "Test example",
						Command:     "test command",
						Output:      "test output",
					},
				},
			},
		},
	}

	if guide.Title != "Test User Guide" {
		t.Errorf("Expected title 'Test User Guide', got '%s'", guide.Title)
	}
	if len(guide.Sections) != 1 {
		t.Errorf("Expected 1 section, got %d", len(guide.Sections))
	}
	if guide.Sections[0].Title != "Test Section" {
		t.Errorf("Expected section title 'Test Section', got '%s'", guide.Sections[0].Title)
	}
	if len(guide.Sections[0].Examples) != 1 {
		t.Errorf("Expected 1 example, got %d", len(guide.Sections[0].Examples))
	}
	if guide.Sections[0].Examples[0].Command != "test command" {
		t.Errorf("Expected command 'test command', got '%s'", guide.Sections[0].Examples[0].Command)
	}
}

func TestTroubleshootingGuideStructure(t *testing.T) {
	guide := &TroubleshootingGuide{
		Title:       "Test Troubleshooting Guide",
		Description: "Test description",
		Version:     "1.0.0",
		Issues: []Issue{
			{
				Title:       "Test Issue",
				Description: "Test issue description",
				Solution:    "Test solution",
				Severity:    "medium",
			},
		},
	}

	if guide.Title != "Test Troubleshooting Guide" {
		t.Errorf("Expected title 'Test Troubleshooting Guide', got '%s'", guide.Title)
	}
	if len(guide.Issues) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(guide.Issues))
	}
	if guide.Issues[0].Title != "Test Issue" {
		t.Errorf("Expected issue title 'Test Issue', got '%s'", guide.Issues[0].Title)
	}
	if guide.Issues[0].Severity != "medium" {
		t.Errorf("Expected severity 'medium', got '%s'", guide.Issues[0].Severity)
	}
}