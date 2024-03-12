package virtualmeter_test

import (
	"testing"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/adapter/service/virtualmeter"
	"github.com/futurehomeno/cliffhanger/database"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedoutlvlswitch "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/outlvlswitch"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	workdir = "../../../testdata/adapter/test_adapter"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Happy paths",
				TearDown: adapterhelper.TearDownAdapter(workdir),
				Setup:    routeService(0, time.Second),
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
				},
			},
			{
				Name:     "Error paths",
				TearDown: adapterhelper.TearDownAdapter(workdir),
				Setup:    routeService(0, time.Second),
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

func routeService(duration, recalculatingPeriod time.Duration) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		return setupService(t, mqtt, duration, recalculatingPeriod)
	}
}

func setupService(
	t *testing.T,
	mqtt *fimpgo.MqttTransport,
	duration time.Duration,
	recalculatingPeriod time.Duration,
) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mocks := []suite.Mock{}
	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "2",
			Groups:  []string{"ch1"},
		},
		Connector: mockedadapter.NewConnector(t),
	}

	db, err := database.NewDatabase(workdir)
	assert.NoError(t, err, "should create database")

	managerWrapper := virtualmeter.NewManagerWrapper(db, recalculatingPeriod, time.Hour)
	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		outLvlSwitchSpec := outlvlswitch.Specification(
			"test_adapter",
			"1",
			"2",
			outlvlswitch.SwitchTypeOnAndOff,
			99,
			0,
			[]string{"ch1"},
		)
		outLvlSwitchService := outlvlswitch.NewService(
			publisher,
			&outlvlswitch.Config{
				Specification: outLvlSwitchSpec,
				Controller: mockedoutlvlswitch.NewController(t).
					MockLevelSwitchLevelReport(1, nil, false),
			},
		)

		thing := adapter.NewThing(publisher, thingState, thingCfg, outLvlSwitchService)

		if err := managerWrapper.RegisterThing(thing, publisher); err != nil {
			log.WithError(err).Errorf("virtual meter: failed to register service template. Thing addr: %s", thing.Address())
		}

		return thing, nil
	})

	ad := adapterhelper.PrepareAdapter(t, workdir, mqtt, factory)
	managerWrapper.WithAdapter(ad)
	adapterhelper.SeedAdapter(t, ad, []*adapter.ThingSeed{seed})

	reportingTask := virtualmeter.Tasks(ad, managerWrapper, duration, duration, duration)

	return virtualmeter.RouteService(ad), task.Combine(reportingTask), mocks
}
