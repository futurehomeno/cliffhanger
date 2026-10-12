package selection_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/manifest"
	"github.com/futurehomeno/cliffhanger/selection"
)

const (
	testBlockID  = "configuration"
	testConfigID = "selected_devices"
)

var testBlock = selection.Block{
	Block:    testBlockID,
	Config:   testConfigID,
	NotReady: "Please log in to continue.",
	Failed:   "Failed to retrieve devices, please try again later.",
}

type testDevice struct{ ID, Name string }

func testDeviceOption(d testDevice) manifest.SelectOption {
	return manifest.SelectOption{Val: d.ID, Label: map[string]string{"en": d.Name}}
}

func testManifest() *manifest.Manifest {
	return &manifest.Manifest{
		UIBlocks: []manifest.AppUBLock{{ID: testBlockID, Text: manifest.MultilingualLabel{"en": "template"}}},
		Configs:  []manifest.AppConfig{{ID: testConfigID, Hidden: true}},
	}
}

func TestPrepareManifest_NotReadyHidesTheSelector(t *testing.T) {
	t.Parallel()

	m := testManifest()

	err := selection.PrepareManifest(m, testBlock, false, func() ([]testDevice, error) {
		t.Fatal("the device list must not be fetched when the application is not ready")

		return nil, nil
	}, testDeviceOption)

	require.NoError(t, err)
	assert.Equal(t, testBlock.NotReady, m.GetUIBlock(testBlockID).Text["en"])
	assert.True(t, m.GetAppConfig(testConfigID).Hidden)
}

func TestPrepareManifest_FetchFailureLeavesTheSelectorAlone(t *testing.T) {
	t.Parallel()

	errFetch := errors.New("fetch failed")
	m := testManifest()

	err := selection.PrepareManifest(m, testBlock, true, func() ([]testDevice, error) {
		return nil, errFetch
	}, testDeviceOption)

	assert.ErrorIs(t, err, errFetch)
	assert.Equal(t, testBlock.Failed, m.GetUIBlock(testBlockID).Text["en"])
	// A stale option list is never presented as current.
	assert.True(t, m.GetAppConfig(testConfigID).Hidden)
	assert.Nil(t, m.GetAppConfig(testConfigID).UI.Select)
}

func TestPrepareManifest_ReadyFillsTheSelector(t *testing.T) {
	t.Parallel()

	m := testManifest()

	err := selection.PrepareManifest(m, testBlock, true, func() ([]testDevice, error) {
		return []testDevice{{"1", "first"}, {"2", "second"}}, nil
	}, testDeviceOption)

	require.NoError(t, err)

	config := m.GetAppConfig(testConfigID)

	assert.False(t, config.Hidden)
	assert.Equal(t, []manifest.SelectOption{
		{Val: "1", Label: map[string]string{"en": "first"}},
		{Val: "2", Label: map[string]string{"en": "second"}},
	}, config.UI.Select)
	// The block text is left as the template wrote it when there is nothing to explain.
	assert.Equal(t, "template", m.GetUIBlock(testBlockID).Text["en"])
}

func TestPrepareManifest_ToleratesAMissingBlockOrConfig(t *testing.T) {
	t.Parallel()

	// Renaming either in packaging must not panic the application on cmd.app.get_manifest.
	empty := &manifest.Manifest{}

	require.NoError(t, selection.PrepareManifest(empty, testBlock, false, func() ([]testDevice, error) {
		return nil, nil
	}, testDeviceOption))

	require.NoError(t, selection.PrepareManifest(empty, testBlock, true, func() ([]testDevice, error) {
		return []testDevice{{"1", "first"}}, nil
	}, testDeviceOption))
}
