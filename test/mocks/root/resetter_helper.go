package mockedroot

func (_m *Resetter) MockReset(err error) *Resetter {
	_m.On("reset").Return(err)

	return _m
}
