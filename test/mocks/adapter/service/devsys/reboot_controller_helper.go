package mockeddevsys

func (_m *RebootController) MockRebootDevice(hard bool, err error, once bool) *RebootController {
	c := _m.On("RebootDevice", hard).Return(err)

	if once {
		c.Once()
	}

	return _m
}
