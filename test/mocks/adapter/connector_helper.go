package mockedadapter

import (
	"testing"

	"github.com/stretchr/testify/mock"
)

func (_m *Connector) MockConnect() *Connector {
	_m.On("Connect", mock.Anything).Return()

	return _m
}

func (_m *Connector) MockDisconnect() *Connector {
	_m.On("Disconnect", mock.Anything).Return()

	return _m
}

func NewDefaultConnector(t *testing.T) *Connector {
	return NewConnector(t).MockConnect()
}
