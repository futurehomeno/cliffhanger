package thing_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/adapter/thing"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	mockedmeterelec "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/meterelec"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var (
	evChargerTestStates        = []string{"ready_to_charge", "charging", "error"}
	evChargerChargingTestModes = []string{"normal", "slow"}

	errTest = errors.New("oops")
)

func TestRouteCarCharger(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful start charging routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging("", nil, false).
						MockChargepointStateReport("charging", nil, false).
						MockChargepointCurrentSessionReport(1.74, nil, false),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "start charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.start", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "charging"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.74),
						},
					},
					{
						Name: "start charging with mode unsupported by controller",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.start", "chargepoint").
							AddProperty(chargepoint.PropertyChargingMode, "slow").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "charging"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.74),
						},
					},
				},
			},
			{
				Name:     "successful start charging routing with mode support",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging("normal", nil, true).
						MockChargepointStateReport("charging", nil, true).
						MockChargepointCurrentSessionReport(1.74, nil, true),
					nil,
					evChargerChargingTestModes,
				),
				Nodes: []*suite.Node{
					{
						Name: "start charging",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.start", "chargepoint").
							AddProperty(chargepoint.PropertyChargingMode, "Normal").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "charging"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.74),
						},
					},
				},
			},
			{
				Name:     "successful stop charging routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStopChargepointCharging(nil, true).
						MockChargepointStateReport("ready_to_charge", nil, true).
						MockChargepointCurrentSessionReport(1.74, nil, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "stop charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.stop", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "ready_to_charge"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.74),
						},
					},
				},
			},
			{
				Name:     "other successful routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockSetChargepointCableLock(true, nil, true).
						MockChargepointCableLockReport(true, nil, false).
						MockChargepointStateReport("charging", nil, true).
						MockChargepointCurrentSessionReport(1.74, nil, true),
					mockedmeterelec.NewReporter(t).
						MockElectricityMeterReport("W", 2, nil, false).
						MockElectricityMeterReport("kWh", 123.45, nil, false),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set cable lock",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.cable_lock.set", "chargepoint", true),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.cable_lock.report", "chargepoint", true),
						},
					},
					{
						Name:    "cable lock report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.cable_lock.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.cable_lock.report", "chargepoint", true),
						},
					},
					{
						Name:    "state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.state.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "charging"),
						},
					},
					{
						Name:    "current session report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.74),
						},
					},
					{
						Name:    "power",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "W"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).
								ExpectProperty("unit", "W"),
						},
					},
					{
						Name:    "energy",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "kWh"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "all electricity units",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).
								ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).
								ExpectProperty("unit", "kWh"),
						},
					},
					{
						Name:    "all electricity units with null",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).
								ExpectProperty("unit", "W"),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).
								ExpectProperty("unit", "kWh"),
						},
					},
				},
			},
			{
				Name:     "failed start charging routing - starting error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging("", errTest, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "start charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.start", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed start charging routing - state report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging("", nil, true).
						MockChargepointStateReport("", errTest, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "start charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.start", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed start charging routing - session report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging("", nil, true).
						MockChargepointStateReport("ready_to_charge", nil, true).
						MockChargepointCurrentSessionReport(0, errTest, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "start charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.start", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed start charging routing - unsupported charging mode",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t),
					nil,
					evChargerChargingTestModes,
				),
				Nodes: []*suite.Node{
					{
						Name: "start charging",
						Command: suite.NewMessageBuilder().
							NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.start", "chargepoint").
							AddProperty(chargepoint.PropertyChargingMode, "dummy").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed stop charging routing - stopping error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStopChargepointCharging(errTest, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "stop charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.stop", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed stop charging routing - state report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStopChargepointCharging(nil, true).
						MockChargepointStateReport("", errTest, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "stop charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.stop", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed stop charging routing - session report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockStopChargepointCharging(nil, true).
						MockChargepointStateReport("ready_to_charge", nil, true).
						MockChargepointCurrentSessionReport(0, errTest, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "stop charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.charge.stop", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed cable lock routing - setter error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockSetChargepointCableLock(true, errTest, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set cable lock",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.cable_lock.set", "chargepoint", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed cable lock routing - cable lock report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockSetChargepointCableLock(true, nil, true).
						MockChargepointCableLockReport(false, errTest, true),
					nil,
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set cable lock",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.cable_lock.set", "chargepoint", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "other failed routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeCarCharger(
					mockedchargepoint.NewController(t).
						MockChargepointCableLockReport(false, errTest, false).
						MockChargepointStateReport("", errTest, true).
						MockChargepointCurrentSessionReport(0, errTest, true),
					mockedmeterelec.NewReporter(t).
						MockElectricityMeterReport("W", 0, errTest, false).
						MockElectricityMeterReport("kWh", 0, errTest, false),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "cable lock report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.cable_lock.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.state.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "current session report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "non existent thing on start charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.charge.start", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non existent thing on stop charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.charge.stop", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non existent thing on setting cable lock",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.cable_lock.set", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non existent thing on state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.state.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non existent thing on current session report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.current_session.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non existent thing on cable lock report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.cable_lock.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non-boolean value on setting cable lock",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.cable_lock.set", "chargepoint", "true"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "power",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "W"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec"),
						},
					},
					{
						Name:    "energy",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", "kWh"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec"),
						},
					},
					{
						Name:    "all electricity units",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec", ""),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec").AtLeastOnce(),
						},
					},
					{
						Name:    "all electricity units with null",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "cmd.meter.get_report", "meter_elec"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "meter_elec").AtLeastOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskCarCharger(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Car charger tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskCarCharger(
					mockedchargepoint.NewController(t).
						MockChargepointCableLockReport(true, nil, true).
						MockChargepointCableLockReport(false, errTest, true).
						MockChargepointCableLockReport(true, nil, true).
						MockChargepointCableLockReport(false, nil, true). // should be sent twice
						MockChargepointCurrentSessionReport(1.23, nil, true).
						MockChargepointCurrentSessionReport(0, errTest, true).
						MockChargepointCurrentSessionReport(1.23, nil, true).
						MockChargepointCurrentSessionReport(4.56, nil, true). // should be sent twice
						MockChargepointStateReport("ready_to_charge", nil, true).
						MockChargepointStateReport("", errTest, true).
						MockChargepointStateReport("ready_to_charge", nil, true).
						MockChargepointStateReport("charging", nil, true), // should be sent twice
					mockedmeterelec.NewReporter(t).
						MockElectricityMeterReport("W", 2, nil, true).
						MockElectricityMeterReport("W", 0, errors.New("test"), true).
						MockElectricityMeterReport("W", 2, nil, true).
						MockElectricityMeterReport("W", 1500, nil, false).
						MockElectricityMeterReport("kWh", 123.45, nil, true).
						MockElectricityMeterReport("kWh", 0, errors.New("test"), true).
						MockElectricityMeterReport("kWh", 123.45, nil, true).
						MockElectricityMeterReport("kWh", 123.56, nil, false),
					100*time.Millisecond,
				),
				Nodes: []*suite.Node{
					{
						Name: "One change and one error during three report cycles",
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.cable_lock.report", "chargepoint", true).ExactlyOnce(),
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.cable_lock.report", "chargepoint", false).ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.23).ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 4.56).ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "ready_to_charge").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.state.report", "chargepoint", "charging").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 2).ExpectProperty("unit", "W").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.45).ExpectProperty("unit", "kWh").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 1500).ExpectProperty("unit", "W").ExactlyOnce(),
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:meter_elec/ad:2", "evt.meter.report", "meter_elec", 123.56).ExpectProperty("unit", "kWh").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeCarCharger(
	chargepointController *mockedchargepoint.Controller,
	meterElecReporter *mockedmeterelec.Reporter,
	supportedChargingModes []string,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupCarCharger(t, mqtt, chargepointController, meterElecReporter, supportedChargingModes, 0)

		return routing, nil, mocks
	}
}

