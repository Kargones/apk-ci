package gitea

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

func (g *API) sendReq(ctx context.Context, urlString, reqBody, respType string) (int, string, error) {
	var req *http.Request
	var err error
	if reqBody == "" {
		req, err = http.NewRequestWithContext(ctx, respType, urlString, nil)
	} else {
		req, err = http.NewRequestWithContext(ctx, respType, urlString, bytes.NewBuffer([]byte(reqBody)))
	}
	if err != nil {
		return -1, "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return -1, "", err
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			slog.Warn("Failed to close response body", "error", closeErr)
		}
	}()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Warn("Failed to read response body", "error", err)
		return -1, "", err
	}
	return resp.StatusCode, string(bodyBytes), nil
}
