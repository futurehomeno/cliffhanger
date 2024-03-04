package mediaplayer

import (
	"fmt"
	"slices"
	"sync"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	"github.com/mitchellh/mapstructure"

	"github.com/futurehomeno/cliffhanger/adapter"
	"github.com/futurehomeno/cliffhanger/adapter/cache"
)

const (
	// PropertySupportedPlayback is a property representing supported playback actions.
	PropertySupportedPlayback = "sup_playback"
	// PropertySupportedModes is a property representing supported playback modes.
	PropertySupportedModes = "sup_modes"
	// PropertySupportedMetadata is a property representing supported metadata.
	PropertySupportedMetadata = "sup_metadata"

	// PlaybackActionPlay is play action.
	PlaybackActionPlay PlaybackAction = "play"
	// PlaybackActionPause is pause action.
	PlaybackActionPause PlaybackAction = "pause"
	// PlaybackActionTogglePlayPause is toggle_play_pause action.
	PlaybackActionTogglePlayPause PlaybackAction = "toggle_play_pause"
	// PlaybackActionNextTrack is next_track action.
	PlaybackActionNextTrack PlaybackAction = "next_track"
	// PlaybackActionPreviousTrack is previous_track action.
	PlaybackActionPreviousTrack PlaybackAction = "previous_track"

	// ModeRepeat is for mode repeat.
	ModeRepeat Mode = "repeat"
	// ModeShuffle is for mode shuffle.
	ModeShuffle Mode = "shuffle"
	// ModeRepeatOne is for mode repeat_one.
	ModeRepeatOne Mode = "repeat_one"
	// ModeCrossFade is for mode cross fade.
	ModeCrossFade Mode = "crossfade"

	// MetadataAlbum is album metadata.
	MetadataAlbum Metadata = "album"
	// MetadataArtist is artist metadata.
	MetadataArtist Metadata = "artist"
	// MetadataTrack is track metadata.
	MetadataTrack Metadata = "track"
	// MetadataImageURL image_url metadata.
	MetadataImageURL Metadata = "image_url"
)

type (
	// Metadata represents a metadata.
	Metadata string
	// Mode represents a mode.
	Mode string
	// PlaybackAction represents a playback action.
	PlaybackAction string
)

// DefaultReportingStrategy is the default reporting strategy used by the service for periodic reports.
var DefaultReportingStrategy = cache.ReportOnChangeOnly()

// Controller is an interface representing an actual device.
type Controller interface {
	// SetPlayback sets the playback state.
	SetPlayback(action PlaybackAction) error
	// Playback returns the playback report.
	Playback() (PlaybackAction, error)
	// SetPlaybackMode sets the playback mode.
	SetPlaybackMode(mode map[string]bool) error
	// PlaybackMode returns the playback mode.
	PlaybackMode() (map[string]bool, error)
	// SetVolume sets the volume level.
	SetVolume(level int64) error
	// Volume returns the volume level.
	Volume() (int64, error)
	// SetMute sets the mute state.
	SetMute(mute bool) error
	// Mute returns the mute state.
	Mute() (bool, error)
	// Metadata returns the metadata.
	Metadata() (map[string]string, error)
}

// Service is an interface representing a media player FIMP service.
type Service interface {
	adapter.Service

	// SetPlayback sets the playback state.
	SetPlayback(action string) error
	// SendPlaybackReport sends a playback report. Returns true if a report was sent.
	SendPlaybackReport(force bool) (bool, error)
	// SetPlaybackMode sets the playback mode.
	SetPlaybackMode(mode map[string]bool) error
	// SendPlaybackModeReport sends a playback mode report. Returns true if a report was sent.
	SendPlaybackModeReport(force bool) (bool, error)
	// SetVolume sets the volume level.
	SetVolume(level int64) error
	// SendVolumeReport sends a volume report. Returns true if a report was sent.
	SendVolumeReport(force bool) (bool, error)
	// SetMute sets the mute state.
	SetMute(mute bool) error
	// SendMuteReport sends a mute report. Returns true if a report was sent.
	SendMuteReport(force bool) (bool, error)
	// SendMetadataReport sends a metadata report. Returns true if a report was sent.
	SendMetadataReport(force bool) (bool, error)
}

type (
	// Config represents a service configuration.
	Config struct {
		Specification     *fimptype.Service
		Controller        Controller
		ReportingStrategy cache.ReportingStrategy
	}
)

// NewService creates a new instance of a mediaplayer FIMP service.
func NewService(
	publisher adapter.ServicePublisher,
	cfg *Config,
) Service {
	cfg.Specification.EnsureInterfaces(requiredInterfaces()...)

	if cfg.ReportingStrategy == nil {
		cfg.ReportingStrategy = DefaultReportingStrategy
	}

	return &service{
		Service:           adapter.NewService(publisher, cfg.Specification),
		controller:        cfg.Controller,
		reportingStrategy: cfg.ReportingStrategy,

		reportingCache: cache.NewReportingCache(),
		lock:           &sync.Mutex{},
	}
}

