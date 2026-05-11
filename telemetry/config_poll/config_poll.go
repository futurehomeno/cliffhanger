package config_poll

import (
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	"github.com/futurehomeno/cliffhanger/backoff"
	"github.com/futurehomeno/cliffhanger/telemetry/types"
)

const (
	CmdGetConfig    = "cmd.telemetry.get_config"
	EvtConfigReport = "evt.telemetry.config_report"

	// ConfigRequestTopic is the MQTT topic for config requests to the cloud.
	ConfigRequestTopic = "pt:j1/mt:cmd/rt:cloud/rn:telemetry/ad:config"

	// ConfigResponseTopic is the shared broadcast topic for config responses.
	// All apps subscribe permanently; one poll response benefits all.
	ConfigResponseTopic = "pt:j1/mt:evt/rt:cloud/rn:telemetry/ad:config"

	DefaultPollInterval = 6 * time.Hour
	// AdditionalRandomPollIntervalRange is added as random jitter to scheduled
	// polls to spread simultaneous timers across apps sharing the same topic.
	// Also used as the freshness window: skip a poll if config arrived within
	// this duration (another app already polled).
	AdditionalRandomPollIntervalRange = 30 * time.Minute
	MaxPollInterval                   = 24 * time.Hour

	channelName = "telemetry-config-poll"
)

func subscribeBackoff() backoff.Stateful {
	return backoff.NewStateful(time.Minute, 10*time.Minute, 10*time.Minute, 1, 0)
}

type configResponseT struct {
	Enabled    bool                             `json:"enabled"`
	Suppressed map[string]types.SuppressedEntry `json:"suppressed"`
	NextUpdate string                           `json:"next_update"`
}

// Config polls and listens for telemetry configuration from the cloud.
type Config struct {
	mqtt         *fimpgo.MqttTransport
	sourceRn     fimptype.ResourceNameT
	fallbackPoll time.Duration
	requestTopic string
	applyConfig  func(enabled bool, suppressed map[string]types.SuppressedEntry)

	lock           sync.Mutex
	timer          *time.Timer
	stopped        bool
	msgCh          fimpgo.MessageCh
	stopCh         chan struct{}
	lastReceivedAt time.Time
}

func New(mqtt *fimpgo.MqttTransport, sourceRn fimptype.ResourceNameT, applyConfig func(enabled bool, suppressed map[string]types.SuppressedEntry)) *Config {
	return &Config{
		mqtt:         mqtt,
		sourceRn:     sourceRn,
		fallbackPoll: DefaultPollInterval,
		requestTopic: ConfigRequestTopic,
		applyConfig:  applyConfig,
	}
}

