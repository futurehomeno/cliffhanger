package hub_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/hub"
)

func TestLoadInfo_CustomPath(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "hub.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"hub_id":"h1","site_id":"s1","environment":"beta","cloud_api_root_url":"https://x"}`), 0o600))

	info, err := hub.LoadInfo(path)
	require.NoError(t, err)
	assert.Equal(t, &hub.Info{HubID: "h1", SiteID: "s1", Environment: hub.EnvBeta, CloudAPIRootURL: "https://x"}, info)
}

func TestLoadInfo_MalformedJSON(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "hub.json")
	require.NoError(t, os.WriteFile(path, []byte(`{"hub_id":`), 0o600))

	info, err := hub.LoadInfo(path)
	require.ErrorContains(t, err, "unmarshal")
	assert.Nil(t, info)
}

func TestLoadInfo_UnreadablePathFallsBackToV2File(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("/var/lib/futurehome/hub/hub_v2.json"); err == nil {
		t.Skip("host has a real hub_v2.json")
	}

	info, err := hub.LoadInfo(filepath.Join(t.TempDir(), "missing.json"))
	require.ErrorContains(t, err, "/var/lib/futurehome/hub/hub_v2.json", "the error names the fallback path")
	assert.Nil(t, info)
}
