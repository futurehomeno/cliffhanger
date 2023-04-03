package mockedadapter

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

func (_m *ControllableConnector) MockConnect() *ControllableConnector {
	_m.On("Connect", mock.Anything).Return()

	return _m
}

func (_m *ControllableConnector) MockDisconnect() *ControllableConnector {
	_m.On("Disconnect", mock.Anything).Return()

	return _m
}

func NewDefaultControllableConnector(t *testing.T) *ControllableConnector {
	return NewControllableConnector(t).MockConnect()
}
