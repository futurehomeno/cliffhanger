package mediaplayer

import (
	"fmt"

	"github.com/futurehomeno/fimpgo"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/router"
)

// Constants defining routing service, commands and events.
const (
	CmdPlaybackSet       = "cmd.playback.set"
	CmdPlaybackGetReport = "cmd.playback.get_report"
	EvtPlaybackReport    = "evt.playback.report"

	CmdPlaybackModeSet       = "cmd.playbackmode.set"
	CmdPlaybackModeGetReport = "cmd.playbackmode.get_report"
	EvtPlaybackModeReport    = "evt.playbackmode.report"

	CmdVolumeSet       = "cmd.volume.set"
	CmdVolumeGetReport = "cmd.volume.get_report"
	EvtVolumeReport    = "evt.volume.report"

	CmdMuteSet       = "cmd.mute.set"
	CmdMuteGetReport = "cmd.mute.get_report"
	EvtMuteReport    = "evt.mute.report"

	CmdMetadataGetReport = "cmd.metadata.get_report"
	EvtMetadataReport    = "evt.metadata.report"

	MediaPlayer = "media_player"
)

// RouteService returns routing table for the service.
func RouteService(adapter adapter.ServiceRegistry) []*router.Routing {
	return []*router.Routing{
		routeCmdPlaybackSet(adapter),
		routeCmdPlaybackGetReport(adapter),
		routeCmdPlaybackModeSet(adapter),
		routeCmdPlaybackModeGetReport(adapter),
		routeCmdVolumeSet(adapter),
		routeCmdVolumeGetReport(adapter),
		routeCmdMuteSet(adapter),
		routeCmdMuteGetReport(adapter),
		routeCmdMetadataGetReport(adapter),
	}
}

// routeCmdPlaybackSet returns a routing responsible for handling the command.
func routeCmdPlaybackSet(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdPlaybackSet(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdPlaybackSet),
	)
}

// routeCmdPlaybackGetReport returns a routing responsible for handling the command.
func routeCmdPlaybackGetReport(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdPlaybackGetReport(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdPlaybackGetReport),
	)
}

// routeCmdPlaybackModeSet returns a routing responsible for handling the command.
func routeCmdPlaybackModeSet(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdPlaybackModeSet(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdPlaybackModeSet),
	)
}

// routeCmdPlaybackModeGetReport returns a routing responsible for handling the command.
func routeCmdPlaybackModeGetReport(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdPlaybackModeGetReport(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdPlaybackModeGetReport),
	)
}

// routeCmdVolumeSet returns a routing responsible for handling the command.
func routeCmdVolumeSet(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdVolumeSet(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdVolumeSet),
	)
}

// routeCmdVolumeGetReport returns a routing responsible for handling the command.
func routeCmdVolumeGetReport(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdVolumeGetReport(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdVolumeGetReport),
	)
}

// routeCmdMuteSet returns a routing responsible for handling the command.
func routeCmdMuteSet(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMuteSet(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdMuteSet),
	)
}

func routeCmdMuteGetReport(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMuteGetReport(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdMuteGetReport),
	)
}

func routeCmdMetadataGetReport(registry adapter.ServiceRegistry) *router.Routing {
	return router.NewRouting(
		handleCmdMetadataGetReport(registry),
		router.ForService(MediaPlayer),
		router.ForType(CmdMetadataGetReport),
	)
}

func handleCmdPlaybackSet(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetStringValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get value from the message: %w", err)
			}

			if err := mediaPlayer.SetPlayback(value); err != nil {
				return nil, fmt.Errorf("adapter: failed to set playback: %w", err)
			}

			_, err = mediaPlayer.SendPlaybackReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get playback report: %w", err)
			}

			_, err = mediaPlayer.SendMetadataReport(false)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get metadata report: %w", err)
			}

			return nil, nil
		}),
	)
}

func handleCmdPlaybackGetReport(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := mediaPlayer.SendPlaybackReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send playback report: %w", err)
			}

			return nil, nil
		}),
	)
}

//nolint:dupl
func handleCmdPlaybackModeSet(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetBoolMapValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get value from the message: %w", err)
			}

			if err := mediaPlayer.SetPlaybackMode(value); err != nil {
				return nil, fmt.Errorf("adapter: failed to set playback mode: %w", err)
			}

			_, err = mediaPlayer.SendPlaybackModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get playback mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

func handleCmdPlaybackModeGetReport(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := mediaPlayer.SendPlaybackModeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send playback mode report: %w", err)
			}

			return nil, nil
		}),
	)
}

//nolint:dupl
func handleCmdVolumeSet(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetIntValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get value from the message: %w", err)
			}

			if err := mediaPlayer.SetVolume(value); err != nil {
				return nil, fmt.Errorf("adapter: failed to set volume: %w", err)
			}

			_, err = mediaPlayer.SendVolumeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get volume report: %w", err)
			}

			return nil, nil
		}),
	)
}

func handleCmdVolumeGetReport(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := mediaPlayer.SendVolumeReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send volume report: %w", err)
			}

			return nil, nil
		}),
	)
}

//nolint:dupl
func handleCmdMuteSet(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			value, err := message.Payload.GetBoolValue()
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get value from the message: %w", err)
			}

			if err := mediaPlayer.SetMute(value); err != nil {
				return nil, fmt.Errorf("adapter: failed to set mute: %w", err)
			}

			_, err = mediaPlayer.SendMuteReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to get mute report: %w", err)
			}

			return nil, nil
		}),
	)
}

func handleCmdMuteGetReport(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := mediaPlayer.SendMuteReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send mute report: %w", err)
			}

			return nil, nil
		}),
	)
}

func handleCmdMetadataGetReport(registry adapter.ServiceRegistry) router.MessageHandler {
	return router.NewMessageHandler(
		router.MessageProcessorFn(func(message *fimpgo.Message) (*fimpgo.FimpMessage, error) {
			s := registry.ServiceByTopic(message.Topic)
			if s == nil {
				return nil, fmt.Errorf("adapter: service not found under the provided address: %s", message.Addr.ServiceAddress)
			}

			mediaPlayer, ok := s.(Service)
			if !ok {
				return nil, fmt.Errorf("adapter: incorrect service found under the provided address: %s", message.Addr.ServiceAddress)
			}

			_, err := mediaPlayer.SendMetadataReport(true)
			if err != nil {
				return nil, fmt.Errorf("adapter: failed to send metadata report: %w", err)
			}

			return nil, nil
		}),
	)
}
