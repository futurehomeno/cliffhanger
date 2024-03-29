package mockednumericmeter

import (
	"github.com/stretchr/testify/mock"
)

type MockedMeter struct {
	*ExtendedReporter
	*Reporter
}

func (m *MockedMeter) AssertExpectations(t mock.TestingT) bool {
	if !m.ExtendedReporter.AssertExpectations(t) {
		return false
	}

	return m.Reporter.AssertExpectations(t)
}

func NewMockedMeter(reporter *Reporter, extendedReporter *ExtendedReporter) *MockedMeter {
	return &MockedMeter{
		Reporter:         reporter,
		ExtendedReporter: extendedReporter,
	}
}
