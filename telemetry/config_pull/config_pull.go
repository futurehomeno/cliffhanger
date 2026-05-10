package config_pull

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

const (
	CmdGetConfig    = "cmd.telemetry.get_config"
	EvtConfigReport = "evt.telemetry.config_report"

	// ConfigRequestTopic is the MQTT topic for config requests to the cloud.
	// Uses mt:cmd/rt:cloud which matches the CloudBridge SDU/MDU
	// LocalToCloud default route (+/mt:cmd/rt:cloud/#).
	ConfigRequestTopic = "pt:j1/mt:cmd/rt:cloud/rn:telemetry/ad:config"

	// configResponseTopicFmt is the MQTT topic the cloud publishes config
	// responses to. Uses mt:evt/rt:cloud which matches the existing
	// CloudBridge CloudToLocal default route (+/mt:evt/rt:cloud/#).
	// The %s placeholder is the app's FIMP resource name (source).
	configResponseTopicFmt = "pt:j1/mt:evt/rt:cloud/rn:%s/ad:telemetry-config"

	// DefaultPollInterval is the fallback interval when the cloud response
	// does not include next_update or on error.
	DefaultPollInterval = 6 * time.Hour
	// MaxPollInterval caps the delay derived from the cloud next_update
	// field. Prevents a misconfigured response from silencing config
	// updates indefinitely.
	MaxPollInterval = 24 * time.Hour
)

//	configResponseT is the payload of evt.telemetry.config_report from the cloud.
//
// Suppressed lists domain names for which Report/Emit are dropped on this
// app; ReportRequired/EmitRequired still publish for those domains. NextUpdate
// is the RFC3339 timestamp the cloud expects the next poll at; the local
// scheduler clamps the derived delay with MaxPollInterval and falls back to
// fallbackPoll on parse error.
type configResponseT struct {
	Enabled           bool     `json:"enabled"`
	SuppressedDomains []string `json:"suppressed_domains"`
	NextUpdate        string   `json:"next_update"`
}

// SyncRequester is the subset of fimpgo.SyncClient used by the internal
// config pull. Exported so tests can inject a mock via WithSyncRequester.
type SyncRequester interface {
	SendFimp(topic string, fimpMsg *fimpgo.FimpMessage, timeout int) (*fimpgo.FimpMessage, error)
	AddSubscription(topic string) error
	Stop()
}

// Config periodically pulls telemetry configuration from the cloud and
// applies it to the owning telemetryT. It is constructed by New and lives
// for the lifetime of the Telemetry; apps do not interact with it directly.
type Config struct {
	mqtt         *fimpgo.MqttTransport
	sourceRn     fimptype.ResourceNameT
	fallbackPoll time.Duration
	requestTopic string
	timeout      int
	applyConfig  func(enabled bool, suppressed []string)

	lock    sync.Mutex
	timer   *time.Timer
	stopped bool
	client  SyncRequester
}

// newConfigPull seeds the Config with package defaults.
func New(mqtt *fimpgo.MqttTransport, sourceRn fimptype.ResourceNameT, applyConfig func(enabled bool, suppressed []string)) *Config {
	return &Config{
		mqtt:         mqtt,
		sourceRn:     sourceRn,
		fallbackPoll: DefaultPollInterval,
		requestTopic: ConfigRequestTopic,
		timeout:      30,
		applyConfig:  applyConfig,
	}
}

// Start begins the poll loop. The first poll is deferred by
// DefaultPollInterval to avoid hammering the cloud at app startup; Start
// itself is non-blocking.
func (ptr *Config) Start() error {
	if ptr.fallbackPoll <= 0 {
		return fmt.Errorf("telemetry: config pull: fallback poll interval must be positive")
	}

	if ptr.timeout <= 0 {
		return fmt.Errorf("telemetry: config pull: request timeout must be positive")
	}

	if ptr.requestTopic == "" {
		return fmt.Errorf("telemetry: config pull: request topic must not be empty")
	}

	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	if ptr.timer != nil {
		return nil // already running
	}

	ptr.stopped = false
	ptr.scheduleLocked(DefaultPollInterval)

	log.Infof("[cliff] Telemetry config pull scheduled (source=%s)", ptr.sourceRn)

	return nil
}

