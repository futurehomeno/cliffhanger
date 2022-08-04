package mockedscenectrl

func (_m *Controller) MockSceneCtrlSceneReport(value string, err error, once bool) *Controller {
	c := _m.On("SceneCtrlSceneReport").Return(value, err)

	if once {
		c.Once()
	}

	return _m
}

func (_m *Controller) MockSetSceneCtrlScene(scene string, err error, once bool) *Controller {
	c := _m.On("SetSceneCtrlScene", scene).Return(err)

	if once {
		c.Once()
	}

	return _m
}
