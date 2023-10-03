package notification_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/notification"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestNotification(t *testing.T) { //nolint:paralleltest
	var service notification.Notification

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Notification",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
					t.Helper()

					service = notification.NewNotification(mqtt)

					return nil, nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Name: "Event",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := service.Event(&notification.Event{
									EventName: "test_event_name",
								})

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap("pt:j1/mt:evt/rt:app/rn:kind_owl/ad:1", "evt.notification.report", "kind_owl", map[string]string{
								"EventName":      "test_event_name",
								"MessageContent": "",
								"DeviceId":       "",
								"DeviceName":     "",
								"RoomId":         "",
								"RoomName":       "",
								"AreaId":         "",
								"AreaName":       "",
								"AreaType":       "",
							}),
						},
					},
					{
						Name: "Event with props",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := service.EventWithProps(&notification.Event{EventName: "test_event_name"}, map[string]string{"foo": "bar", "bar": "foo"})

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.NewExpectation(router.MessageVoterFn(func(msg *fimpgo.Message) bool {
								if msg.Payload.Properties["foo"] != "bar" || msg.Payload.Properties["bar"] != "foo" {
									return false
								}

								m, err := msg.Payload.GetStrMapValue()
								if err != nil {
									return false
								}

								if m["EventName"] != "test_event_name" {
									return false
								}

								return true
							})),
						},
					},
					{
						Name: "Message",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								t.Helper()

								err := service.Message("custom test message notification")

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap("pt:j1/mt:evt/rt:app/rn:kind_owl/ad:1", "evt.notification.report", "kind_owl", map[string]string{
								"EventName":      "custom",
								"MessageContent": "custom test message notification",
								"DeviceId":       "",
								"DeviceName":     "",
								"RoomId":         "",
								"RoomName":       "",
								"AreaId":         "",
								"AreaName":       "",
								"AreaType":       "",
							}),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
