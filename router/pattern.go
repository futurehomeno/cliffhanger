package router

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
)

type TopicPattern struct {
	PayloadType     string
	MessageType     fimptype.MsgTypeT
	ResourceType    fimptype.ResourceTypeT
	ResourceName    fimptype.ResourceNameT
	ResourceAddress string
	ServiceName     fimptype.ServiceNameT
	ServiceAddress  string
}

func (tp *TopicPattern) String() string {
	switch tp.ResourceType {
	case fimptype.ResourceTypeDiscovery:
		return fmt.Sprintf("%s/%s/%s", tp.pt(), tp.mt(), tp.rt())
	case fimptype.ResourceTypeAdapter, fimptype.ResourceTypeApp, fimptype.ResourceTypeCloud:
		return fmt.Sprintf("%s/%s/%s/%s/%s", tp.pt(), tp.mt(), tp.rt(), tp.rn(), tp.ra())
	case fimptype.ResourceTypeDevice, fimptype.ResourceTypeLocation:
		fallthrough
	default:
		return fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s", tp.pt(), tp.mt(), tp.rt(), tp.rn(), tp.ra(), tp.sv(), tp.sa())
	}
}

func (tp *TopicPattern) pt() string {
	if tp.PayloadType == "" {
		return "+"
	}

	return "pt:" + tp.PayloadType
}

func (tp *TopicPattern) mt() string {
	if tp.MessageType == "" {
		return "+"
	}

	return "mt:" + tp.MessageType.Str()
}

func (tp *TopicPattern) rt() string {
	if tp.ResourceType == "" {
		return "+"
	}

	return "rt:" + tp.ResourceType.Str()
}

func (tp *TopicPattern) rn() string {
	if tp.ResourceName == "" {
		return "+"
	}

	return "rn:" + tp.ResourceName.Str()
}

func (tp *TopicPattern) ra() string {
	if tp.ResourceAddress == "" {
		return "+"
	}

	return "ad:" + tp.ResourceAddress
}

func (tp *TopicPattern) sv() string {
	if tp.ServiceName == "" {
		return "+"
	}

	return "sv:" + tp.ServiceName.Str()
}

func (tp *TopicPattern) sa() string {
	if tp.ServiceAddress == "" {
		return "+"
	}

	return "ad:" + tp.ServiceAddress
}

// TopicPatternAdapter returns a topic pattern for an adapter useful for subscriptions.
func TopicPatternAdapter(resourceName fimptype.ResourceNameT) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		ResourceType:    fimptype.ResourceTypeAdapter,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

// TopicPatternDevices returns a topic pattern for devices useful for subscriptions.
func TopicPatternDevices(resourceName fimptype.ResourceNameT) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		ResourceType:    fimptype.ResourceTypeDevice,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

// TopicPatternApplication returns a topic pattern for application useful for subscriptions.
func TopicPatternApplication(resourceName fimptype.ResourceNameT) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		ResourceType:    fimptype.ResourceTypeApp,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

// TopicPatternDeviceService returns a topic pattern for all device services of the provided type.
func TopicPatternDeviceService(serviceName fimptype.ServiceNameT) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		ResourceType: fimptype.ResourceTypeDevice,
		ServiceName:  serviceName,
	}).String()
}

// TopicPatternDeviceServiceEvents returns a topic pattern for all device services of the provided type.
func TopicPatternDeviceServiceEvents(serviceName fimptype.ServiceNameT) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		MessageType:  fimptype.MsgTypeEvt,
		ResourceType: fimptype.ResourceTypeDevice,
		ServiceName:  serviceName,
	}).String()
}

// TopicPatternRoomService returns a topic pattern for all device services of the provided type.
func TopicPatternRoomService(serviceName fimptype.ServiceNameT) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		ResourceType: fimptype.ResourceTypeLocation,
		ResourceName: "room",
		ServiceName:  serviceName,
	}).String()
}

// TopicPatternRoomServiceEvents returns a topic pattern for all device services of the provided type.
func TopicPatternRoomServiceEvents(serviceName fimptype.ServiceNameT) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		MessageType:  fimptype.MsgTypeEvt,
		ResourceType: fimptype.ResourceTypeLocation,
		ResourceName: "room",
		ServiceName:  serviceName,
	}).String()
}

// CombineTopicPatterns is a helper to easily combine multiple topic pattern slices into one slice.
func CombineTopicPatterns(patterns ...[]string) []string {
	var combined []string

	for _, p := range patterns {
		combined = append(combined, p...)
	}

	return combined
}
