package telemetry

import (
	"fmt"
	"slices"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"
)

// SyncRequester is the subset of fimpgo.SyncClient used by ConfigPull.
// Extracted as an interface for testability.
type SyncRequester interface {
	SendFimp(topic string, fimpMsg *fimpgo.FimpMessage, timeout int) (*fimpgo.FimpMessage, error)
	AddSubscription(topic string) error
	Stop()
}

// ConfigPull periodically pulls telemetry configuration from the cloud
// and applies it to the reporter. It implements root.Service (Start/Stop).
//
// Apps opt in by constructing a ConfigPull and passing it to WithServices()
// on the EdgeAppBuilder.
type ConfigPull struct {
	mqtt         *fimpgo.MqttTransport
	source       string
	reporter     Telemetry
	fallbackPoll time.Duration
	requestTopic string
	timeout      int

	lock    sync.Mutex
	timer   *time.Timer
	stopped bool
	client  SyncRequester
}

// ConfigPullOption configures optional parameters for NewConfigPull.
type ConfigPullOption func(*ConfigPull)

// WithFallbackPoll sets the fallback polling interval used when the cloud
// response does not include next_update or on error. Default: DefaultPollInterval.
func WithFallbackPoll(d time.Duration) ConfigPullOption {
	return func(cp *ConfigPull) {
		cp.fallbackPoll = d
	}
}

// WithRequestTopic overrides the MQTT topic for config requests.
// Default: ConfigRequestTopic.
func WithRequestTopic(topic string) ConfigPullOption {
	return func(cp *ConfigPull) {
		cp.requestTopic = topic
	}
}

// WithRequestTimeout sets the MQTT request timeout in seconds.
// Default: 30.
func WithRequestTimeout(seconds int) ConfigPullOption {
	return func(cp *ConfigPull) {
		cp.timeout = seconds
	}
}

// WithSyncRequester injects a SyncRequester for testing.
func WithSyncRequester(client SyncRequester) ConfigPullOption {
	return func(cp *ConfigPull) {
		cp.client = client
	}
}

// NewConfigPull creates a config pull service that periodically fetches
// telemetry config from the cloud and applies it to the reporter.
func NewConfigPull(mqtt *fimpgo.MqttTransport, source string, reporter Telemetry, opts ...ConfigPullOption) (*ConfigPull, error) {
	if mqtt == nil {
		return nil, fmt.Errorf("telemetry: config pull: mqtt transport is nil")
	}

	if source == "" {
		return nil, fmt.Errorf("telemetry: config pull: source is not set")
	}

	if reporter == nil {
		return nil, fmt.Errorf("telemetry: config pull: reporter is nil")
	}

	cp := &ConfigPull{
		mqtt:         mqtt,
		source:       source,
		reporter:     reporter,
		fallbackPoll: DefaultPollInterval,
		requestTopic: ConfigRequestTopic,
		timeout:      30,
	}

	for _, opt := range opts {
		opt(cp)
	}

	if cp.fallbackPoll <= 0 {
		return nil, fmt.Errorf("telemetry: config pull: fallback poll interval must be positive")
	}

	if cp.timeout <= 0 {
		return nil, fmt.Errorf("telemetry: config pull: request timeout must be positive")
	}

	return cp, nil
}

// Start begins the config pull loop. The first poll fires immediately
// and is non-blocking (runs in a background goroutine). Calling Start
// on an already-running service is a no-op.
func (cp *ConfigPull) Start() error {
	cp.lock.Lock()
	defer cp.lock.Unlock()

	if cp.timer != nil {
		return nil // already running
	}

	if cp.client == nil {
		cp.client = fimpgo.NewSyncClient(cp.mqtt)
	}

	responseTopic := fmt.Sprintf("pt:j1/mt:cmd/rt:app/rn:%s/ad:1", cp.source)

	if err := cp.client.AddSubscription(responseTopic); err != nil {
		return fmt.Errorf("telemetry: config pull: subscribe: %w", err)
	}

	cp.stopped = false
	cp.scheduleLocked(0)

	log.Infof("[cliff] Telemetry config pull started (source=%s)", cp.source)

	return nil
}

// Stop cancels any pending poll and stops the sync client.
func (cp *ConfigPull) Stop() error {
	cp.lock.Lock()
	defer cp.lock.Unlock()

	cp.stopped = true

	if cp.timer != nil {
		cp.timer.Stop()
		cp.timer = nil
	}

	if cp.client != nil {
		cp.client.Stop()
		cp.client = nil
	}

	log.Infof("[cliff] Telemetry config pull stopped (source=%s)", cp.source)

	return nil
}

// scheduleLocked schedules the next poll after the given delay.
// Must be called with cp.lock held.
func (cp *ConfigPull) scheduleLocked(delay time.Duration) {
	var t *time.Timer

	t = time.AfterFunc(delay, func() {
		cp.lock.Lock()

		if cp.stopped || cp.timer != t {
			cp.lock.Unlock()
			return
		}

		// Release lock during network I/O. If Stop() is called
		// concurrently, it cancels the SyncClient's transport which
		// causes SendFimp to return an error promptly.
		client := cp.client
		cp.lock.Unlock()

		nextDelay := cp.poll(client)

		cp.lock.Lock()
		defer cp.lock.Unlock()

		if cp.stopped {
			return
		}

		cp.scheduleLocked(nextDelay)
	})

	cp.timer = t
}

// poll sends a config request and applies the response. Returns the
// delay until the next poll.
func (cp *ConfigPull) poll(client SyncRequester) time.Duration {
	msg := fimpgo.NewNullMessage(CmdGetConfig, Service, nil, nil, nil)
	msg.ResponseToTopic = fmt.Sprintf("pt:j1/mt:cmd/rt:app/rn:%s/ad:1", cp.source)

	resp, err := client.SendFimp(cp.requestTopic, msg, cp.timeout)
	if err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry config pull failed, retrying in %s", cp.fallbackPoll)

		return cp.fallbackPoll
	}

	if resp.Interface != EvtConfigReport {
		log.Warnf("[cliff] Telemetry config pull: unexpected response type %q, retrying in %s", resp.Interface, cp.fallbackPoll)

		return cp.fallbackPoll
	}

	var cfg ConfigResponse
	if err := resp.GetObjectValue(&cfg); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry config pull: failed to parse response, retrying in %s", cp.fallbackPoll)

		return cp.fallbackPoll
	}

	// Enable and SetSuppressed are applied independently: a failure in
	// one should not prevent the other from being applied. The next poll
	// will reconcile any partial state.
	if err := cp.reporter.Enable(cfg.Enabled); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry config pull: failed to apply enabled=%v", cfg.Enabled)
	}

	suppressed := slices.Contains(cfg.Suppressed, cp.source)

	if err := cp.reporter.SetSuppressed(suppressed); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry config pull: failed to apply suppressed=%v", suppressed)
	}

	log.Infof("[cliff] Telemetry config applied (enabled=%v, suppressed=%v)", cfg.Enabled, suppressed)

	if cfg.NextUpdate == "" {
		return cp.fallbackPoll
	}

	nextUpdate, err := time.Parse(time.RFC3339, cfg.NextUpdate)
	if err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry config pull: failed to parse next_update %q", cfg.NextUpdate)

		return cp.fallbackPoll
	}

	delay := time.Until(nextUpdate)
	if delay <= 0 {
		return cp.fallbackPoll
	}

	return delay
}
