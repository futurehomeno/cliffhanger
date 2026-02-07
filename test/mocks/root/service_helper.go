package mockedroot

func (_m *Service) MockStart(err error) *Service {
	_m.On("Start").Return(err)

	return _m
}

func (_m *Service) MockStop(err error) *Service {
	_m.On("Stop").Return(err)

	return _m
}