func (ptr *Config) Start() error {
	if ptr.fallbackPoll <= 0 {
		return fmt.Errorf("telemetry: config poll: fallback poll interval must be positive")
	}

	if ptr.requestTopic == "" {
		return fmt.Errorf("telemetry: config poll: request topic must not be empty")
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if ptr.timer != nil {
		return nil
	}

	ptr.stopped = false
	ptr.stopCh = make(chan struct{})
	ptr.msgCh = make(fimpgo.MessageCh, 8)
	stopCh := ptr.stopCh

	ptr.mqtt.RegisterChannelWithFilter(channelName, ptr.msgCh, fimpgo.FimpFilter{
		Topic:     ConfigResponseTopic,
		Interface: EvtConfigReport,
		Service:   "*",
	})

	subscribed := true
	if err := ptr.mqtt.Subscribe(ConfigResponseTopic); err != nil {
		subscribed = false
		log.Warnf("[cliff] Subscribe telemetry config topic err: %v", err)
	}

	go ptr.listen(stopCh, subscribed)

	ptr.scheduleLocked(DefaultPollInterval)

	return nil
}

func (ptr *Config) listen(stopCh <-chan struct{}, subscribed bool) {
	defer ptr.mqtt.UnregisterChannel(channelName)

	if !subscribed && !ptr.ensureSubscribed(stopCh) {
		return
	}

	for {
		select {
		case <-stopCh:
			return

		case msg, ok := <-ptr.msgCh:
			if !ok {
				return
			}

			if msg == nil || msg.Payload == nil {
				continue
			}

			if msg.Topic != ConfigResponseTopic {
				continue
			}

			switch msg.Payload.Interface {
			case EvtConfigReport:
				ptr.handleConfigReport(msg.Payload)
			default:
				continue
			}
		}
	}
}

func (ptr *Config) ensureSubscribed(stopCh <-chan struct{}) bool {
	bo := subscribeBackoff()

	for {
		if err := ptr.mqtt.Subscribe(ConfigResponseTopic); err == nil {
			log.Debug("[cliff] Telemetry config poll started")
			return true
		} else {
			delay := bo.Next()
			log.Warnf("[cliff] Telemetry config subscribe err: %v", err)

			select {
			case <-stopCh:
				return false
			case <-time.After(delay):
			}
		}
	}
}

func (ptr *Config) handleConfigReport(payload *fimpgo.FimpMessage) {
	var cfg configResponseT
	if err := payload.GetObjectValue(&cfg); err != nil {
		log.Warnf("[cliff] Telemetry config config parse err: %v", err)

		return
	}

	ptr.applyConfig(cfg.Enabled, cfg.Suppressed)

	delay := ptr.nextUpdate(cfg.NextUpdate)

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	ptr.lastReceivedAt = time.Now()

	if !ptr.stopped {
		ptr.scheduleLocked(delay)
	}
}

func (ptr *Config) nextUpdate(at string) time.Duration {
	jitter := time.Duration(rand.Int64N(int64(AdditionalRandomPollIntervalRange))) //nolint:gosec // non-cryptographic jitter

	if at == "" {
		return ptr.fallbackPoll + jitter
	}

	t, err := time.Parse(time.RFC3339, at)
	if err != nil {
		log.Warnf("[cliff] Parse next_update %q err: %v", at, err)
		return ptr.fallbackPoll + jitter
	}

	if d := time.Until(t); d > 0 {
		return min(d, MaxPollInterval) + jitter
	}

	return ptr.fallbackPoll + jitter
}

func (ptr *Config) Stop() {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	ptr.stopped = true

	if ptr.stopCh != nil {
		close(ptr.stopCh)
		ptr.stopCh = nil
	}

	if ptr.timer != nil {
		ptr.timer.Stop()
		ptr.timer = nil
	}

	log.Info("[cliff] Telemetry config poll stopped")
}

func (ptr *Config) scheduleLocked(delay time.Duration) {
	if ptr.timer != nil {
		ptr.timer.Stop()
	}

	var t *time.Timer

	t = time.AfterFunc(delay, func() {
		ptr.lock.Lock()

		if ptr.stopped || ptr.timer != t {
			ptr.lock.Unlock()

			return
		}

		// Skip if config is fresh — another app already polled.
		if !ptr.lastReceivedAt.IsZero() && time.Since(ptr.lastReceivedAt) < AdditionalRandomPollIntervalRange {
			ptr.scheduleLocked(ptr.fallbackPoll)
			ptr.lock.Unlock()

			return
		}

		ptr.lock.Unlock()

		ptr.sendGetConfigCmd()
	})

	ptr.timer = t
}

func (ptr *Config) sendGetConfigCmd() {
	msg := fimpgo.NewNullMessage(CmdGetConfig, fimptype.ServiceNameT(ptr.sourceRn), nil, nil, nil)
	msg.Source = ptr.sourceRn
	msg.ResponseToTopic = ConfigResponseTopic

	if err := ptr.mqtt.PublishToTopic(ptr.requestTopic, msg); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry config poll: send request failed, retrying in %s", ptr.fallbackPoll)
	}

	// Always schedule a fallback retry; handleConfigReport reschedules sooner
	// if a response arrives, so polling never stops on lost responses.
	ptr.lock.Lock()
	if !ptr.stopped {
		ptr.scheduleLocked(ptr.fallbackPoll)
	}
	ptr.lock.Unlock()
}
