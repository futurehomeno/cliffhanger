package mockedvirtualmeter

import (
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/stretchr/testify/mock"
)

type (
	ManagerFull struct {
		*Manager
		*virtualmeter.MockedPrivateManager
	}
)

func NewFullManager(t interface {
	mock.TestingT
	Cleanup(func())
}) *ManagerFull {
	return &ManagerFull{
		Manager:              NewManager(t),
		MockedPrivateManager: virtualmeter.NewMockedPrivateManager(t),
	}
}

func (m *ManagerFull) WithUpdateRequired(val, once bool, args ...interface{}) *ManagerFull {
	c := m.Manager.On("UpdateRequired", args...).Return(val)

	if once {
		c.Once()
	}

	return m
}

func (m *ManagerFull) WithUpdate(err error, once bool, args ...interface{}) *ManagerFull {
	c := m.Manager.On("Update", args...).Return(err)

	if once {
		c.Once()
	}

	return m
}

func (m *ManagerFull) WithUpdateDeviceActivity(err error, once bool, args ...interface{}) *ManagerFull {
	c := m.MockedPrivateManager.On("updateDeviceActivity", args...).Return(err)

	if once {
		c.Once()
	}

	return m
}
