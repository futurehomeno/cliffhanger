package chargepoint_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedchargepoint "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/chargepoint"
	"github.com/futurehomeno/cliffhanger/test/suite"
	"github.com/futurehomeno/cliffhanger/types"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Car charger tasks",
				TearDown: adapterhelper.TearDownAdapter("../../testdata/adapter/test_adapter"),
				Setup: taskChargepoint(
					mockedchargepoint.NewMockedChargepoint(
						mockedchargepoint.NewController(t).
							MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 1.23}, nil, true).
							MockChargepointCurrentSessionReport(nil, errTest, true).
							MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 1.23}, nil, true).
							MockChargepointCurrentSessionReport(&chargepoint.SessionReport{SessionEnergy: 4.56}, nil, true). // should be sent twice
							MockChargepointStateReport("ready_to_charge", nil, true).
							MockChargepointStateReport("", errTest, true).
							MockChargepointStateReport("ready_to_charge", nil, true).
							MockChargepointStateReport("charging", nil, true), // should be sent twice
						mockedchargepoint.NewAdjustableMaxCurrentController(t).
							MockChargepointMaxCurrentReport(10, nil, true).
							MockChargepointMaxCurrentReport(0, errTest, true).
							MockChargepointMaxCurrentReport(10, nil, true).
							MockChargepointMaxCurrentReport(8, nil, true),
						nil,
						mockedchargepoint.NewAdjustablePhaseModeController(t).
							MockChargepointPhaseModeReport(types.PhaseModeNL1L2L3, nil, true).
							MockChargepointPhaseModeReport("", errTest, true).
							MockChargepointPhaseModeReport(types.PhaseModeNL1L2L3, nil, true).
							MockChargepointPhaseModeReport(types.PhaseModeNL1, nil, true),
						mockedchargepoint.NewAdjustableCableLockController(t).
							MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: false}, nil, true). // should be sent twice
							MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: true}, nil, true).
							MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: false}, errTest, true).
							MockChargepointCableLockReport(&chargepoint.CableReport{CableLock: true}, nil, true),
					),
					[]adapter.SpecificationOption{
						chargepoint.WithSupportedMaxCurrent(16),
						chargepoint.WithGridType(types.GridTypeTN),
						chargepoint.WithPhases(3),
						chargepoint.WithSupportedPhaseModes(types.PhaseModeNL1L2L3, types.PhaseModeNL1, types.PhaseModeNL2, types.PhaseModeNL3),
					},
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
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.max_current.report", "chargepoint", 10).ExactlyOnce(),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.max_current.report", "chargepoint", 8).ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.phase_mode.report", "chargepoint", "NL1L2L3").ExactlyOnce(),
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:chargepoint/ad:2", "evt.phase_mode.report", "chargepoint", "NL1").ExactlyOnce(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func taskChargepoint(
	chargepointController chargepoint.Controller,
	chargepointOptions []adapter.SpecificationOption,
	interval time.Duration,
) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		_, tasks, mocks := setupService(t, mqtt, chargepointController, chargepointOptions, interval)

		return nil, tasks, mocks
	}
}
