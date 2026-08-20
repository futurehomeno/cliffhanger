package router

import (
	"slices"
	"strings"

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
	segments := []string{
		segment("pt", tp.PayloadType),
		segment("mt", tp.MessageType.Str()),
		segment("rt", tp.ResourceType.Str()),
	}

	switch tp.ResourceType {
	case fimptype.ResourceTypeDiscovery:
	case fimptype.ResourceTypeAdapter, fimptype.ResourceTypeApp, fimptype.ResourceTypeCloud:
		segments = append(segments, segment("rn", tp.ResourceName.Str()), segment("ad", tp.ResourceAddress))
	default:
		segments = append(segments,
			segment("rn", tp.ResourceName.Str()),
			segment("ad", tp.ResourceAddress),
			segment("sv", tp.ServiceName.Str()),
			segment("ad", tp.ServiceAddress),
		)
	}

	return strings.Join(segments, "/")
}

// segment renders one topic segment, falling back to the single-level wildcard when unset.
func segment(prefix, value string) string {
	if value == "" {
		return "+"
	}

	return prefix + ":" + value
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
	return slices.Concat(patterns...)
}