func taskCarCharger(
	chargepointController *mockedchargepoint.Controller,
	meterElecReporter *mockedmeterelec.Reporter,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupCarCharger(t, mqtt, chargepointController, meterElecReporter, nil, interval)

		return nil, tasks, mocks
	}
}

func setupCarCharger(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	chargepointController *mockedchargepoint.Controller,
	meterElecReporter *mockedmeterelec.Reporter,
	supportedChargingModes []string,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{chargepointController}

	cfg := &thing.CarChargerConfig{
		ThingConfig: &adapter.ThingConfig{
			InclusionReport: &fimptype.ThingInclusionReport{
				Address: "2",
			},
			Connector: mockedadapter.NewConnector(t),
		},
		ChargepointConfig: &chargepoint.Config{
			Specification: chargepoint.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				evChargerTestStates,
				supportedChargingModes,
			),
			Controller: chargepointController,
		},
	}

	if meterElecReporter != nil {
		cfg.MeterElecConfig = &meterelec.Config{
			Specification: meterelec.Specification(
				"test_adapter",
				"1",
				"2",
				nil,
				[]string{"W", "kWh"},
				nil,
			),
			Reporter: meterElecReporter,
		}

		mocks = append(mocks, meterElecReporter)
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(adapter adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return thing.NewCarCharger(publisher, thingState, cfg), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return thing.RouteCarCharger(ad), thing.TaskCarCharger(ad, duration), mocks
}
