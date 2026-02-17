package rac

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMockRAC creates a temporary mock RAC script that outputs predefined data
func createMockRAC(t *testing.T, output string) (racPath string, cleanup func()) {
	t.Helper()
	tmpDir := t.TempDir()
	racPath = filepath.Join(tmpDir, "mock-rac")
	
	script := "#!/bin/sh\ncat << 'EOF'\n" + output + "\nEOF\n"
	
	err := os.WriteFile(racPath, []byte(script), 0755)
	require.NoError(t, err)
	
	return racPath, func() {}
}

// === Tests for GetClusterInfo ===

func TestGetClusterInfo_Success(t *testing.T) {
	output := "cluster        : 2e4b5c7a-8d3f-4a1b-9c6e-f0d2a3b4c5d6\nhost           : server-1c\nport           : 1541\nname           : \"Central cluster\""
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	info, err := c.GetClusterInfo(context.Background())
	require.NoError(t, err)
	assert.Equal(t, "2e4b5c7a-8d3f-4a1b-9c6e-f0d2a3b4c5d6", info.UUID)
	assert.Equal(t, "server-1c", info.Host)
	assert.Equal(t, 1541, info.Port)
	assert.Equal(t, "Central cluster", info.Name)
}

func TestGetClusterInfo_NoCluster(t *testing.T) {
	racPath, cleanup := createMockRAC(t, "")
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	_, err = c.GetClusterInfo(context.Background())
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrRACNotFound)
}

// === Tests for GetInfobaseInfo ===

func TestGetInfobaseInfo_Success(t *testing.T) {
	output := "infobase       : b2c3d4e5-f6a7-8901-bcde-f12345678901\nname           : TestBase\ndescr          : \"Test database\"\n\ninfobase       : a1b2c3d4-e5f6-7890-abcd-ef1234567890\nname           : AnotherBase\ndescr          : \"Another database\""
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	info, err := c.GetInfobaseInfo(context.Background(), "cluster-uuid", "TestBase")
	require.NoError(t, err)
	assert.Equal(t, "b2c3d4e5-f6a7-8901-bcde-f12345678901", info.UUID)
	assert.Equal(t, "TestBase", info.Name)
	assert.Equal(t, "Test database", info.Description)
}

func TestGetInfobaseInfo_NotFound(t *testing.T) {
	output := "infobase       : b2c3d4e5-f6a7-8901-bcde-f12345678901\nname           : OtherBase\ndescr          : \"Other database\""
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	_, err = c.GetInfobaseInfo(context.Background(), "cluster-uuid", "NonExistentBase")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrRACNotFound)
}

// === Tests for GetSessions ===

