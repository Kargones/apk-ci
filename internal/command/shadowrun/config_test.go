package shadowrun

import (
	"os"
	"testing"

	"github.com/Kargones/apk-ci/internal/constants"
	"github.com/stretchr/testify/assert"
)

func TestIsEnabled_True(t *testing.T) {
	t.Setenv(constants.EnvShadowRun, "true")
	assert.True(t, IsEnabled())
}

func TestIsEnabled_TrueUpperCase(t *testing.T) {
	t.Setenv(constants.EnvShadowRun, "TRUE")
	assert.True(t, IsEnabled())
}

func TestIsEnabled_TrueMixedCase(t *testing.T) {
	t.Setenv(constants.EnvShadowRun, "True")
	assert.True(t, IsEnabled())
}

func TestIsEnabled_False(t *testing.T) {
	t.Setenv(constants.EnvShadowRun, "false")
	assert.False(t, IsEnabled())
}

func TestIsEnabled_Empty(t *testing.T) {
	t.Setenv(constants.EnvShadowRun, "")
	assert.False(t, IsEnabled())
}

func TestIsEnabled_Unset(t *testing.T) {
	os.Unsetenv(constants.EnvShadowRun)
	assert.False(t, IsEnabled())
}

func TestIsEnabled_InvalidValue(t *testing.T) {
	t.Setenv(constants.EnvShadowRun, "yes")
	assert.False(t, IsEnabled())
}

func TestEnvShadowRunConstant(t *testing.T) {
	assert.Equal(t, "BR_SHADOW_RUN", constants.EnvShadowRun)
}
