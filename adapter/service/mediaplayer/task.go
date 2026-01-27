package mediaplayer

import (
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/task"
)

// TaskReporting creates a reporting task.
func TaskReporting(serviceRegistry adapter.ServiceRegistry, frequency time.Duration, voters ...task.Voter) *task.Task {
	voters = append(voters, adapter.IsRegistryInitialized(serviceRegistry))

	return task.New(handleReporting(serviceRegistry), frequency, voters...)
}

// handleReporting creates handler of a reporting task.
func handleReporting(serviceRegistry adapter.ServiceRegistry) func() {
	return func() {
		for _, s := range serviceRegistry.Services(MediaPlayer) {
			mediaPlayer, ok := s.(Service)
			if !ok {
				continue
			}

			if adapter.ShouldSkipServiceTask(serviceRegistry, mediaPlayer) {
				continue
			}

			if _, err := mediaPlayer.SendPlaybackReport(false); err != nil {
				log.WithError(err).Errorf("failed to send playback report")
			}

			if _, err := mediaPlayer.SendPlaybackModeReport(false); err != nil {
				log.WithError(err).Errorf("failed to send playback mode report")
			}

			if _, err := mediaPlayer.SendVolumeReport(false); err != nil {
				log.WithError(err).Errorf("failed to send volume report")
			}

			if _, err := mediaPlayer.SendMuteReport(false); err != nil {
				log.WithError(err).Errorf("failed to send mute report")
			}

			if _, err := mediaPlayer.SendMetadataReport(false); err != nil {
				log.WithError(err).Errorf("failed to send metadata report")
			}
		}
	}
}
