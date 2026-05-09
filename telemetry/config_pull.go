package telemetry

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"
)

// SyncRequester is the subset of fimpgo.SyncClient used by the internal
// config pull. Exported so tests can inject a mock via WithSyncRequester.
type SyncRequester interface {
	SendFimp(topic string, fimpMsg *fimpgo.FimpMessage, timeout int) (*fimpgo.FimpMessage, error)
	AddSubscription(topic string) error
	Stop()
}

// configPull periodically pulls telemetry configuration from the cloud and
// applies it to the owning telemetryT. It is constructed by New and lives
// for the lifetime of the Telemetry; apps do not interact with it directly.
type configPull struct {
	mqtt         *fimpgo.MqttTransport
	sourceRn     fimptype.ResourceNameT
	telemetry    Telemetry
	fallbackPoll time.Duration
	requestTopic string
	timeout      int

	lock    sync.Mutex
	timer   *time.Timer
	stopped bool
	client  SyncRequester
}

// newConfigPull seeds the configPull with package defaults.
func newConfigPull(mqtt *fimpgo.MqttTransport, sourceRn fimptype.ResourceNameT, telemetry Telemetry) *configPull {
	return &configPull{
		mqtt:         mqtt,
		sourceRn:     sourceRn,
		telemetry:    telemetry,
		fallbackPoll: DefaultPollInterval,
		requestTopic: ConfigRequestTopic,
		timeout:      30,
	}
}

// start begins the poll loop. The first poll fires immediately and is
// non-blocking (runs in a background goroutine).
func (cp *configPull) start() error {
	if cp.fallbackPoll <= 0 {
		return fmt.Errorf("telemetry: config pull: fallback poll interval must be positive")
	}

	if cp.timeout <= 0 {
		return fmt.Errorf("telemetry: config pull: request timeout must be positive")
	}

	if cp.requestTopic == "" {
		return fmt.Errorf("telemetry: config pull: request topic must not be empty")
	}

	cp.lock.Lock()
	defer cp.lock.Unlock()

	if cp.timer != nil {
		return nil // already running
	}

	cp.client = fimpgo.NewSyncClient(cp.mqtt)

	responseTopic := fmt.Sprintf(configResponseTopicFmt, cp.sourceRn)

	if err := cp.client.AddSubscription(responseTopic); err != nil {
		cp.client.Stop()
		cp.client = nil

		return fmt.Errorf("telemetry: config pull: subscribe: %w", err)
	}

	cp.stopped = false
	cp.scheduleLocked(0)

	log.Infof("[cliff] Telemetry config pull from src=%s", cp.sourceRn)

	return nil
}

// stop cancels any pending poll and stops an owned sync client.
func (cp *configPull) stop() {
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

	log.Infof("[cliff] Telemetry config pull stopped (source=%s)", cp.sourceRn)
}

// pollResult holds the outcome of a single config poll.
type pollResult struct {
	delay time.Duration
	cfg   *ConfigResponse // nil when the request or parse failed
}

// scheduleLocked schedules the next poll after the given delay.
// Must be called with cp.lock held.
func (cp *configPull) scheduleLocked(delay time.Duration) {
	var t *time.Timer

	t = time.AfterFunc(delay, func() {
		cp.lock.Lock()

		if cp.stopped || cp.timer != t {
			cp.lock.Unlock()

			return
		}

		// Release lock during network I/O. If stop() is called
		// concurrently, it cancels the SyncClient's transport which
		// causes SendFimp to return an error promptly.
		client := cp.client
		cp.lock.Unlock()

		result := cp.poll(client)

		cp.lock.Lock()
		defer cp.lock.Unlock()

		// Re-check both flags: stopped gates shutdown, timer identity
		// gates stop+restart races where a new timer replaced this one.
		if cp.stopped || cp.timer != t {
			return
		}

		if result.cfg != nil {
			cp.applyConfig(result.cfg)
		}

		cp.scheduleLocked(result.delay)
	})

	cp.timer = t
}

// poll sends a config request and parses the response. Returns the
// delay until the next poll and the parsed config (nil on failure).
// Does not apply config - the caller decides based on stop/timer state.
func (cp *configPull) poll(client SyncRequester) pollResult {
	msg := fimpgo.NewNullMessage(CmdGetConfig, Service, nil, nil, nil)
	msg.Source = cp.sourceRn
	msg.ResponseToTopic = fmt.Sprintf(configResponseTopicFmt, cp.sourceRn)

	resp, err := client.SendFimp(cp.requestTopic, msg, cp.timeout)
	if err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry config pull failed, retrying in %s", cp.fallbackPoll)

		return pollResult{delay: cp.fallbackPoll}
	}

	if resp.Interface != EvtConfigReport {
		log.Warnf("[cliff] Telemetry config pull: unexpected response type %q, retrying in %s", resp.Interface, cp.fallbackPoll)

		return pollResult{delay: cp.fallbackPoll}
	}

	var cfg ConfigResponse
	if err := resp.GetObjectValue(&cfg); err != nil {
		log.WithError(err).Warnf("[cliff] Telemetry config pull: failed to parse response, retrying in %s", cp.fallbackPoll)

		return pollResult{delay: cp.fallbackPoll}
	}

	delay := cp.fallbackPoll

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

// applyConfig persists the cloud config to the reporter. Enable and
// SetSuppressedDomains are applied independently: a failure in one does
// not prevent the other. The next poll reconciles any partial state.
func (cp *configPull) applyConfig(cfg *ConfigResponse) {
	var failed bool

	if err := cp.telemetry.Enable(cfg.Enabled); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry config pull: failed to apply enabled=%v", cfg.Enabled)

		failed = true
	}

	if err := cp.telemetry.SetSuppressedDomains(cfg.Suppressed); err != nil {
		log.WithError(err).Errorf("[cliff] Telemetry config pull: failed to apply suppressed domains")

		failed = true
	}

	if !failed {
		log.Infof("[cliff] Telemetry config applied (enabled=%v, suppressed=%v)", cfg.Enabled, cfg.Suppressed)
	}
}
