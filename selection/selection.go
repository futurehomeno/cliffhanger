// Package selection models the user-chosen subset of a third-party account's devices: its
// storage in the application configuration, its rendering into the manifest and its maintenance
// when a device is deleted from the hub.
package selection

import "slices"

// Selection is a user's chosen subset of device IDs.
//
// A nil Selection includes every available device; a non-nil, zero-length Selection includes
// none. The distinction is load-bearing - an application that has never been configured must not
// be mistaken for one where the user deselected everything - and it survives JSON in both
// directions: an absent key or null unmarshals to nil and marshals back to null, while []
// unmarshals to an empty non-nil Selection and marshals back to [].
//
// Every copy taken in this package preserves it too. Note that the idiomatic
// append([]string(nil), s...) and slices.Clone both collapse empty to nil, so use Clone.
type Selection []string

// IncludeAll reports whether the selection includes every available device.
func (s Selection) IncludeAll() bool { return s == nil }

// Contains reports whether a device is included. It is always true for a nil Selection.
func (s Selection) Contains(id string) bool {
	return s == nil || slices.Contains(s, id)
}

// Clone returns a copy that preserves the nil/empty distinction.
func (s Selection) Clone() Selection {
	if s == nil {
		return nil
	}

	return append(make(Selection, 0, len(s)), s...)
}

// Without returns a copy with the device removed and reports whether anything was removed. A nil
// Selection is returned unchanged: "every device" cannot express an exclusion, so an application
// that supports device deletion must keep an explicit selection.
func (s Selection) Without(id string) (Selection, bool) {
	if s == nil || !slices.Contains(s, id) {
		return s, false
	}

	return slices.DeleteFunc(s.Clone(), func(v string) bool { return v == id }), true
}

// Devices is a configuration mixin carrying a device selection. Embed it in an application's
// public configuration to get the standard selected_devices field.
type Devices struct {
	SelectedDevices Selection `json:"selected_devices"`
}

// Selection returns a pointer to the stored selection, satisfying Selectable.
func (d *Devices) Selection() *Selection { return &d.SelectedDevices }

// Selectable is a configuration model carrying a device selection. It is satisfied by any model
// embedding Devices.
type Selectable interface {
	Selection() *Selection
}
