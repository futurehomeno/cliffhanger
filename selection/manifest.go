package selection

import (
	"fmt"

	"github.com/futurehomeno/cliffhanger/manifest"
)

// defaultLanguage is the language key of the block texts, matching the manifest templates.
const defaultLanguage = "en"

// Block identifies the manifest UI block and application config holding the device selector, and
// the texts shown when the selector cannot be rendered.
type Block struct {
	// Block is the ID of the UI block hosting the selector.
	Block string
	// Config is the ID of the application config holding the multi-select.
	Config string
	// NotReady is the block text shown when the application is not ready to list devices, e.g.
	// because the user is not logged in. It also hides the selector.
	NotReady string
	// Failed is the block text shown when the device fetch fails. The selector is left as the
	// template defines it, so a stale option list is never presented as current.
	Failed string
}

// PrepareManifest renders the device selector into the manifest, covering its three states: not
// ready (show NotReady and hide the selector), fetch failed (show Failed and return the error for
// the caller to log) and ready (fill the selector and show it).
//
// A manifest template that does not declare the referenced block or config is ignored rather than
// dereferenced, so renaming either in packaging cannot panic the application.
func PrepareManifest[T any](
	m *manifest.Manifest,
	b Block,
	ready bool,
	fetch func() ([]T, error),
	option func(T) manifest.SelectOption,
) error {
	block, config := m.GetUIBlock(b.Block), m.GetAppConfig(b.Config)

	if !ready {
		setText(block, b.NotReady)

		if config != nil {
			config.Hide()
		}

		return nil
	}

	devices, err := fetch()
	if err != nil {
		setText(block, b.Failed)

		return fmt.Errorf("selection: fetch available devices: %w", err)
	}

	if config == nil {
		return nil
	}

	options := make([]manifest.SelectOption, 0, len(devices))

	for _, device := range devices {
		options = append(options, option(device))
	}

	config.UI.Select = options
	config.Hidden = false

	return nil
}

func setText(block *manifest.AppUBLock, text string) {
	if block == nil || text == "" {
		return
	}

	if block.Text == nil {
		block.Text = manifest.MultilingualLabel{}
	}

	block.Text[defaultLanguage] = text
}
