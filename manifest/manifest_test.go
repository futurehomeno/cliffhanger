package manifest_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/manifest"
)

func TestManifest_GetUIBlock(t *testing.T) {
	t.Parallel()

	m := manifest.New()
	m.UIBlocks = []manifest.AppUBLock{
		{ID: "block1"},
		{ID: "block2"},
	}

	assert.NotNil(t, m.GetUIBlock("block1"))
	assert.Equal(t, "block1", m.GetUIBlock("block1").ID)
	assert.NotNil(t, m.GetUIBlock("block2"))
	assert.Nil(t, m.GetUIBlock("missing"))
}

func TestManifest_GetUIBlock_ReturnsPointerToSliceElement(t *testing.T) {
	t.Parallel()

	m := manifest.New()
	m.UIBlocks = []manifest.AppUBLock{{ID: "block1"}}

	m.GetUIBlock("block1").Hide()

	assert.True(t, m.UIBlocks[0].Hidden)
}

func TestManifest_GetButton(t *testing.T) {
	t.Parallel()

	m := manifest.New()
	m.UIButtons = []manifest.UIButton{
		{ID: "btn1"},
		{ID: "btn2"},
	}

	assert.NotNil(t, m.GetButton("btn1"))
	assert.Equal(t, "btn1", m.GetButton("btn1").ID)
	assert.Nil(t, m.GetButton("missing"))
}

func TestManifest_GetButton_ReturnsPointerToSliceElement(t *testing.T) {
	t.Parallel()

	m := manifest.New()
	m.UIButtons = []manifest.UIButton{{ID: "btn1"}}

	m.GetButton("btn1").Hide()

	assert.True(t, m.UIButtons[0].Hidden)
}

func TestManifest_GetAppConfig(t *testing.T) {
	t.Parallel()

	m := manifest.New()
	m.Configs = []manifest.AppConfig{
		{ID: "cfg1"},
		{ID: "cfg2"},
	}

	assert.NotNil(t, m.GetAppConfig("cfg1"))
	assert.Equal(t, "cfg1", m.GetAppConfig("cfg1").ID)
	assert.Nil(t, m.GetAppConfig("missing"))
}

func TestManifest_GetAppConfig_ReturnsPointerToSliceElement(t *testing.T) {
	t.Parallel()

	m := manifest.New()
	m.Configs = []manifest.AppConfig{{ID: "cfg1"}}

	m.GetAppConfig("cfg1").Hide()

	assert.True(t, m.Configs[0].Hidden)
}

func TestAppConfig_HideShow(t *testing.T) {
	t.Parallel()

	cfg := &manifest.AppConfig{}

	assert.False(t, cfg.Hidden)
	cfg.Hide()
	assert.True(t, cfg.Hidden)
	cfg.Show()
	assert.False(t, cfg.Hidden)
}

func TestUIButton_HideShow(t *testing.T) {
	t.Parallel()

	btn := &manifest.UIButton{}

	assert.False(t, btn.Hidden)
	btn.Hide()
	assert.True(t, btn.Hidden)
	btn.Show()
	assert.False(t, btn.Hidden)
}

func TestAppUBLock_HideShow(t *testing.T) {
	t.Parallel()

	block := &manifest.AppUBLock{}

	assert.False(t, block.Hidden)
	block.Hide()
	assert.True(t, block.Hidden)
	block.Show()
	assert.False(t, block.Hidden)
}
