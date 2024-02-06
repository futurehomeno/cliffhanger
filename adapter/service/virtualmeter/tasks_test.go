package virtualmeter_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) {
	t.Parallel()

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "",
				TearDown: adapterhelper.TearDownAdapter(workdir),
				Setup:    routeService(time.Millisecond*50, time.Second),
				Nodes: []*suite.Node{
					{
						Name: "should report empty modes when nothing set",
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.meter.report",
								"virtual_meter_elec",
								map[string]float64{},
							),
						},
					},
					{
						Name: "Cmd meter add",
						Command: suite.NewMessageBuilder().
							FloatMapMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"cmd.meter.add",
								"virtual_meter_elec",
								map[string]float64{"on": 100, "off": 1},
							).
							AddProperty(virtualmeter.PropertyNameUnit, "W").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.meter.report",
								"virtual_meter_elec",
								map[string]float64{"on": 100, "off": 1},
							),
						},
					},
					{
						Name: "should report latest set modes",
						Command: suite.NewMessageBuilder().
							NullMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"cmd.meter.get_report",
								"virtual_meter_elec",
							).
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.meter.report",
								"virtual_meter_elec",
								map[string]float64{"on": 100, "off": 1},
							),
						},
					},
					{
						Name: "Cmd meter remove",
						Command: suite.NullMessage(
							"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
							"cmd.meter.remove",
							"virtual_meter_elec",
						),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.meter.report",
								"virtual_meter_elec",
								map[string]float64{},
							),
						},
					},
					{
						Name: "should report empty when modes removed",
						Command: suite.NewMessageBuilder().
							NullMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"cmd.meter.get_report",
								"virtual_meter_elec",
							).
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.meter.report",
								"virtual_meter_elec",
								map[string]float64{},
							),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
