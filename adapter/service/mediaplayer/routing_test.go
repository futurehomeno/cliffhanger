package mediaplayer_test

import (
	"errors"
	"testing"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/service/mediaplayer"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	adapterhelper "github.com/futurehomeno/cliffhanger/test/helper/adapter"
	mockedadapter "github.com/futurehomeno/cliffhanger/test/mocks/adapter"
	mockedmediaplayer "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/mediaplayer"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestRouteService(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name:     "Route media player tests",
				TearDown: adapterhelper.TearDownAdapter("../../../testdata/adapter/test_adapter"),
				Setup: routeService(
					mockedmediaplayer.NewController(t).
						MockedMediaPlayerPlaybackSet("play", nil, true).
						MockedMediaPlayerPlaybackReport("play", nil, true).
						MockedMediaPlayerPlaybackSet("play", errors.New("cannot set play"), true).
						MockedMediaPlayerPlaybackSet("play", nil, true).
						MockedMediaPlayerPlaybackReport("play", errors.New("cannot return report"), true).
						MockedMediaPlayerPlaybackReport("play", nil, true).
						MockedMediaPlayerPlaybackReport("play", errors.New("report error"), true).
						MockedMediaPlayerPlaybackModeSet(samplePlaybackModeMap(), nil, true).
						MockedMediaPlayerPlaybackModeReport(samplePlaybackModeMap(), nil, true).
						MockedMediaPlayerPlaybackModeSet(samplePlaybackModeMap(), errors.New("playback mode set error"), true).
						MockedMediaPlayerPlaybackModeSet(samplePlaybackModeMap(), nil, true).
						MockedMediaPlayerPlaybackModeReport(samplePlaybackModeMap(), errors.New("playback mode report error"), true).
						MockedMediaPlayerPlaybackModeReport(samplePlaybackModeMap(), nil, true).
						MockedMediaPlayerPlaybackModeReport(samplePlaybackModeMap(), errors.New("playback mode report error"), true).
						MockedMediaPlayerVolumeSet(10, nil, true).
						MockedMediaPlayerVolumeReport(10, nil, true).
						MockedMediaPlayerVolumeSet(10, errors.New("volume set error"), true).
						MockedMediaPlayerVolumeSet(10, nil, true).
						MockedMediaPlayerVolumeReport(10, errors.New("report error"), true).
						MockedMediaPlayerVolumeReport(10, nil, true).
						MockedMediaPlayerVolumeReport(10, errors.New("report error"), true).
						MockedMediaPlayerMuteSet(true, nil, true).
						MockedMediaPlayerMuteReport(true, nil, true).
						MockedMediaPlayerMuteSet(true, errors.New("mute set error"), true).
						MockedMediaPlayerMuteSet(true, nil, true).
						MockedMediaPlayerMuteReport(true, errors.New("report error"), true).
						MockedMediaPlayerMuteReport(true, nil, true).
						MockedMediaPlayerMuteReport(true, errors.New("report error"), true).
						MockedMediaPlayerMetadataReport(sampleMetadata(), nil, true).
						MockedMediaPlayerMetadataReport(sampleMetadata(), errors.New("metadata report error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Set playback success",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playback.set", "media_player", "play"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playback.report", "media_player", "play"),
						},
					},
					{
						Name:    "Set playback wrong action",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playback.set", "media_player", "not_supported_action"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "Set playback - wrong topic",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.playback.set", "media_player", "play"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "Set playback - wrong message type",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playback.set", "media_player", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "Set playback errored service",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playback.set", "media_player", "play"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "Set playback errored report",
						Command: suite.StringMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playback.set", "media_player", "play"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "get playback report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playback.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playback.report", "media_player", "play"),
						},
					},
					{
						Name:    "get playback report - wrong topic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.playback.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "get playback report - errored sending",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playback.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set playback mode - success",
						Command: suite.BoolMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playbackmode.set", "media_player", samplePlaybackModeMap()),
						Expectations: []*suite.Expectation{
							suite.ExpectBoolMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playbackmode.report", "media_player", samplePlaybackModeMap()),
						},
					},
					{
						Name:    "Set playback mode - wrong key",
						Command: suite.BoolMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playbackmode.set", "media_player", map[string]bool{"not_supported_key": true}),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set playback mode - wrong topic",
						Command: suite.BoolMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.playbackmode.set", "media_player", samplePlaybackModeMap()),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "set playback mode - wrong message type",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playbackmode.set", "media_player", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set playback mode - errored service",
						Command: suite.BoolMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playbackmode.set", "media_player", samplePlaybackModeMap()),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set playback mode - errored report",
						Command: suite.BoolMapMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playbackmode.set", "media_player", samplePlaybackModeMap()),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "get playback mode report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playbackmode.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectBoolMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playbackmode.report", "media_player", samplePlaybackModeMap()),
						},
					},
					{
						Name:    "get playback mode report - wrong topic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.playbackmode.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "get playback mode report - errored sending",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.playbackmode.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set volume - success",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.volume.set", "media_player", 10),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.volume.report", "media_player", 10),
						},
					},
					{
						Name:    "set volume - wrong topic",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.volume.set", "media_player", 10),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "set volume - wrong message type",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.volume.set", "media_player", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set volume - errored service",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.volume.set", "media_player", 10),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set volume - errored report",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.volume.set", "media_player", 10),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "get volume report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.volume.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.volume.report", "media_player", 10),
						},
					},
					{
						Name:    "get volume report - wrong topic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.volume.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "get volume report - errored sending",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.volume.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set mute - success",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.mute.set", "media_player", true),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.mute.report", "media_player", true),
						},
					},
					{
						Name:    "set mute - wrong topic",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.mute.set", "media_player", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "set mute - wrong message type",
						Command: suite.IntMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.mute.set", "media_player", 10),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set mute - errored service",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.mute.set", "media_player", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "set mute - errored report",
						Command: suite.BoolMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.mute.set", "media_player", true),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "get mute report",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.mute.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.mute.report", "media_player", true),
						},
					},
					{
						Name:    "get mute report - wrong topic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.mute.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "get mute report - errored service",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.mute.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
					{
						Name:    "get metadata report - success",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.metadata.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectObject("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.metadata.report", "media_player", sampleMetadata()),
						},
					},
					{
						Name:    "get metadata report - wrong topic",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "cmd.metadata.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:666", "media_player"),
						},
					},
					{
						Name:    "get metadata report - errored sending",
						Command: suite.NullMessage("pt:j1/mt:cmd/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "cmd.metadata.get_report", "media_player"),
						Expectations: []*suite.Expectation{
							suite.ExpectError("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "media_player"),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func supportedPlayback() []string {
	return []string{"play", "pause", "stop"}
}

func supportedModes() []string {
	return []string{"repeat", "shuffle", "crossfade", "repeat_one"}
}

func supportedMetadata() []string {
	return []string{"album", "track", "artist", "image_url"}
}

func sampleMetadata() map[string]string {
	return map[string]string{
		"album":    "the album",
		"track":    "a track",
		"artist":   "artist name",
		"imageURL": "http://the.image.url",
	}
}

func samplePlaybackMode() map[string]bool {
	return map[string]bool{"repeat": true, "shuffle": true}
}

func samplePlaybackModeMap() map[string]bool {
	return map[string]bool{
		"repeat":     true,
		"shuffle":    true,
		"crossfade":  false,
		"repeat_one": false,
	}
}

func routeService(controller mediaplayer.Controller, options ...adapter.SpecificationOption) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, _, mocks := setupService(t, mqtt, controller, options...)

		return routing, nil, mocks
	}
}

func routeServiceWithTasks(controller mediaplayer.Controller, options ...adapter.SpecificationOption) suite.BaseSetup {
	return func(t *testing.T, mqtt *fimpgo.MqttTransport) ([]*router.Routing, []*task.Task, []suite.Mock) {
		t.Helper()

		routing, tasks, mocks := setupService(t, mqtt, controller, options...)

		return routing, tasks, mocks
	}
}

func setupService(t *testing.T, mqtt *fimpgo.MqttTransport, controller mediaplayer.Controller, options ...adapter.SpecificationOption) ([]*router.Routing, []*task.Task, []suite.Mock) {
	t.Helper()

	mockedController, ok := controller.(suite.Mock)
	if !ok {
		t.Fatal("controller must be a mock")
	}

	mocks := []suite.Mock{mockedController}
	thingCfg := &adapter.ThingConfig{
		InclusionReport: &fimptype.ThingInclusionReport{
			Address: "3",
		},
		Connector: mockedadapter.NewDefaultConnector(t),
	}

	mediaplayerCfg := &mediaplayer.Config{
		Specification: mediaplayer.Specification(
			"test_adapter",
			"1",
			"3",
			nil,
			supportedPlayback(),
			supportedModes(),
			supportedMetadata(),
			options...,
		),
		Controller: controller,
	}

	seed := &adapter.ThingSeed{ID: "B", CustomAddress: "2"}

	factory := adapterhelper.FactoryHelper(func(a adapter.Adapter, publisher adapter.Publisher, thingState adapter.ThingState) (adapter.Thing, error) {
		return adapter.NewThing(
			publisher,
			thingState,
			thingCfg,
			mediaplayer.NewService(publisher, mediaplayerCfg),
		), nil
	})

	ad := adapterhelper.PrepareSeededAdapter(t, "../../testdata/adapter/test_adapter", mqtt, factory, adapter.ThingSeeds{seed})

	return mediaplayer.RouteService(ad), task.Combine(mediaplayer.TaskReporting(ad, 0)), mocks
}
