package telemetry_test

import (
	"reflect"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestNew_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	t.Run("nil mqtt", func(t *testing.T) {
		t.Parallel()

		reporter, err := telemetry.New(nil, "core-energy-guard")

		require.Error(t, err)
		assert.Nil(t, reporter)
	})

	t.Run("empty source", func(t *testing.T) {
		t.Parallel()

		reporter, err := telemetry.New(&fimpgo.MqttTransport{}, "")

		require.Error(t, err)
		assert.Nil(t, reporter)
	})
}

func TestReporter(t *testing.T) { //nolint:paralleltest
	var reporter telemetry.Reporter

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Telemetry",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
					t.Helper()

					var err error

					reporter, err = telemetry.New(mqtt, "core-energy-guard")
					require.NoError(t, err)

					return nil, nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Name: "Report with domain and data",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := reporter.Report("energy_limit_exceeded", "max_guard", map[string]any{
									"hourly_energy_limit": 15.0,
									"energy_import":       16.2,
								})

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(telemetry.Topic),
								router.ForType(telemetry.MessageType),
								router.ForService(telemetry.Service),
								router.MessageVoterFn(func(msg *fimpgo.Message) bool {
									if string(msg.Payload.Source) != "core-energy-guard" {
										return false
									}

									var got telemetry.Event
									if err := msg.Payload.GetObjectValue(&got); err != nil {
										return false
									}

									want := telemetry.Event{
										Event:  "energy_limit_exceeded",
										Domain: "max_guard",
										Data: map[string]any{
											"hourly_energy_limit": 15.0,
											"energy_import":       16.2,
										},
									}

									return reflect.DeepEqual(got, want)
								}),
							),
						},
					},
					{
						Name: "ReportEvent",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := reporter.ReportEvent(&telemetry.Event{
									Event:  "user_login",
									Domain: "auth",
									Data: map[string]any{
										"charger_id":  "ZAP000123",
										"auth_method": "rfid",
									},
								})

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.ExpectObject(telemetry.Topic, telemetry.MessageType, telemetry.Service, &telemetry.Event{
								Event:  "user_login",
								Domain: "auth",
								Data: map[string]any{
									"charger_id":  "ZAP000123",
									"auth_method": "rfid",
								},
							}),
						},
					},
					{
						Name: "Report without domain or data omits them",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := reporter.Report("app_started", "", nil)

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(router.MessageVoterFn(func(msg *fimpgo.Message) bool {
								raw := msg.Payload.GetRawObjectValue()

								return string(raw) == `{"event":"app_started"}`
							})),
						},
					},
					{
						Name: "Report with empty event name returns error",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := reporter.Report("", "auth", map[string]any{"x": 1})

								assert.Error(t, err)
							},
						},
						// No Expectations: nothing should be published.
					},
					{
						Name: "ReportEvent with nil event returns error",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := reporter.ReportEvent(nil)

								assert.Error(t, err)
							},
						},
					},
					{
						Name: "SetTargetTopic redirects publishes",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								reporter.SetTargetTopic("pt:j1/mt:evt/rt:app/rn:custom/ad:1")

								err := reporter.Report("app_started", "", nil)
								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic("pt:j1/mt:evt/rt:app/rn:custom/ad:1"),
								router.ForType(telemetry.MessageType),
							),
						},
					},
					{
						Name: "SetTargetTopic with empty restores default",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								reporter.SetTargetTopic("")

								err := reporter.Report("app_started", "", nil)
								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(telemetry.Topic),
								router.ForType(telemetry.MessageType),
							),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
