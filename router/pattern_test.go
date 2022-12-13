package router_test

import (
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/router"
)

func TestTopicPattern(t *testing.T) {
	t.Parallel()

	tt := []struct {
		name    string
		pattern router.TopicPattern
		want    string
	}{
		{
			name:    "All default messages for test_resource adapter",
			pattern: router.TopicPattern{PayloadType: fimpgo.DefaultPayload, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: "test_resource", ResourceAddress: "1"},
			want:    "pt:j1/+/rt:ad/rn:test_resource/ad:1",
		},
		{
			name:    "All default messages for test_resource application",
			pattern: router.TopicPattern{PayloadType: fimpgo.DefaultPayload, ResourceType: fimpgo.ResourceTypeApp, ResourceName: "test_resource", ResourceAddress: "1"},
			want:    "pt:j1/+/rt:app/rn:test_resource/ad:1",
		},
		{
			name:    "All default messages for test_resource devices",
			pattern: router.TopicPattern{PayloadType: fimpgo.DefaultPayload, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: "test_resource", ResourceAddress: "1"},
			want:    "pt:j1/+/rt:dev/rn:test_resource/ad:1/+/+",
		},
		{
			name:    "All messages for room location",
			pattern: router.TopicPattern{PayloadType: fimpgo.DefaultPayload, ResourceType: fimpgo.ResourceTypeLocation, ResourceName: "room"},
			want:    "pt:j1/+/rt:loc/rn:room/+/+/+",
		},
		{
			name:    "All messages for device meter_elec service",
			pattern: router.TopicPattern{ResourceType: fimpgo.ResourceTypeDevice, Service: "meter_elec"},
			want:    "+/+/rt:dev/+/+/sv:meter_elec/+",
		},
		{
			name:    "All events for device meter_elec service at address 1",
			pattern: router.TopicPattern{MessageType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, Service: "meter_elec", ServiceAddress: "1"},
			want:    "+/mt:evt/rt:dev/+/+/sv:meter_elec/ad:1",
		},
		{
			name:    "All discovery events",
			pattern: router.TopicPattern{MessageType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDiscovery},
			want:    "+/mt:evt/rt:discovery",
		},
		{
			name:    "Empty pattern",
			pattern: router.TopicPattern{},
			want:    "+/+/+/+/+/+/+",
		},
	}

	for _, tc := range tt {
		tc := tc

		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.pattern.String()

			assert.Equal(t, tc.want, got)
		})
	}
}

func TestTopicPatternHelpers(t *testing.T) {
	t.Parallel()

	got := router.TopicPatternAdapter("test_resource")
	assert.Equal(t, "pt:j1/+/rt:ad/rn:test_resource/ad:1", got)

	got = router.TopicPatternDevices("test_resource")
	assert.Equal(t, "pt:j1/+/rt:dev/rn:test_resource/ad:1/+/+", got)

	got = router.TopicPatternApplication("test_resource")
	assert.Equal(t, "pt:j1/+/rt:app/rn:test_resource/ad:1", got)

	got = router.TopicPatternDeviceService("sensor_temp")
	assert.Equal(t, "pt:j1/+/rt:dev/+/+/sv:sensor_temp/+", got)

	got = router.TopicPatternDeviceServiceEvents("sensor_temp")
	assert.Equal(t, "pt:j1/mt:evt/rt:dev/+/+/sv:sensor_temp/+", got)

	got = router.TopicPatternRoomService("sensor_temp")
	assert.Equal(t, "pt:j1/+/rt:loc/rn:room/+/sv:sensor_temp/+", got)

	got = router.TopicPatternRoomServiceEvents("sensor_temp")
	assert.Equal(t, "pt:j1/mt:evt/rt:loc/rn:room/+/sv:sensor_temp/+", got)
}
