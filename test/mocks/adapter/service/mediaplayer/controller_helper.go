package mockedmediaplayer

// MockedMediaPlayerPlaybackSet is a helper function that sets up a mock for the SetPlayback method.
func (_m *Controller) MockedMediaPlayerPlaybackSet(value string, err error, once bool) *Controller {
	c := _m.On("SetPlayback", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

// MockedMediaPlayerPlaybackReport is a helper function that sets up a mock for the Playback method.
func (_m *Controller) MockedMediaPlayerPlaybackReport(value string, err error, once bool) *Controller {
	c := _m.On("Playback").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

// MockedMediaPlayerPlaybackModeSet is a helper function that sets up a mock for the SetPlaybackMode method.
func (_m *Controller) MockedMediaPlayerPlaybackModeSet(value map[string]bool, err error, once bool) *Controller {
	c := _m.On("SetPlaybackMode", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

// MockedMediaPlayerPlaybackModeReport is a helper function that sets up a mock for the PlaybackMode method.
func (_m *Controller) MockedMediaPlayerPlaybackModeReport(value map[string]bool, err error, once bool) *Controller {
	c := _m.On("PlaybackMode").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

// MockedMediaPlayerVolumeSet is a helper function that sets up a mock for the SetVolume method.
func (_m *Controller) MockedMediaPlayerVolumeSet(value int64, err error, once bool) *Controller {
	c := _m.On("SetVolume", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

// MockedMediaPlayerVolumeReport is a helper function that sets up a mock for the Volume method.
func (_m *Controller) MockedMediaPlayerVolumeReport(value int64, err error, once bool) *Controller {
	c := _m.On("Volume").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

// MockedMediaPlayerMuteSet is a helper function that sets up a mock for the SetMute method.
func (_m *Controller) MockedMediaPlayerMuteSet(value bool, err error, once bool) *Controller {
	c := _m.On("SetMute", value).Return(err)

	if once {
		c.Once()
	}

	return _m
}

// MockedMediaPlayerMuteReport is a helper function that sets up a mock for the Mute method.
func (_m *Controller) MockedMediaPlayerMuteReport(value bool, err error, once bool) *Controller {
	c := _m.On("Mute").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

// MockedMediaPlayerMetadataReport is a helper function that sets up a mock for the Metadata method.
func (_m *Controller) MockedMediaPlayerMetadataReport(value map[string]string, err error, once bool) *Controller {
	c := _m.On("Metadata").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
