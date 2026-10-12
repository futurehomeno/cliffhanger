package root

import (
	"errors"
	"fmt"
	"os"
	"os/signal"
	"runtime/pprof"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/fimptype"
	log "github.com/sirupsen/logrus"

	cliffapp "github.com/futurehomeno/cliffhanger/app"
	"github.com/futurehomeno/cliffhanger/config"
	"github.com/futurehomeno/cliffhanger/debug"
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
	"github.com/futurehomeno/cliffhanger/telemetry"
	"github.com/futurehomeno/cliffhanger/utils"
)

// App is an interface representing a root application.
type App interface {
	// Start starts the edge application.
	Start() error
	// Stop stops the edge application maintaining a graceful shutdown.
	Stop() error
	// Reset gracefully stops the application and then resets its data.
	Reset() error
	// Wait waits for the application to stop.
	Wait() error
	// Run starts the application and waits for it to stop.
	Run() error
}

// Service is an interface representing an application service.
type Service interface {
	// Start starts the application service.
	Start() error
	// Stop stops the application service maintaining a graceful shutdown.
	Stop() error
}

// Resetter is an interface representing an application factory reset service.
type Resetter interface {
	// Reset performs a factory reset of the application data.
	Reset() error
}

// app is an implementation of root application interface.
type app struct {
	running bool
	lock    *sync.Mutex
	errCh   chan error

	mqtt                  *fimpgo.MqttTransport
	lifecycle             *lifecycle.Lifecycle
	telemetry             telemetry.Telemetry
	authLossNotify        func() error
	authLossReportEnabled func() bool
	resourceName          fimptype.ResourceNameT
	topicSubscriptions    []string
	messageRouter         router.Router
	taskManager           task.Manager
	services              []Service
	resetters             []Resetter

	authWatcherStopCh chan struct{}
	authWatcherDoneCh chan struct{}
}

// Start starts the root application.
func (a *app) Start() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.doStart()
}

// Stop stops the root application maintaining a graceful shutdown.
func (a *app) Stop() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	return a.passErr(a.doStop())
}

// Reset gracefully stops the application and then resets its data.
func (a *app) Reset() error {
	a.lock.Lock()
	defer a.lock.Unlock()

	err := a.doStop()
	if err != nil {
		log.Errorf("[cliff] Stop before reset err: %v", err)
	}

	return a.passErr(a.doReset())
}

// Wait waits for the application to stop.
func (a *app) Wait() error {
	a.lock.Lock()

	if !a.running {
		a.lock.Unlock()

		return nil
	}

	a.lock.Unlock()

	return <-a.errCh
}

// Run starts the application and waits for it to stop.
func (a *app) Run() error {
	defer telemetry.RecoverAndEmit(a.telemetry, "run", true)
	err := a.Start()
	if err != nil {
		return err
	}

	go func() {
		defer telemetry.RecoverAndEmit(a.telemetry, "signal", true)

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		defer signal.Stop(signals)

		<-signals
		s := strings.Builder{}
		if err := pprof.Lookup("goroutine").WriteTo(&s, 2); err == nil {
			if dump := utils.FilterGoroutinesByKeywords(s.String(), []string{"mutex", "semaphore", "panic", "lock"}); dump != "" {
				log.Warnf("[cliff] Goroutine dump:\n%s", dump)
			}
		}

		err = a.Stop()
		if err != nil {
			log.Errorf("[cliff] Stop err: %v", err)
		}
	}()

	if a.lifecycle != nil {
		telemetry.EmitRebootMilestone(a.telemetry, a.lifecycle.RestartsCount())
	}

	return a.Wait()
}

