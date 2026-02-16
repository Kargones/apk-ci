package gitea

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
)

func (g *API) sendReq(urlString, reqBody, respType string) (int, string, error) {
	var client *http.Client
	var req *http.Request
	var err error
	if reqBody == "" {
		req, err = http.NewRequest(respType, urlString, nil)
	} else {
		req, err = http.NewRequest(respType, urlString, bytes.NewBuffer([]byte(reqBody)))
	}
	if err != nil {
		return -1, "", err
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", g.AccessToken))
	req.Header.Set("Content-Type", "application/json")

	client = &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return -1, "", err
	}
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read response body: %v", err)
		return -1, "", err
	}
	bodyString := string(bodyBytes)
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Failed to close response body: %v", closeErr)
		}
	}()

	return resp.StatusCode, bodyString, err
}
