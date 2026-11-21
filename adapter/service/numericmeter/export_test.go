package numericmeter_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockednumericmeter "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/numericmeter"
)

type exportMeter struct {
	*mockednumericmeter.Reporter
	*mockednumericmeter.ExportReporter
}

func newExportMeter(t *testing.T, publisher adapter.ServicePublisher, reporter numericmeter.Reporter, options ...adapter.SpecificationOption) numericmeter.Service {
	t.Helper()

	return numericmeter.NewService(publisher, &numericmeter.Config{
		Specification: numericmeter.Specification(
			numericmeter.MeterElec, "test_adapter", "1", "2", nil,
			numericmeter.Units{numericmeter.UnitKWh}, options...,
		),
		Reporter: reporter,
	})
}

func TestService_SendMeterExportReport(t *testing.T) {
	t.Parallel()

	exporter := mockednumericmeter.NewExportReporter(t)
	exporter.On("MeterExportReport", numericmeter.UnitKWh).Return(12.5, nil).Twice()

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.MatchedBy(func(m *fimpgo.FimpMessage) bool {
		return m.Interface == numericmeter.EvtMeterExportReport &&
			m.Properties[numericmeter.PropertyUnit] == "kWh" &&
			m.Properties[numericmeter.PropertyIsVirtual] == "false" &&
			m.Storage != nil && m.Storage.Strategy == fimpgo.StorageStrategyAggregate && m.Storage.SubValue == "kWh"
	})).Return(nil).Once()

	svc := newExportMeter(t, publisher,
		&exportMeter{Reporter: mockednumericmeter.NewReporter(t), ExportReporter: exporter},
		numericmeter.WithExportUnits(numericmeter.UnitKWh),
	)

	assert.True(t, svc.SupportsExportReport())
	assert.True(t, hasInterface(svc.Specification(), numericmeter.CmdMeterExportGetReport))
	assert.True(t, hasInterface(svc.Specification(), numericmeter.EvtMeterExportReport))

	sent, err := svc.SendMeterExportReport("KWH", false)
	require.NoError(t, err)
	assert.True(t, sent, "a unit sent in another case must be reported under the advertised spelling")

	sent, err = svc.SendMeterExportReport(numericmeter.UnitKWh, false)
	require.NoError(t, err)
	assert.False(t, sent, "an unchanged export value must not be republished")

	_, err = svc.SendMeterExportReport(numericmeter.UnitW, true)
	assert.ErrorContains(t, err, "unit is unsupported")
}

func TestService_ExportReportNeedsBothReporterAndUnits(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		reporter numericmeter.Reporter
		options  []adapter.SpecificationOption
	}{
		{
			name:     "export reporter without sup_export_units",
			reporter: &exportMeter{Reporter: mockednumericmeter.NewReporter(t), ExportReporter: mockednumericmeter.NewExportReporter(t)},
		},
		{
			name:     "sup_export_units without export reporter",
			reporter: mockednumericmeter.NewReporter(t),
			options:  []adapter.SpecificationOption{numericmeter.WithExportUnits(numericmeter.UnitKWh)},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			svc := newExportMeter(t, mockedadapter.NewServicePublisher(t), tt.reporter, tt.options...)

			assert.False(t, svc.SupportsExportReport())
			assert.False(t, hasInterface(svc.Specification(), numericmeter.EvtMeterExportReport))

			_, err := svc.SendMeterExportReport(numericmeter.UnitKWh, true)
			assert.ErrorContains(t, err, "export meter report is not supported")
		})
	}
}

func TestService_SendMeterReport_PublishFailureIsRetried(t *testing.T) {
	t.Parallel()

	reporter := mockednumericmeter.NewReporter(t).
		MockMeterReport(numericmeter.UnitKWh, 7, nil, false)

	publisher := mockedadapter.NewServicePublisher(t)
	publisher.On("PublishServiceMessage", mock.Anything, mock.Anything).Return(errors.New("broker down")).Once()
	publisher.On("PublishServiceMessage", mock.Anything, mock.Anything).Return(nil).Once()

	svc := newExportMeter(t, publisher, reporter)

	sent, err := svc.SendMeterReport(numericmeter.UnitKWh, false)
	assert.False(t, sent)
	assert.ErrorContains(t, err, "broker down")

	sent, err = svc.SendMeterReport(numericmeter.UnitKWh, false)
	require.NoError(t, err)
	assert.True(t, sent, "a value whose publication failed must not be cached as reported")

	sent, err = svc.SendMeterReport(numericmeter.UnitKWh, false)
	require.NoError(t, err)
	assert.False(t, sent)
}
