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

func TestTimeline(t *testing.T) {
	var service notification.Timeline

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Timeline",
				Setup: suite.BaseSetup(func(t *testing.T, mqtt *fimpgo.MqttTransport) (routing []*router.Routing, tasks []*task.Task, mocks []suite.Mock) {
					service = notification.NewTimeline(mqtt)

					return nil, nil, nil
				}),
				Nodes: []*suite.Node{
					{
						Name: "Event",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								err := service.Event(&notification.TimelineEvent{
									EventName: "test_event_name",
								})

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap("pt:j1/mt:cmd/rt:app/rn:time_owl/ad:1", "cmd.timeline.set", "kind_owl", map[string]string{
								"EventName":      "test_event_name",
								"MessageContent": "",
								"DeviceId":       "",
								"DeviceName":     "",
								"RoomName":       "",
								"AreaName":       "",
							}),
						},
					},
					{
						Name: "Message",
						Callbacks: []suite.Callback{
							func(t *testing.T) {
								err := service.Message(&notification.TimelineMessage{
									Sender:    "test_sender",
									MessageEN: "test_message_en",
									MessageNO: "test_message_no",
								})

								assert.NoError(t, err)
							},
						},
						Expectations: []*suite.Expectation{
							suite.ExpectStringMap("pt:j1/mt:cmd/rt:app/rn:time_owl/ad:1", "cmd.timeline.set", "time_owl", map[string]string{
								"sender":     "test_sender",
								"message_en": "test_message_en",
								"message_no": "test_message_no",
							}),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}
