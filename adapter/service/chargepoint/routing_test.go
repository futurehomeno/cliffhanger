package chargepoint_test

import (
	"errors"
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

var (
	evChargerChargingTestModes = []string{"normal", "slow"}
	errTest                    = errors.New("test")
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "successful start charging routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging(&chargepoint.ChargingSettings{}, nil, false).
						MockChargepointStateReport("charging", nil, false).
						MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 1.74}, nil, false),
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging(&chargepoint.ChargingSettings{Mode: "normal"}, nil, true).
						MockChargepointStateReport("charging", nil, true).
						MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 1.74}, nil, true),
					[]adapter.SpecificationOption{chargepoint.WithChargingModes(evChargerChargingTestModes...)},
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStopChargepointCharging(nil, true).
						MockChargepointStateReport("ready_to_charge", nil, true).
						MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 1.74}, nil, true),
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
				Setup: routeService(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t).
							MockSetChargepointCableLock(true, nil, true).
							MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: true}, nil, false).
							MockChargepointStateReport("charging", nil, true).
							MockChargepointCurrentSessionReport(&chargepoint.SessionReport{
								SessionEnergy:         1.74,
								PreviousSessionEnergy: 4.5,
								StartedAt:             time.Date(2023, 9, 1, 12, 0, 0, 0, time.UTC),
								FinishedAt:            time.Date(2023, 8, 31, 2, 0, 0, 0, time.UTC),
								OfferedCurrent:        10,
							}, nil, false),
						mockedchargepoint.NewAdjustableCurrentController(t).
							MockSetChargepointOfferedCurrent(10, nil, true).
							MockSetChargepointMaxCurrent(16, nil, true).
							MockChargepointMaxCurrentReport(16, nil, false),
						mockedchargepoint.NewAdjustablePhaseModeController(t).
							MockSetChargepointPhaseMode(chargepoint.PhaseModeNL1L2L3, nil, true).
							MockChargepointPhaseModeReport(chargepoint.PhaseModeNL1L2L3, nil, false),
					),
					[]adapter.SpecificationOption{
						chargepoint.WithSupportedMaxCurrent(16),
						chargepoint.WithSupportedPhaseModes(chargepoint.PhaseModeNL1L2L3, chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3),
					},
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
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.74).
								ExpectProperty("offered_current", "10").
								ExpectProperty("previous_session", "4.50"),
						},
					},
					{
						Name:    "set offered current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.set_current", "chargepoint", 10),
						Expectations: []*suite.Expectation{
							suite.ExpectFloat("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.current_session.report", "chargepoint", 1.74).
								ExpectProperty("offered_current", "10").
								ExpectProperty("previous_session", "4.50"),
						},
					},
					{
						Name:    "set max current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.set", "chargepoint", 16),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.max_current.report", "chargepoint", 16),
						},
					},
					{
						Name:    "get max current",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.max_current.report", "chargepoint", 16),
						},
					},
					{
						Name:    "set phase mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.set", "chargepoint", "NL1L2L3"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.phase_mode.report", "chargepoint", "NL1L2L3"),
						},
					},
					{
						Name:    "get phase mode",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.phase_mode.report", "chargepoint", "NL1L2L3"),
						},
					},
				},
			},
			{
				Name:     "failed start charging routing - starting error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging(&chargepoint.ChargingSettings{}, errTest, true),
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging(&chargepoint.ChargingSettings{}, nil, true).
						MockChargepointStateReport("", errTest, true),
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStartChargepointCharging(&chargepoint.ChargingSettings{}, nil, true).
						MockChargepointStateReport("ready_to_charge", nil, true).
						MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 0}, errTest, true),
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
				Setup: routeService(
					mockedchargepoint.NewController(t),
					[]adapter.SpecificationOption{chargepoint.WithChargingModes(evChargerChargingTestModes...)},
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStopChargepointCharging(errTest, true),
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStopChargepointCharging(nil, true).
						MockChargepointStateReport("", errTest, true),
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockStopChargepointCharging(nil, true).
						MockChargepointStateReport("ready_to_charge", nil, true).
						MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 0}, errTest, true),
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockSetChargepointCableLock(true, errTest, true),
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
				Setup: routeService(
					mockedchargepoint.NewController(t).
						MockSetChargepointCableLock(true, nil, true).
						MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: false}, errTest, true),
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
				Name:     "failed set offered current routing - current session report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t).
							MockChargepointCurrentSessionReport(nil, errTest, false),
						mockedchargepoint.NewAdjustableCurrentController(t).
							MockSetChargepointOfferedCurrent(10, nil, false),
						nil,
					),
					[]adapter.SpecificationOption{chargepoint.WithSupportedMaxCurrent(16)},
				),
				Nodes: []*suite.Node{
					{
						Name:    "set offered current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.set_current", "chargepoint", 10),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "failed set max current routing - max current report error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t),
						mockedchargepoint.NewAdjustableCurrentController(t).
							MockChargepointMaxCurrentReport(0, errTest, false).
							MockSetChargepointMaxCurrent(14, nil, false),
						nil,
					),
					[]adapter.SpecificationOption{chargepoint.WithSupportedMaxCurrent(16)},
				),
				Nodes: []*suite.Node{
					{
						Name:    "set max current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.set", "chargepoint", 14),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "no adjustable current controller - missing max current",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedchargepoint.NewController(t), nil),
				Nodes: []*suite.Node{
					{
						Name:    "set offered current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.set_current", "chargepoint", 14),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "set max current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.set", "chargepoint", 14),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "get max current",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "no adjustable current controller - missing implementation",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedchargepoint.NewController(t), []adapter.SpecificationOption{chargepoint.WithSupportedMaxCurrent(16)}),
				Nodes: []*suite.Node{
					{
						Name:    "set offered current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.set_current", "chargepoint", 14),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "set max current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.set", "chargepoint", 14),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "get max current",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "adjustable phase mode controller - unsupported phase mode on set",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t),
						nil,
						mockedchargepoint.NewAdjustablePhaseModeController(t),
					),
					[]adapter.SpecificationOption{chargepoint.WithSupportedPhaseModes(chargepoint.PhaseModeNL1)},
				),
				Nodes: []*suite.Node{
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.set", "chargepoint", "L1L2"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "adjustable phase mode controller - set error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t),
						nil,
						mockedchargepoint.NewAdjustablePhaseModeController(t).
							MockSetChargepointPhaseMode(chargepoint.PhaseModeNL1, errTest, true),
					),
					[]adapter.SpecificationOption{chargepoint.WithSupportedPhaseModes(chargepoint.PhaseModeNL1L2L3, chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3)},
				),
				Nodes: []*suite.Node{
					{
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.set", "chargepoint", "NL1"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "adjustable phase mode controller - get error",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t),
						nil,
						mockedchargepoint.NewAdjustablePhaseModeController(t).
							MockChargepointPhaseModeReport(chargepoint.PhaseModeNL1, errTest, true),
					),
					[]adapter.SpecificationOption{chargepoint.WithSupportedPhaseModes(chargepoint.PhaseModeNL1L2L3, chargepoint.PhaseModeNL1, chargepoint.PhaseModeNL2, chargepoint.PhaseModeNL3)},
				),
				Nodes: []*suite.Node{
					{
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "no adjustable phase mode controller - missing implementation",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup:    routeService(mockedchargepoint.NewController(t), nil),
				Nodes: []*suite.Node{
					{
						Name:    "set phase mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.set", "chargepoint", "NL1"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "get phase mode",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "no adjustable phase mode controller - implementation exists, but supported phase modes are not provided",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t),
						nil,
						mockedchargepoint.NewAdjustablePhaseModeController(t),
					),
					nil,
				),
				Nodes: []*suite.Node{
					{
						Name:    "set phase mode",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.set", "chargepoint", "NL1"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "get phase mode",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.phase_mode.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
			{
				Name:     "other failed routing",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t).
							MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: false}, errTest, false).
							MockChargepointStateReport("", errTest, true).
							MockChargepointCurrentSessionReport(nil, errTest, true),
						mockedchargepoint.NewAdjustableCurrentController(t).
							MockChargepointMaxCurrentReport(0, errTest, true).
							MockSetChargepointOfferedCurrent(10, errTest, true).
							MockSetChargepointMaxCurrent(14, errTest, true),
						nil,
					),
					[]adapter.SpecificationOption{chargepoint.WithSupportedMaxCurrent(16)},
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
						Name:    "max current report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "set max current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.set", "chargepoint", 14),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "set offered current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.set_current", "chargepoint", 10),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "non-existent thing on start charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.charge.start", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non-existent thing on stop charging",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.charge.stop", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non-existent thing on setting cable lock",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.cable_lock.set", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non-existent thing on state report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.state.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non-existent thing on current session report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.current_session.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non-existent thing on cable lock report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.cable_lock.get_report", "chargepoint"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "chargepoint"),
						},
					},
					{
						Name:    "non-existent thing on max current report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:3", "cmd.max_current.get_report", "chargepoint"),
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
						Name:    "non-integer value on setting offered current",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.set_current", "chargepoint", "10"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "non-integer value on setting max current",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.set", "chargepoint", "10"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "too-high value on setting offered current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.set_current", "chargepoint", 32),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "too-small value on setting offered current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.current_session.set_current", "chargepoint", 5),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "too-high value on setting max current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.set", "chargepoint", 32),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
					{
						Name:    "too-small value on setting max current",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "cmd.max_current.set", "chargepoint", 5),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "chargepoint"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(
	controller chargepoint.Controller,
	options []adapter.SpecificationOption,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller, options, 0)

		return routing, nil, mocks
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	controller chargepoint.Controller,
	options []adapter.SpecificationOption,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mockedController, ok := controller.(suite.Mock)
	if !ok {
		t.Fatal("controller is not a mock")
	}

	mocks := []suite.Mock{mockedController}

	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	chargepointCfg := &chargepoint.Config{
		Specification: chargepoint.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			[]chargepoint.State{"ready_to_charge", "charging", "error"},
			options...,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(publisher, thingState, thingCfg, chargepoint.NewService(publisher, chargepointCfg)), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return chargepoint.RouteService(ad), task.Combine(chargepoint.TaskReporting(ad, duration)), mocks
}
