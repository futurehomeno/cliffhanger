package mediaplayer_test

import (
	"errors"
	"fmt"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"

	"github.com/futurehomeno/cliffhanger/adapter"
	mockedmediaplayer "github.com/futurehomeno/cliffhanger/test/mocks/adapter/service/mediaplayer"
	"github.com/futurehomeno/cliffhanger/test/suite"
)

func TestTaskReporting(t *testing.T) { //nolint:paralleltest
	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Media player tasks - success",
				Setup: routeServiceWithTasks(
					mockedmediaplayer.NewController(t).
						MockedMediaPlayerPlaybackReport("play", nil, true).
						MockedMediaPlayerPlaybackModeReport(samplePlaybackMode(), nil, true).
						MockedMediaPlayerVolumeReport(50, nil, true).
						MockedMediaPlayerMuteReport(false, nil, true).
						MockedMediaPlayerMetadataReport(sampleMetadata(), nil, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Should report playback state",
						Timeout: time.Second,
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playback.report", "media_player", "play"),
							suite.ExpectBoolMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playbackmode.report", "media_player", samplePlaybackMode()),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.volume.report", "media_player", 50),
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.mute.report", "media_player", false),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.metadata.report", "media_player", sampleMetadata()),
						},
					},
				},
			},
			{
				Name: "Media player tasks - failed",
				Setup: routeServiceWithTasks(
					mockedmediaplayer.NewController(t).
						MockedMediaPlayerPlaybackReport("play", errors.New("test error"), true).
						MockedMediaPlayerPlaybackModeReport(samplePlaybackMode(), errors.New("test error"), true).
						MockedMediaPlayerVolumeReport(50, errors.New("test error"), true).
						MockedMediaPlayerMuteReport(false, errors.New("test error"), true).
						MockedMediaPlayerMetadataReport(sampleMetadata(), errors.New("test error"), true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "Should report playback state",
						Timeout: time.Second,
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playback.report", "media_player", "play").
								Never(),
							suite.ExpectBoolMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playbackmode.report", "media_player", samplePlaybackMode()).
								Never(),
							suite.ExpectInt("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.volume.report", "media_player", 50).
								Never(),
							suite.ExpectBool("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.mute.report", "media_player", false).
								Never(),
							suite.ExpectStringMap("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.metadata.report", "media_player", sampleMetadata()).
								Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)
}

func TestTaskReportingSkipsNotReported(t *testing.T) { //nolint:paralleltest
	old := log.StandardLogger().ReplaceHooks(make(log.LevelHooks))
	defer log.StandardLogger().ReplaceHooks(old)

	hook := logtest.NewGlobal()

	notReported := fmt.Errorf("no state: %w", adapter.ErrNotReported)

	s := &suite.Suite{
		Cases: []*suite.Case{
			{
				Name: "Media player state not reported yet",
				Setup: routeServiceWithTasks(
					mockedmediaplayer.NewController(t).
						MockedMediaPlayerPlaybackReport("", notReported, true).
						MockedMediaPlayerPlaybackModeReport(nil, notReported, true).
						MockedMediaPlayerVolumeReport(0, notReported, true).
						MockedMediaPlayerMuteReport(false, notReported, true).
						MockedMediaPlayerMetadataReport(nil, notReported, true),
				),
				Nodes: []*suite.Node{
					{
						Name:    "no report is sent",
						Timeout: 300 * time.Millisecond,
						Expectations: []*suite.Expectation{
							suite.ExpectString("pt:j1/mt:evt/rt:dev/rn:test_adapter/ad:1/sv:media_player/ad:3", "evt.playback.report", "media_player", "").Never(),
						},
					},
				},
			},
		},
	}

	s.Run(t)

	for _, e := range hook.AllEntries() {
		assert.NotContains(t, e.Message, "[mediaplayer]")
	}
}
