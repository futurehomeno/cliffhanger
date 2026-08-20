package numericmeter_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockednumericmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericmeter"
)

func TestNewService_AdvertisedInterfaces(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		reporter   func(t *testing.T) numericmeter.Reporter
		wantIn     []string
		wantAbsent []string
	}{
		{
			name: "plain meter advertises neither reset nor export",
			reporter: func(t *testing.T) numericmeter.Reporter {
				t.Helper()

				return mockednumericmeter.NewReporter(t)
			},
			wantAbsent: []string{numericmeter.CmdMeterReset, numericmeter.CmdMeterExportGetReport},
		},
		{
			name: "resettable meter advertises cmd.meter.reset",
			reporter: func(t *testing.T) numericmeter.Reporter {
				t.Helper()

				return &resettableMeter{
					Reporter:           mockednumericmeter.NewReporter(t),
					ResettableReporter: mockednumericmeter.NewResettableReporter(t),
				}
			},
			wantIn:     []string{numericmeter.CmdMeterReset},
			wantAbsent: []string{numericmeter.CmdMeterExportGetReport},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			specification := numericmeter.Specification(
				numericmeter.MeterElec,
				"test_adapter",
				"1",
				"2",
				nil,
				numericmeter.Units{numericmeter.UnitW},
			)

			numericmeter.NewService(mockedadapter.NewServicePublisher(t), &numericmeter.Config{
				Specification: specification,
				Reporter:      tt.reporter(t),
			})

			for _, msgType := range tt.wantIn {
				assert.True(t, hasInterface(specification, msgType), "expected interface %s to be advertised", msgType)
			}

			for _, msgType := range tt.wantAbsent {
				assert.False(t, hasInterface(specification, msgType), "expected interface %s not to be advertised", msgType)
			}
		})
	}
}

// TestNewService_ResetInterfaceShape pins the reset interface's direction and value type, so a
// reset-capable meter keeps announcing the command the router actually handles.
func TestNewService_ResetInterfaceShape(t *testing.T) {
	t.Parallel()

	specification := numericmeter.Specification(
		numericmeter.MeterElec,
		"test_adapter",
		"1",
		"2",
		nil,
		numericmeter.Units{numericmeter.UnitW},
	)

	numericmeter.NewService(mockedadapter.NewServicePublisher(t), &numericmeter.Config{
		Specification: specification,
		Reporter: &resettableMeter{
			Reporter:           mockednumericmeter.NewReporter(t),
			ResettableReporter: mockednumericmeter.NewResettableReporter(t),
		},
	})

	var found *fimptype.Interface

	for i, intf := range specification.Interfaces {
		if intf.MsgType == numericmeter.CmdMeterReset {
			found = &specification.Interfaces[i]

			break
		}
	}

	if assert.NotNil(t, found, "reset interface must be advertised") {
		assert.Equal(t, fimptype.TypeIn, found.Type)
		assert.Equal(t, fimptype.VTypeNull, found.ValueType)
	}
}

func hasInterface(specification *fimptype.Service, msgType string) bool {
	for _, intf := range specification.Interfaces {
		if intf.MsgType == msgType {
			return true
		}
	}

	return false
}

var _ adapter.ServicePublisher = (*mockedadapter.ServicePublisher)(nil)