// ensureClientLocked lazily creates the SyncClient and subscribes to the
// response topic. Caller must hold ptr.lock.
func (ptr *Config) ensureClientLocked() error {
	if ptr.client != nil {
		return nil
	}

	client := fimpgo.NewSyncClient(ptr.mqtt)
	responseTopic := fmt.Sprintf(configResponseTopicFmt, ptr.sourceRn)

	if err := client.AddSubscription(responseTopic); err != nil {
		client.Stop()

		return fmt.Errorf("telemetry: config pull: subscribe: %w", err)
	}

	ptr.client = client

	return nil
}

// stop cancels any pending poll and stops an owned sync client.
func (ptr *Config) Stop() {
	ptr.lock.Lock()
	defer ptr.lock.Unlock()

	ptr.stopped = true

	if ptr.timer != nil {
		ptr.timer.Stop()
		ptr.timer = nil
	}

	if ptr.client != nil {
		ptr.client.Stop()
		ptr.client = nil
	}

	log.Infof("[cliff] Telemetry config pull stopped (source=%s)", ptr.sourceRn)
}

// pollResult holds the outcome of a single config poll: the next-poll delay
// derived from the cloud's NextUpdate (clamped to MaxPollInterval, falling
// back to fallbackPoll on error), and the parsed config (nil on failure).
type pollResult struct {
	delay time.Duration
	cfg   *configResponseT
}

// scheduleLocked schedules the next poll after the given delay.
// Must be called with ptr.lock held.
func (ptr *Config) scheduleLocked(delay time.Duration) {
	var t *time.Timer

	t = time.AfterFunc(delay, func() {
		ptr.lock.Lock()

		if ptr.stopped || ptr.timer != t {
			ptr.lock.Unlock()

			return
		}

		if err := ptr.ensureClientLocked(); err != nil {
			log.WithError(err).Warnf("[cliff] Telemetry config pull: not ready yet, retrying in %s", ptr.fallbackPoll)
			ptr.scheduleLocked(ptr.fallbackPoll)
			ptr.lock.Unlock()

			return
		}

		client := ptr.client
		ptr.lock.Unlock()

		result := ptr.poll(client)

		ptr.lock.Lock()
		defer ptr.lock.Unlock()

		if ptr.stopped || ptr.timer != t {
			return
		}

		if result.cfg != nil {
			ptr.applyConfig(result.cfg.Enabled, result.cfg.SuppressedDomains)
		}

		ptr.scheduleLocked(result.delay)
	})

	ptr.timer = t
}

// poll sends a config request, parses the response, and computes the next
// poll delay from the cloud's NextUpdate. Returns fallbackPoll + nil cfg on
// any error. Does not apply config - the caller decides based on stop/timer
// state.
func (ptr *Config) poll(client SyncRequester) pollResult {
	msg := fimpgo.NewNullMessage(CmdGetConfig, fimptype.ServiceNameT(ptr.sourceRn), nil, nil, nil)
	msg.Source = ptr.sourceRn
	msg.ResponseToTopic = fmt.Sprintf(configResponseTopicFmt, ptr.sourceRn)

	resp, err := client.SendFimp(ptr.requestTopic, msg, ptr.timeout)
	if err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry config pull failed, retrying in %s", ptr.fallbackPoll)

		return pollResult{delay: ptr.fallbackPoll}
	}

	if resp.Interface != EvtConfigReport {
		log.Warnf("[cliff] Telemetry config pull: unexpected response type %q, retrying in %s", resp.Interface, ptr.fallbackPoll)

		return pollResult{delay: ptr.fallbackPoll}
	}

	var cfg configResponseT
	if err := resp.GetObjectValue(&cfg); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry config pull: failed to parse response, retrying in %s", ptr.fallbackPoll)

		return pollResult{delay: ptr.fallbackPoll}
	}

	delay := ptr.fallbackPoll

	if cfg.NextUpdate != "" {
		nextUpdate, err := time.Parse(time.RFC3339, cfg.NextUpdate)
		if err != nil {
			log.WithError(err).Warnf("[cliff] Telemetry config pull: failed to parse next_update %q", cfg.NextUpdate)
		} else if d := time.Until(nextUpdate); d > 0 {
			delay = min(d, MaxPollInterval)
		}
	}

	return pollResult{delay: delay, cfg: &cfg}
}
