package gitea_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/Kargones/apk-ci/internal/adapter/gitea"
	entity_gitea "github.com/Kargones/apk-ci/internal/entity/gitea"
)

func TestNewAPIClient_ImplementsClient(t *testing.T) {
	// Verify that APIClient implements the Client interface at compile time.
	api := &entity_gitea.API{}
	client := gitea.NewAPIClient(api)
	var _ gitea.Client = client
	assert.NotNil(t, client)
}

func TestNewAPIClientWithLogger(t *testing.T) {
	api := &entity_gitea.API{}
	client := gitea.NewAPIClientWithLogger(api, nil)
	assert.NotNil(t, client)
}