// service is a private implementation of a mediaplayer FIMP service.
type service struct {
	adapter.Service

	controller        Controller
	lock              *sync.Mutex
	reportingCache    cache.ReportingCache
	reportingStrategy cache.ReportingStrategy
}

// SetPlayback sets the playback state.
func (s *service) SetPlayback(action string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.ValidatePlaybackAction(action); err != nil {
		return fmt.Errorf("%s: invalid playback action: %w", s.Name(), err)
	}

	err := s.controller.SetPlayback(PlaybackAction(action))
	if err != nil {
		return fmt.Errorf("%s: failed to set playback: %w", s.Name(), err)
	}

	return nil
}

// SendPlaybackReport sends a playback report. Returns true if a report was sent.
func (s *service) SendPlaybackReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	action, err := s.controller.Playback()
	if err != nil {
		return false, fmt.Errorf("%s: failed to get playback report: %w", s.Name(), err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtPlaybackReport, "", action) {
		return false, nil
	}

	message := fimpgo.NewStringMessage(EvtPlaybackReport, s.Name(), string(action), nil, nil, nil)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("%s: failed to send playback report: %w", s.Name(), err)
	}

	s.reportingCache.Reported(EvtPlaybackReport, "", action)

	return true, nil
}

// SetPlaybackMode sets the playback mode.
func (s *service) SetPlaybackMode(mode map[string]bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetPlaybackMode(mode)
	if err != nil {
		return fmt.Errorf("mediaplayer: failed to set playback mode: %w", err)
	}

	return nil
}

// SendPlaybackModeReport sends a playback mode report. Returns true if a report was sent.
func (s *service) SendPlaybackModeReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	mode, err := s.controller.PlaybackMode()
	if err != nil {
		return false, fmt.Errorf("mediaplayer: failed to get playback mode: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtPlaybackModeReport, "", mode) {
		return false, nil
	}

	var modeMap map[string]bool

	if err := mapstructure.Decode(mode, &modeMap); err != nil {
		return false, fmt.Errorf("mediaplayer: failed to decode playback mode: %w", err)
	}

	message := fimpgo.NewBoolMapMessage(EvtPlaybackModeReport, s.Name(), modeMap, nil, nil, nil)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("mediaplayer: failed to send playback mode report: %w", err)
	}

	s.reportingCache.Reported(EvtPlaybackModeReport, "", mode)

	return true, nil
}

// SetVolume sets the volume level.
func (s *service) SetVolume(level int64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetVolume(level)
	if err != nil {
		return fmt.Errorf("mediaplayer: failed to set volume: %w", err)
	}

	return nil
}

// SendVolumeReport sends a volume report. Returns true if a report was sent.
func (s *service) SendVolumeReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	level, err := s.controller.Volume()
	if err != nil {
		return false, fmt.Errorf("mediaplayer: failed to get volume: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtVolumeReport, "", level) {
		return false, nil
	}

	message := fimpgo.NewIntMessage(EvtVolumeReport, s.Name(), level, nil, nil, nil)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("mediaplayer: failed to send volume report: %w", err)
	}

	s.reportingCache.Reported(EvtVolumeReport, "", level)

	return true, nil
}

// SetMute sets the mute state.
func (s *service) SetMute(mute bool) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	err := s.controller.SetMute(mute)
	if err != nil {
		return fmt.Errorf("mediaplayer: failed to set mute: %w", err)
	}

	return nil
}

// SendMuteReport sends a mute report. Returns true if a report was sent.
func (s *service) SendMuteReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	mute, err := s.controller.Mute()
	if err != nil {
		return false, fmt.Errorf("mediaplayer: failed to get mute: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMuteReport, "", mute) {
		return false, nil
	}

	message := fimpgo.NewBoolMessage(EvtMuteReport, s.Name(), mute, nil, nil, nil)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("mediaplayer: failed to send mute report: %w", err)
	}

	s.reportingCache.Reported(EvtMuteReport, "", mute)

	return true, nil
}

// SendMetadataReport sends a metadata report. Returns true if a report was sent.
func (s *service) SendMetadataReport(force bool) (bool, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	metadata, err := s.controller.Metadata()
	if err != nil {
		return false, fmt.Errorf("mediaplayer: failed to get metadata: %w", err)
	}

	if !force && !s.reportingCache.ReportRequired(s.reportingStrategy, EvtMetadataReport, "", metadata) {
		return false, nil
	}

	message := fimpgo.NewObjectMessage(EvtMetadataReport, s.Name(), metadata, nil, nil, nil)

	err = s.SendMessage(message)
	if err != nil {
		return false, fmt.Errorf("mediaplayer: failed to send metadata report: %w", err)
	}

	s.reportingCache.Reported(EvtMetadataReport, "", metadata)

	return true, nil
}

// ValidatePlaybackAction validates the playback action.
func (s *service) ValidatePlaybackAction(action string) error {
	supportedActions := s.Specification().PropertyStrings(PropertySupportedPlayback)

	if !slices.Contains(supportedActions, action) {
		return fmt.Errorf("mediaplayer: unsupported playback action: %s", action)
	}

	return nil
}
