package router

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"
)

type TopicPattern struct {
	PayloadType     string
	MessageType     string
	ResourceType    string
	ResourceName    string
	ResourceAddress string
	Service         string
	ServiceAddress  string
}

func (tp *TopicPattern) String() string {
	switch tp.ResourceType {
	case fimpgo.ResourceTypeDiscovery:
		return fmt.Sprintf("%s/%s/%s", tp.pt(), tp.mt(), tp.rt())
	case fimpgo.ResourceTypeAdapter, fimpgo.ResourceTypeApp, fimpgo.ResourceTypeCloud:
		return fmt.Sprintf("%s/%s/%s/%s/%s", tp.pt(), tp.mt(), tp.rt(), tp.rn(), tp.ra())
	case fimpgo.ResourceTypeDevice, fimpgo.ResourceTypeLocation:
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

	return "mt:" + tp.MessageType
}

func (tp *TopicPattern) rt() string {
	if tp.ResourceType == "" {
		return "+"
	}

	return "rt:" + tp.ResourceType
}

func (tp *TopicPattern) rn() string {
	if tp.ResourceName == "" {
		return "+"
	}

	return "rn:" + tp.ResourceName
}

func (tp *TopicPattern) ra() string {
	if tp.ResourceAddress == "" {
		return "+"
	}

	return "ad:" + tp.ResourceAddress
}

func (tp *TopicPattern) sv() string {
	if tp.Service == "" {
		return "+"
	}

	return "sv:" + tp.Service
}

func (tp *TopicPattern) sa() string {
	if tp.ServiceAddress == "" {
		return "+"
	}

	return "ad:" + tp.ServiceAddress
}

// TopicPatternAdapter returns a topic pattern for an adapter useful for subscriptions.
func TopicPatternAdapter(resourceName string) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		ResourceType:    fimpgo.ResourceTypeAdapter,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

// TopicPatternDevices returns a topic pattern for devices useful for subscriptions.
func TopicPatternDevices(resourceName string) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		ResourceType:    fimpgo.ResourceTypeDevice,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

// TopicPatternApplication returns a topic pattern for application useful for subscriptions.
func TopicPatternApplication(resourceName string) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		ResourceType:    fimpgo.ResourceTypeApp,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

// TopicPatternDeviceService returns a topic pattern for all device services of the provided type.
func TopicPatternDeviceService(serviceName string) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		ResourceType: fimpgo.ResourceTypeDevice,
		Service:      serviceName,
	}).String()
}

// TopicPatternDeviceServiceEvents returns a topic pattern for all device services of the provided type.
func TopicPatternDeviceServiceEvents(serviceName string) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		MessageType:  fimpgo.MsgTypeEvt,
		ResourceType: fimpgo.ResourceTypeDevice,
		Service:      serviceName,
	}).String()
}

// TopicPatternRoomService returns a topic pattern for all device services of the provided type.
func TopicPatternRoomService(serviceName string) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		ResourceType: fimpgo.ResourceTypeLocation,
		ResourceName: "room",
		Service:      serviceName,
	}).String()
}

// TopicPatternRoomServiceEvents returns a topic pattern for all device services of the provided type.
func TopicPatternRoomServiceEvents(serviceName string) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		MessageType:  fimpgo.MsgTypeEvt,
		ResourceType: fimpgo.ResourceTypeLocation,
		ResourceName: "room",
		Service:      serviceName,
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
