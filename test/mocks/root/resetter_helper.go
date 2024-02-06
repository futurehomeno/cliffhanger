package mockedroot

func (_m *Resetter) MockReset(err error) *Resetter {
	_m.On("Reset").Return(err)

	return _m
}