// doStart performs the application startup.
func (a *app) doStart() (err error) {
	if a.running {
		return nil
	}

	log.Info("[cliff] Start app")

	logBootstrapDirs()

	if a.lifecycle != nil {
		a.lifecycle.SetAppHealth(lifecycle.AppHealthStarting, nil)
	}

	// Unwind whatever did start, in reverse. Without it a failed start leaves MQTT, the services
	// and the router running while a.running stays false, so the Stop() that follows returns
	// immediately and a Reset() wipes the app data from under live services. Best effort: a
	// failure in one step must not strand the ones before it. It cannot go through doStop, which
	// short-circuits on a.running and whose task manager and router stops error out when they were
	// never started.
	var undo []func()

	pushUndo := func(what string, stop func() error) {
		undo = append(undo, func() {
			if stopErr := stop(); stopErr != nil {
				log.Errorf("[cliff] Rollback %s err: %v", what, stopErr)
			}
		})
	}

	defer func() {
		if err == nil {
			return
		}

		for i := len(undo) - 1; i >= 0; i-- {
			undo[i]()
		}

		if a.lifecycle != nil {
			a.lifecycle.SetAppHealth(lifecycle.AppHealthStartupError, nil)
		}

		debug.FlushLogs()
	}()

	if err = a.mqtt.Start(10 * time.Second); err != nil {
		return fmt.Errorf("start MQTT err: %w", err)
	}

	undo = append(undo, a.mqtt.Stop)

	// Started right after the transport so its subscription meets a live one, and only logged on
	// failure: telemetry is best effort, and aborting the whole start over it would be worse than
	// running without it.
	if a.telemetry != nil {
		if telErr := a.telemetry.Start(); telErr != nil {
			log.Errorf("[cliff] Start telemetry err: %v", telErr)
		}

		pushUndo("stop telemetry", a.telemetry.Stop)
	}

	for i, service := range a.services {
		if err = service.Start(); err != nil {
			return fmt.Errorf("start service err: %w", err)
		}

		pushUndo(fmt.Sprintf("stop service[%d]", i), service.Stop)
	}

	if err = a.messageRouter.Start(); err != nil {
		return fmt.Errorf("start message router err: %w", err)
	}

	pushUndo("stop message router", a.messageRouter.Stop)

	for _, topic := range config.Deduplicate(a.topicSubscriptions) {
		if err = a.mqtt.Subscribe(topic); err != nil {
			return fmt.Errorf("subscribe topic=%s err: %w", topic, err)
		}

		pushUndo("unsubscribe topic="+topic, func() error { return a.mqtt.Unsubscribe(topic) })
	}

	// Subscribed before the task manager starts: a WhenAppIsRunning-gated task (e.g.
	// ConnectivityChecker) can run its first probe as soon as taskManager.Start() spawns its
	// goroutine, and an AuthStateLost emitted before this subscription exists is gone for good
	// (Lifecycle.emitStateChangeEvent does not buffer for a not-yet-subscribed watcher).
	a.startAuthLossWatcher(a.telemetry)

	undo = append(undo, a.stopAuthLossWatcher)

	if err = a.taskManager.Start(); err != nil {
		return fmt.Errorf("start task manager err: %w", err)
	}

	log.Info("[cliff] App started")

	a.running = true

	return nil
}

func (a *app) startAuthLossWatcher(tel telemetry.Telemetry) {
	if a.lifecycle == nil || a.mqtt == nil || a.resourceName == "" {
		return
	}

	const subID = "auth_lost"

	// Only an authenticated session can be lost: arm on AUTHENTICATED and report a LOST only
	// while armed. Seed armed BEFORE subscribing. SetAuthState updates the state field and then
	// emits under the same lock, so a LOST landing after Subscribe but before the seed read would
	// be captured in ch yet make AuthState() return LOST — seeding armed=false and dropping the
	// very event that was queued. Reading first means any event ch delivers is evaluated against
	// the state that preceded it; the only thing missed is a LOST that fires between the read and
	// Subscribe, which is never queued and is equivalent to an already-lost startup.
	armed := a.lifecycle.AuthState() == lifecycle.AuthStateAuthenticated

	ch := a.lifecycle.Subscribe(subID, 5)

	a.authWatcherStopCh = make(chan struct{})
	a.authWatcherDoneCh = make(chan struct{})

	go func() {
		defer close(a.authWatcherDoneCh)
		defer a.lifecycle.Unsubscribe(subID)

		for {
			select {
			case event, ok := <-ch:
				if !ok {
					return
				}

				if event.Type != lifecycle.StateTypeAuthState {
					continue
				}

				var report bool
				report, armed = nextAuthArm(armed, event.State)
				if report {
					a.reportAuthLoss(tel, event.Params["reason"])
				}

			case <-a.authWatcherStopCh:
				return
			}
		}
	}()
}

// nextAuthArm advances the arm state for one auth-state event: an AUTHENTICATED
// session arms the watcher; a LOST reports only while armed, then disarms so a
// repeated LOST (e.g. a connection-only change) does not re-fire; NOT_AUTHENTICATED
// (logout/reset) disarms so a later failed re-login does not report a phantom loss.
func nextAuthArm(armed bool, s lifecycle.State) (report, nowArmed bool) {
	switch s {
	case lifecycle.AuthStateAuthenticated:
		return false, true
	case lifecycle.AuthStateLost:
		if armed {
			return true, false
		}

		return false, false
	case lifecycle.AuthStateNotAuthenticated:
		// Logout/reset drive the app to NOT_AUTHENTICATED (MarkNotConfigured). Disarm so a later
		// failed re-login — which never reaches AUTHENTICATED, only 401 -> LOST — does not report a
		// loss for a session that was never re-established.
		return false, false
	}

	return false, armed
}

