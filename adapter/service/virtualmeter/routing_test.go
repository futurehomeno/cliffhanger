package virtualmeter_test

import (
	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/numericmeter"
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	"github.com/futurehomeno/cliffhanger/test/suite"
	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
	"testing"
	"time"
)

const (
	workdir = "../../../testdata/adapter/test_adapter"
)

func TestRouteService(t *testing.T) {
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Happy paths",
				TearDown: adapterhelper.TearDownAdapter(workdir),
				Setup:    routeService(0),
				Nodes: []*suite.Node{
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
						Name: "Cmd get report after set",
						Command: suite.NullMessage(
							"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
							"cmd.meter.get_report",
							"virtual_meter_elec",
						),
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
						Name: "Cmd meter add idempotent",
						Command: suite.NewMessageBuilder().
							FloatMapMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"cmd.meter.add",
								"virtual_meter_elec",
								map[string]float64{"on": 123, "off": 321},
							).
							AddProperty(virtualmeter.PropertyNameUnit, "W").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectFloatMap(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.meter.report",
								"virtual_meter_elec",
								map[string]float64{"on": 123, "off": 321},
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
						Name: "Cmd meter remove idempotent",
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
						Name: "Cmd get report after remove",
						Command: suite.NullMessage(
							"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
							"cmd.meter.get_report",
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
						Name: "Cmd get reporting interval. Should return default value when not set",
						Command: suite.NullMessage(
							"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
							"cmd.config.get_interval",
							"virtual_meter_elec",
						),
						Expectations: []*suite.Expectation{
							suite.ExpectInt(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.config.interval_report",
								"virtual_meter_elec",
								30,
							),
						},
					},
					{
						Name: "Cmd set reporting interval",
						Command: suite.NewMessageBuilder().
							IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"cmd.config.set_interval",
								"virtual_meter_elec",
								13,
							).Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectInt(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.config.interval_report",
								"virtual_meter_elec",
								13,
							),
						},
					},
					{
						Name: "Cmd get reporting interval after set.",
						Command: suite.NullMessage(
							"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
							"cmd.config.get_interval",
							"virtual_meter_elec",
						),
						Expectations: []*suite.Expectation{
							suite.ExpectInt(
								"pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"evt.config.interval_report",
								"virtual_meter_elec",
								13,
							),
						},
					},
				},
			},
			{
				Name:     "Error paths",
				TearDown: adapterhelper.TearDownAdapter(workdir),
				Setup:    routeService(0),
				Nodes: []*suite.Node{
					{
						Name: "Error when unsupported unit property provided",
						Command: suite.NewMessageBuilder().
							FloatMapMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"cmd.meter.add",
								"virtual_meter_elec",
								map[string]float64{"on": 100},
							).
							AddProperty(virtualmeter.PropertyNameUnit, "invalid").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2", "virtual_meter_elec"),
						},
					},
					{
						Name: "Error when no unit property provided",
						Command: suite.NewMessageBuilder().
							FloatMapMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"cmd.meter.add",
								"virtual_meter_elec",
								map[string]float64{"on": 100},
							).
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2", "virtual_meter_elec"),
						},
					},
					{
						Name: "Error when value type isn't float map",
						Command: suite.NewMessageBuilder().
							StringMessage(
								"pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2",
								"cmd.meter.add",
								"virtual_meter_elec",
								"invalid",
							).
							AddProperty(virtualmeter.PropertyNameUnit, "W").
							Build(),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:virtual_meter_elec/ad:2", "virtual_meter_elec"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func routeService(duration time.Duration) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		return setupService(t, mqtt, duration)
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	duration time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	mocks := []suite.Mock{}
	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
		},
		Connector: mockedadapter.NewConnector(t),
	}

	vmeterManager := virtualmeter.NewVirtualMeterManager(workdir)

	virtualMeterConfig := &virtualmeter.Config{
		Specification: virtualmeter.Specification(
			"test_adapter",
			"1",
			"2",
			nil,
			[]string{"W"},
			[]string{"off", "on"},
		),
		VirtualMeterManager: vmeterManager,
	}

	numericMeterConfig := &numericmeter.Config{
		Specification: numericmeter.Specification(
			"meter_elec",
			"test_adapter",
			"1",
			"2",
			nil,
			[]numericmeter.Unit{numericmeter.UnitW},
			numericmeter.WithIsVirtual(),
		),
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {

		numericMeterService := numericmeter.NewService(publisher, numericMeterConfig)
		thing := adapter.NewThing(publisher, thingState, thingCfg, virtualmeter.NewService(publisher, virtualMeterConfig))

		if err := vmeterManager.RegisterDevice(thing, numericMeterService.Specification().Address, numericMeterService); err != nil {
			log.WithError(err).Errorf("virtual meter: failed to register service template. Thing addr: %s", thing.Address())
		}

		return thing, nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, workdir, mqtt, factory, adapter.ThingSeeds{seed})
	reportingTask := virtualmeter.TaskReporting(ad, duration)

	virtualmeter.WithAdapter(vmeterManager, ad)
	virtualmeter.WithTaskManager(vmeterManager, task.NewManager(reportingTask))

	return virtualmeter.RouteService(ad), task.Combine(reportingTask), mocks
}
