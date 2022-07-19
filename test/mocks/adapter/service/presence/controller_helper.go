package mockedpresence

func (_m *Controller) MockSensorPresenceReport(value bool, err error, once bool) *Controller {
	c := _m.On("SensorPresenceReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