// reportAuthLoss notifies the user, publishes the FIMP app state report and emits a
// telemetry event for a lost authenticated session. reason, when set, states why
// (e.g. "unauthorized"). All three are suppressed when auth-loss reporting is disabled.
func (a *app) reportAuthLoss(tel telemetry.Telemetry, reason string) {
	// The gate is evaluated per loss so it can follow a runtime config flag; a nil gate
	// reports every loss.
	if a.authLossReportEnabled != nil && !a.authLossReportEnabled() {
		return
	}

	if err := sendAppStateReport(a.mqtt, a.resourceName, fimptype.ServiceNameT(a.resourceName), a.lifecycle); err != nil {
		log.Errorf("[cliff] Publish app state report err: %v, auth loss reason: %s", err, reason)
	}

	// Emit under the historical "cause" key that shipped on develop so downstream telemetry
	// consumers keyed on data.cause keep matching; the value is the same loss reason.
	telemetry.Emit(tel, telemetry.DomainAuth, telemetry.EventLoggedOut, map[string]any{"cause": reason})

	if a.authLossNotify != nil {
		if err := a.authLossNotify(); err != nil {
			log.Errorf("[cliff] Send auth-loss notification err: %v, reason: %s", err, reason)
		}
	}
}

// stopAuthLossWatcher stops the auth loss watcher goroutine started by startAuthLossWatcher.
func (a *app) stopAuthLossWatcher() {
	if a.authWatcherStopCh == nil {
		return
	}

	close(a.authWatcherStopCh)
	<-a.authWatcherDoneCh
	a.authWatcherStopCh = nil
	a.authWatcherDoneCh = nil
}

// doStop performs the application shutdown.
func (a *app) doStop() error {
	if !a.running {
		return nil
	}

	// Deferred so buffered diagnostics leading up to a failure are not lost on an early
	// return below - exactly the case where they matter most.
	defer debug.FlushLogs()

	a.stopAuthLossWatcher()

	if a.lifecycle != nil {
		a.lifecycle.SetAppHealth(lifecycle.AppHealthTerminate, nil)
	}

	// Best effort throughout: returning on the first failure left the app half stopped and still
	// marked as running, and Reset() goes on to wipe the data either way.
	var errs []error

	if err := a.taskManager.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("stop task manager err: %w", err))
	}

	for _, topic := range config.Deduplicate(a.topicSubscriptions) {
		if err := a.mqtt.Unsubscribe(topic); err != nil {
			errs = append(errs, fmt.Errorf("unsubscribe topic=%s err: %w", topic, err))
		}
	}

	if err := a.messageRouter.Stop(); err != nil {
		errs = append(errs, fmt.Errorf("stop message router err: %w", err))
	}

	for i := len(a.services) - 1; i >= 0; i-- {
		if err := a.services[i].Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop service[%d] err: %w", i, err))
		}
	}

	// Stopped before the transport it publishes on. Leaving it running let a late cloud config
	// report land after a Reset() and write telemetry config back into the store just wiped.
	if a.telemetry != nil {
		if err := a.telemetry.Stop(); err != nil {
			errs = append(errs, fmt.Errorf("stop telemetry err: %w", err))
		}
	}

	a.mqtt.Stop()

	a.running = false

	return errors.Join(errs...)
}

// doReset performs the application factory reset.
func (a *app) doReset() error {
	log.Info("[cliff] Start app data reset")

	for _, resetter := range a.resetters {
		err := resetter.Reset()
		if err != nil {
			return fmt.Errorf("factory reset app data err: %w", err)
		}
	}

	log.Info("[cliff] App data reset done")

	return nil
}

// passErr optionally passes the error to the error channel.
func (a *app) passErr(err error) error {
	select {
	case a.errCh <- err:
	default:
	}

	return err
}

func sendAppStateReport(mqtt *fimpgo.MqttTransport, resourceName fimptype.ResourceNameT, srvName fimptype.ServiceNameT, appLifecycle *lifecycle.Lifecycle) error {
	msg := fimpgo.NewMessage(
		cliffapp.EvtAppStateReport,
		srvName,
		fimptype.VTypeObject,
		appLifecycle.AllStates(),
		nil,
		nil,
		nil,
	)

	topic := fmt.Sprintf("pt:j1/mt:evt/rt:app/rn:%s/ad:1", resourceName)

	if err := mqtt.PublishToTopic(topic, msg); err != nil {
		return fmt.Errorf("failed to publish app state report: %w", err)
	}

	return nil
}
