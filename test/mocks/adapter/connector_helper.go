package mockedadapter

import (
	"testing"

	"github.com/futurehomeno/cliffhanger/adapter"
)

func NewDefaultConnector(t *testing.T) *Connector {
	t.Helper()
	return NewConnector(t).MockConnectivity(&adapter.ConnectivityDetails{
		ConnStatus:       adapter.ConnStatusUp,
		Operationability: nil,
		ConnQuality:      adapter.ConnQualityUndefined,
		ConnType:         adapter.ConnTypeIndirect,
	}, false)
}

func (_m *Connector) MockConnectivity(details *adapter.ConnectivityDetails, once bool) *Connector {
	c := _m.On("Connectivity").Return(details)

	if once {
		c.Once()
	} else {
		c.Maybe()
	}

	return _m
}
