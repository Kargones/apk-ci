// Package sonarqube provides implementation of SonarQube entity.
// This package contains the low-level implementation for interacting with SonarQube API,
// including HTTP client configuration, authentication, and basic API methods.
package sonarqube

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	"github.com/Kargones/apk-ci/internal/config"
)

// Entity represents the low-level interaction with SonarQube API.
// This struct contains the HTTP client configuration and implements basic
// methods for interacting with SonarQube REST API.
type Entity struct {
	// client is the HTTP client used for making requests to SonarQube API.
	client *http.Client

	// config contains the SonarQube configuration settings.
	config *config.SonarQubeConfig

	// logger is the structured logger for this entity.
	logger *slog.Logger
}

// NewEntity creates a new instance of Entity.
// This function initializes the HTTP client with appropriate timeouts and
// configures the entity with the provided SonarQube configuration.
//
// Parameters:
//   - cfg: SonarQube configuration settings
//   - logger: structured logger instance
//
// Returns:
//   - *Entity: initialized SonarQube entity
func NewEntity(cfg *config.SonarQubeConfig, logger *slog.Logger) *Entity {
	// Create HTTP client with timeout from config
	client := &http.Client{
		Timeout: cfg.Timeout,
	}

	return &Entity{
		client: client,
		config: cfg,
		logger: logger,
	}
}

// Authenticate authenticates with SonarQube using the provided token.
// This method validates the token by making a simple API request.
//
// Parameters:
//   - token: authentication token
//
// Returns:
//   - error: error if authentication fails
func (s *Entity) Authenticate(token string) error {
	// Temporarily set the token for validation
	originalToken := s.config.Token
	s.config.Token = token
	defer func() {
		s.config.Token = originalToken
	}()

	// Validate the token
	return s.ValidateToken(context.Background())
}

// authenticate adds authentication header to the request.
// This method adds the Authorization header with the Bearer token
// from the configuration to the provided HTTP request.
//
// Parameters:
//   - req: HTTP request to authenticate
//
// Returns:
//   - error: error if authentication fails
func (s *Entity) authenticate(req *http.Request) error {
	if s.config.Token == "" {
		return &ValidationError{
			Field:   "token",
			Message: "SonarQube token is not configured",
		}
	}

	// Add Authorization header with Bearer token
	req.Header.Set("Authorization", "Bearer "+s.config.Token)
	return nil
}

// ValidateToken validates the configured authentication token.
// This method checks if the configured token is valid by making a simple
// API request to SonarQube.
//
// Returns:
//   - error: error if token is invalid or validation fails
func (s *Entity) ValidateToken(ctx context.Context) error {

	// Make a simple request to validate the token
	_, err := s.makeRequest(ctx, "GET", "/authentication/validate", nil)
	if err != nil {
		return fmt.Errorf("token validation failed: %w", err)
	}

	return nil
}

// executeRequest выполняет HTTP-запрос без механизма повторных попыток
func (s *Entity) executeRequest(_ context.Context, req *http.Request) ([]byte, error) {
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer func() {
		if errBody := resp.Body.Close(); errBody != nil {
			s.logger.Error("Failed to close response body", "error", errBody)
		}
	}()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return respBody, nil
	}

	// Handle API errors
	if resp.StatusCode >= 400 {
		// Try to parse error response
		var apiErr struct {
			Errors []struct {
				Msg string `json:"msg"`
			} `json:"errors"`
		}

		if parseErr := json.Unmarshal(respBody, &apiErr); parseErr == nil && len(apiErr.Errors) > 0 {
			return nil, &Error{
				Code:    resp.StatusCode,
				Message: "SonarQube API error",
				Details: apiErr.Errors[0].Msg,
			}
		}

		// If we can't parse the error response, return a generic error
		return nil, &Error{
			Code:    resp.StatusCode,
			Message: fmt.Sprintf("SonarQube API error: %s", resp.Status),
		}
	}

	return respBody, nil
}

// makeRequest performs an HTTP request to SonarQube API.
// This method handles the common logic for making HTTP requests to SonarQube API,
// including authentication, request execution, and error handling.
//
// Parameters:
//   - ctx: context for the request
//   - method: HTTP method (GET, POST, PUT, DELETE)
//   - endpoint: API endpoint path
//   - body: request body (can be nil)
//
// Returns:
//   - []byte: response body
//   - error: error if request fails
func (s *Entity) makeRequest(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	// Construct full URL
	url := s.config.URL + "/api" + endpoint

	// Serialize request body if provided
	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize request body: %w", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set content type for requests with body
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	// Authenticate request
	if authErr := s.authenticate(req); authErr != nil {
		return nil, fmt.Errorf("authentication failed: %w", authErr)
	}

	// Execute request with retry mechanism
	return s.executeRequest(ctx, req)
}

// makeFormRequest performs an HTTP request to SonarQube API with form data.
// This method handles the common logic for making HTTP requests to SonarQube API
// with form-encoded data, including authentication, request execution, and error handling.
//
// Parameters:
//   - ctx: context for the request
//   - method: HTTP method (GET, POST, PUT, DELETE)
//   - endpoint: API endpoint path
//   - formData: form data values
//
// Returns:
//   - []byte: response body
//   - error: error if request fails
func (s *Entity) makeFormRequest(ctx context.Context, method, endpoint string, formData url.Values) ([]byte, error) {
	// Construct full URL
	url := s.config.URL + "/api" + endpoint

	// Prepare form data body
	var bodyReader io.Reader
	if formData != nil {
		bodyReader = strings.NewReader(formData.Encode())
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Set content type for form data
	if formData != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	// Authenticate request
	if authErr := s.authenticate(req); authErr != nil {
		return nil, fmt.Errorf("authentication failed: %w", authErr)
	}

	// Execute request with retry mechanism
	return s.executeRequest(ctx, req)
}

// ToDo: необходимо дополнительно реализовать следующий функционал:
// - Add unit tests for all methods
// - Implement additional API methods as needed
// - Add more detailed error handling and logging
//
// Ссылки на пункты плана и требований:
// - tasks.md: 2.1, 2.2
// - requirements.md: 1.1, 3.1, 3.2, 9.1, 9.2
