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

func TopicPatternAdapter(resourceName fimptype.ResourceNameT, msgType fimptype.MsgTypeT) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		MessageType:     msgType,
		ResourceType:    fimptype.ResourceTypeAdapter,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

func TopicPatternDevice(resourceName fimptype.ResourceNameT, msgType fimptype.MsgTypeT) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		MessageType:     msgType,
		ResourceType:    fimptype.ResourceTypeDevice,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

func TopicPatternDeviceService(serviceName fimptype.ServiceNameT, msgType fimptype.MsgTypeT) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		MessageType:  msgType,
		ResourceType: fimptype.ResourceTypeDevice,
		ServiceName:  serviceName,
	}).String()
}

func TopicPatternApplication(resourceName fimptype.ResourceNameT, msgType fimptype.MsgTypeT) string {
	return (&TopicPattern{
		PayloadType:     fimpgo.DefaultPayload,
		MessageType:     msgType,
		ResourceType:    fimptype.ResourceTypeApp,
		ResourceName:    resourceName,
		ResourceAddress: "1",
	}).String()
}

func TopicPatternRoomService(serviceName fimptype.ServiceNameT, msgType fimptype.MsgTypeT) string {
	return (&TopicPattern{
		PayloadType:  fimpgo.DefaultPayload,
		MessageType:  msgType,
		ResourceType: fimptype.ResourceTypeLocation,
		ResourceName: "room",
		ServiceName:  serviceName,
	}).String()
}

func CombineTopicPatterns(patterns ...[]string) []string {
	var combined []string

	for _, p := range patterns {
		combined = append(combined, p...)
	}

	return combined
}
