package onec

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Kargones/apk-ci/internal/util/runner"
)

func TestUpdater_UpdateDBCfg_EmptyBinPath(t *testing.T) {
	u := NewUpdater("", "/work", "/tmp")

	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
	})

	assert.Nil(t, result)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "путь к 1cv8 не указан")
}

func TestUpdater_UpdateDBCfg_WithOptionsBinPath(t *testing.T) {
	u := NewUpdater("", "/work", "/tmp")

	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
		Bin1cv8:       "/usr/bin/1cv8",
	})

	// Will fail because 1cv8 doesn't exist
	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestUpdater_UpdateDBCfg_WithExtension(t *testing.T) {
	u := NewUpdater("/usr/bin/1cv8", "/work", "/tmp")

	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
		Extension:     "MyExtension",
	})

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestUpdater_UpdateDBCfg_WithTimeout(t *testing.T) {
	u := NewUpdater("/usr/bin/1cv8", "/work", "/tmp")

	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
		Timeout:       5 * time.Second,
	})

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestUpdater_UpdateDBCfg_CancelledContext(t *testing.T) {
	u := NewUpdater("/usr/bin/1cv8", "/work", "/tmp")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	result, err := u.UpdateDBCfg(ctx, UpdateOptions{
		ConnectString: "/S server\\base",
	})

	require.Error(t, err)
	assert.NotNil(t, result)
	assert.False(t, result.Success)
}

func TestAddDisableParam(t *testing.T) {
	r := &runner.Runner{}
	addDisableParam(r)

	// Check that the expected params were added
	assert.Contains(t, r.Params, "/DisableStartupDialogs")
	assert.Contains(t, r.Params, "/DisableStartupMessages")
	assert.Contains(t, r.Params, "/DisableUnrecoverableErrorMessage")
	assert.Contains(t, r.Params, "/UC ServiceMode")
}

func TestUpdater_UpdateDBCfg_OptionsTakePrecedence(t *testing.T) {
	// When Bin1cv8 is provided in options, it should take precedence
	u := NewUpdater("/default/1cv8", "/work", "/tmp")

	result, err := u.UpdateDBCfg(context.Background(), UpdateOptions{
		ConnectString: "/S server\\base",
		Bin1cv8:       "/custom/1cv8", // This should be used
	})

	// Will fail because the binary doesn't exist
	require.Error(t, err)
	assert.NotNil(t, result)
}

func TestUpdater_NewUpdater_AllFields(t *testing.T) {
	u := NewUpdater("/opt/1cv8/1cv8", "/var/work", "/var/tmp")

	assert.Equal(t, "/opt/1cv8/1cv8", u.bin1cv8)
	assert.Equal(t, "/var/work", u.workDir)
	assert.Equal(t, "/var/tmp", u.tmpDir)
}
