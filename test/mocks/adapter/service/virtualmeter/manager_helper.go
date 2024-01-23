package mockedvirtualmeter

func (m *Manager) WithUpdateRequired(val, once bool, args ...interface{}) *Manager {
	c := m.On("updateRequired", args...).Return(val)

	if once {
		c.Once()
	}

	return m
}

func (m *Manager) WithUpdate(err error, once bool, args ...interface{}) *Manager {
	c := m.On("update", args...).Return(err)

	if once {
		c.Once()
	}

	return m
}

func (m *Manager) WithUpdateDeviceActivity(err error, once bool, args ...interface{}) *Manager {
	c := m.On("updateDeviceActivity", args...).Return(err)

	if once {
		c.Once()
	}

	return m
}
