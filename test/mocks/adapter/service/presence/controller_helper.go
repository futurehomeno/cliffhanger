package mockedpresence

func (_m *Controller) MockPresencePresenceReport(value bool, err error, once bool) *Controller {
	c := _m.On("PresencePresenceReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}
