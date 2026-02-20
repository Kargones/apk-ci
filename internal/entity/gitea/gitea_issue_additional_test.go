package gitea

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestGetIssueAdditionalCases tests additional edge cases for GetIssue function
func TestGetIssueAdditionalCases(t *testing.T) {
	ctx := context.Background()
	// Test with empty response
	t.Run("empty_response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(""))
		}))
		defer server.Close()

		api := &API{
			GiteaURL:    server.URL,
			Owner:       "testowner",
			Repo:        "testrepo",
			AccessToken: "testtoken",
		}

		issue, err := api.GetIssue(ctx, 1)
		if err == nil {
			t.Error("Expected error for empty response, got none")
		}
		if issue != nil {
			t.Error("Expected nil issue for empty response")
		}
	})

	// Test with malformed JSON
	t.Run("malformed_json", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("{invalid json"))
		}))
		defer server.Close()

		api := &API{
			GiteaURL:    server.URL,
			Owner:       "testowner",
			Repo:        "testrepo",
			AccessToken: "testtoken",
		}

		issue, err := api.GetIssue(ctx, 1)
		if err == nil {
			t.Error("Expected error for malformed JSON, got none")
		}
		if issue != nil {
			t.Error("Expected nil issue for malformed JSON")
		}
	})

	// Test with valid response
	t.Run("valid_response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			issue := Issue{
				ID:     1,
				Number: 123,
				Title:  "Test Issue",
				Body:   "Test body",
				State:  "open",
			}
			json.NewEncoder(w).Encode(issue)
		}))
		defer server.Close()

		api := &API{
			GiteaURL:    server.URL,
			Owner:       "testowner",
			Repo:        "testrepo",
			AccessToken: "testtoken",
		}

		issue, err := api.GetIssue(ctx, 123)
		if err != nil {
			t.Errorf("Expected no error, got: %v", err)
		}
		if issue == nil {
			t.Error("Expected issue, got nil")
		}
		if issue != nil && issue.ID != 1 {
			t.Errorf("Expected issue ID 1, got %d", issue.ID)
		}
	})

	// Test with server error
	t.Run("server_error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		api := &API{
			GiteaURL:    server.URL,
			Owner:       "testowner",
			Repo:        "testrepo",
			AccessToken: "testtoken",
		}

		issue, err := api.GetIssue(ctx, 1)
		if err == nil {
			t.Error("Expected error for server error, got none")
		}
		if issue != nil {
			t.Error("Expected nil issue for server error")
		}
	})
}