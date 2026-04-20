package telemetry_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

const (
	testTopic         = "pt:j1/mt:rsp/rt:cloud/rn:backend-service/ad:telemetry"
	testServiceName   = "telemetry"
	testMessageType   = "evt.telemetry.report"
	testSource        = "core-energy-guard"
	appTopic          = "pt:j1/mt:cmd/rt:app/rn:" + testServiceName + "/ad:1"
	appReportTopic    = "pt:j1/mt:evt/rt:app/rn:" + testServiceName + "/ad:1"
	cmdSetEnabledType = "cmd.config.set_" + telemetry.SettingEnabled
	cmdGetEnabledType = "cmd.config.get_" + telemetry.SettingEnabled
	evtEnabledReport  = "evt.config." + telemetry.SettingEnabled + "_report"
)

func TestNew_RejectsInvalidInput(t *testing.T) {
	t.Parallel()

	t.Run("nil mqtt", func(t *testing.T) {
		t.Parallel()

		reporter, err := telemetry.New(nil, testSource)

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

					reporter, err = telemetry.New(mqtt, testSource)
					require.NoError(t, err)

					return telemetry.RoutingForReporter(testServiceName, reporter), nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Name: "Report publishes the event",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := reporter.Report("energy_limit_exceeded", "max_guard", map[string]any{
									"hourly_energy_limit": 15.0,
								})

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(testTopic),
								router.ForType(testMessageType),
								router.ForService(testServiceName),
								router.MessageVoterFn(func(msg *fimpgo.Message) bool {
									if string(msg.Payload.Source) != testSource {
										return false
									}

									return string(msg.Payload.ValueObj) == "energy_limit_exceeded"
								}),
							),
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
								router.ForType(testMessageType),
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
								router.ForTopic(testTopic),
								router.ForType(testMessageType),
							),
						},
					},
					{
						Name: "Enable(false) silences Report",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, reporter.Enable(false))
								assert.False(t, reporter.IsEnabled())

								err := reporter.Report("should_not_publish", "", nil)
								assert.NoError(t, err)
							},
						},
						// No Expectations: nothing should be published.
					},
					{
						Name: "Enable(true) restores Report",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								require.NoError(t, reporter.Enable(true))
								assert.True(t, reporter.IsEnabled())

								err := reporter.Report("after_enable", "", nil)
								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(
								router.ForTopic(testTopic),
								router.ForType(testMessageType),
								router.MessageVoterFn(func(msg *fimpgo.Message) bool {
									return string(msg.Payload.ValueObj) == "after_enable"
								}),
							),
						},
					},
					{
						Name:    "cmd.config.set_telemetry_enabled = false disables the reporter",
						Command: suite.BoolMessage(appTopic, cmdSetEnabledType, testServiceName, false),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtEnabledReport, testServiceName, false),
						},
					},
					{
						Name:    "cmd.config.get_telemetry_enabled reports current state",
						Command: suite.NullMessage(appTopic, cmdGetEnabledType, testServiceName),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtEnabledReport, testServiceName, false),
						},
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								assert.False(t, reporter.IsEnabled())
							},
						},
					},
					{
						Name:    "cmd.config.set_telemetry_enabled = true re-enables the reporter",
						Command: suite.BoolMessage(appTopic, cmdSetEnabledType, testServiceName, true),
						Expectations: []*suite.Expectation{
							suite.ExpectBool(appReportTopic, evtEnabledReport, testServiceName, true),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