func TestGetSessions_Success(t *testing.T) {
	output := "session        : a1b2c3d4-e5f6-7890-abcd-ef1234567890\nuser-name      : User1\napp-id         : 1CV8C\nhost           : 192.168.1.100\n\nsession        : b2c3d4e5-f6a7-8901-bcde-f12345678901\nuser-name      : User2\napp-id         : 1CV8\nhost           : 192.168.1.101"
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	sessions, err := c.GetSessions(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
	require.Len(t, sessions, 2)
	assert.Equal(t, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", sessions[0].SessionID)
	assert.Equal(t, "User1", sessions[0].UserName)
	assert.Equal(t, "b2c3d4e5-f6a7-8901-bcde-f12345678901", sessions[1].SessionID)
	assert.Equal(t, "User2", sessions[1].UserName)
}

func TestGetSessions_Empty(t *testing.T) {
	racPath, cleanup := createMockRAC(t, "")
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	sessions, err := c.GetSessions(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
	assert.Empty(t, sessions)
}

// === Tests for TerminateSession ===

func TestTerminateSession_Success(t *testing.T) {
	racPath, cleanup := createMockRAC(t, "")
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.TerminateSession(context.Background(), "cluster-uuid", "session-id")
	require.NoError(t, err)
}

// === Tests for TerminateAllSessions ===

func TestTerminateAllSessions_NoSessions(t *testing.T) {
	racPath, cleanup := createMockRAC(t, "")
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.TerminateAllSessions(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
}

func TestTerminateAllSessions_WithSessions(t *testing.T) {
	tmpDir := t.TempDir()
	racPath := filepath.Join(tmpDir, "mock-rac")
	
	script := `#!/bin/sh
if echo "$@" | grep -q "session list"; then
cat << 'SESSIONS'
session        : a1b2c3d4-e5f6-7890-abcd-ef1234567890
user-name      : User1

session        : b2c3d4e5-f6a7-8901-bcde-f12345678901
user-name      : User2
SESSIONS
fi
`
	err := os.WriteFile(racPath, []byte(script), 0755)
	require.NoError(t, err)

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.TerminateAllSessions(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
}

// === Tests for EnableServiceMode ===

func TestEnableServiceMode_Success(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : off\nscheduled-jobs-deny : off"
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.EnableServiceMode(context.Background(), "cluster-uuid", "infobase-uuid", false)
	require.NoError(t, err)
}

func TestEnableServiceMode_WithTerminateSessions(t *testing.T) {
	racPath, cleanup := createMockRAC(t, "")
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.EnableServiceMode(context.Background(), "cluster-uuid", "infobase-uuid", true)
	require.NoError(t, err)
}

func TestEnableServiceMode_WithScheduledJobsBlocked(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : off\nscheduled-jobs-deny : on"
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.EnableServiceMode(context.Background(), "cluster-uuid", "infobase-uuid", false)
	require.NoError(t, err)
}

// === Tests for DisableServiceMode ===

func TestDisableServiceMode_Success(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : on\nscheduled-jobs-deny : on\ndenied-message      : \"Service mode\""
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.DisableServiceMode(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
}

func TestDisableServiceMode_WithDotMarker(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : on\nscheduled-jobs-deny : on\ndenied-message      : \"Service mode.\""
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.DisableServiceMode(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
}

// === Tests for GetServiceModeStatus ===

func TestGetServiceModeStatus_Enabled(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : on\nscheduled-jobs-deny : on\ndenied-message      : \"Database update\""
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	status, err := c.GetServiceModeStatus(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
	assert.True(t, status.Enabled)
	assert.True(t, status.ScheduledJobsBlocked)
	assert.Equal(t, "Database update", status.Message)
}

func TestGetServiceModeStatus_Disabled(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : off\nscheduled-jobs-deny : off"
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	status, err := c.GetServiceModeStatus(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
	assert.False(t, status.Enabled)
	assert.False(t, status.ScheduledJobsBlocked)
}

func TestGetServiceModeStatus_WithSessions(t *testing.T) {
	tmpDir := t.TempDir()
	racPath := filepath.Join(tmpDir, "mock-rac")
	
	script := `#!/bin/sh
if echo "$@" | grep -q "infobase info"; then
cat << 'INFO'
infobase            : test-uuid
sessions-deny       : on
scheduled-jobs-deny : on
denied-message      : "Service mode"
INFO
elif echo "$@" | grep -q "session list"; then
cat << 'SESSIONS'
session        : a1b2c3d4-e5f6-7890-abcd-ef1234567890
user-name      : User1

session        : b2c3d4e5-f6a7-8901-bcde-f12345678901
user-name      : User2
SESSIONS
fi
`
	err := os.WriteFile(racPath, []byte(script), 0755)
	require.NoError(t, err)

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	status, err := c.GetServiceModeStatus(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
	assert.True(t, status.Enabled)
	assert.Equal(t, 2, status.ActiveSessions)
}

func TestGetServiceModeStatus_SessionError(t *testing.T) {
	tmpDir := t.TempDir()
	racPath := filepath.Join(tmpDir, "mock-rac")
	
	script := `#!/bin/sh
if echo "$@" | grep -q "infobase info"; then
cat << 'INFO'
infobase            : test-uuid
sessions-deny       : on
scheduled-jobs-deny : on
denied-message      : "Service mode"
INFO
else
    exit 1
fi
`
	err := os.WriteFile(racPath, []byte(script), 0755)
	require.NoError(t, err)

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	status, err := c.GetServiceModeStatus(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
	assert.True(t, status.Enabled)
	assert.Equal(t, 0, status.ActiveSessions)
}

// === Tests for VerifyServiceMode ===

func TestVerifyServiceMode_MatchEnabled(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : on\nscheduled-jobs-deny : on\ndenied-message      : \"Service mode\""
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.VerifyServiceMode(context.Background(), "cluster-uuid", "infobase-uuid", true)
	require.NoError(t, err)
}

func TestVerifyServiceMode_MatchDisabled(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : off\nscheduled-jobs-deny : off"
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.VerifyServiceMode(context.Background(), "cluster-uuid", "infobase-uuid", false)
	require.NoError(t, err)
}

func TestVerifyServiceMode_Mismatch(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : on\nscheduled-jobs-deny : on"
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	err = c.VerifyServiceMode(context.Background(), "cluster-uuid", "infobase-uuid", false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrRACVerify)
}

// === Tests for getInfobaseRawStatus ===

func TestGetInfobaseRawStatus_Success(t *testing.T) {
	output := "infobase            : test-uuid\nsessions-deny       : on\nscheduled-jobs-deny : on\ndenied-message      : \"Test message\""
	racPath, cleanup := createMockRAC(t, output)
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	rc := c.(*racClient)
	status, err := rc.getInfobaseRawStatus(context.Background(), "cluster-uuid", "infobase-uuid")
	require.NoError(t, err)
	assert.True(t, status.Enabled)
	assert.True(t, status.ScheduledJobsBlocked)
	assert.Equal(t, "Test message", status.Message)
}

func TestGetInfobaseRawStatus_NoData(t *testing.T) {
	racPath, cleanup := createMockRAC(t, "")
	defer cleanup()

	c, err := NewClient(ClientOptions{
		RACPath: racPath,
		Server:  "localhost",
	})
	require.NoError(t, err)

	rc := c.(*racClient)
	_, err = rc.getInfobaseRawStatus(context.Background(), "cluster-uuid", "infobase-uuid")
	require.Error(t, err)
	assert.Contains(t, err.Error(), ErrRACNotFound)
}
