package root

import (
	"fmt"
	"os"
	"os/signal"
	"runtime/debug"
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
	"github.com/futurehomeno/cliffhanger/lifecycle"
	"github.com/futurehomeno/cliffhanger/router"
	"github.com/futurehomeno/cliffhanger/task"
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

	mqtt               *fimpgo.MqttTransport
	lifecycle          *lifecycle.Lifecycle
	resourceName       fimptype.ResourceNameT
	topicSubscriptions []string
	messageRouter      router.Router
	taskManager        task.Manager
	services           []Service
	resetters          []Resetter

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
		log.Error("Reset app err: %w", err)
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
	err := a.Start()
	if err != nil {
		return err
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	defer signal.Stop(signals)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Error(string(debug.Stack()))
				log.Error(r)
				panic(r)
			}
		}()

		<-signals
		s := strings.Builder{}
		if err := pprof.Lookup("goroutine").WriteTo(&s, 2); err == nil {
			log.Warnf("%s\n", utils.FilterGoroutinesByKeywords(s.String(), []string{"mutex", "semaphore", "panic", "lock"}))
		}

		err = a.Stop()

		if err != nil {
			log.Errorf("[cliff] Stop err: %v", err)
		}
	}()

	return a.Wait()
}

// doStart performs the application startup.
func (a *app) doStart() error {
	if a.running {
		return nil
	}

	log.Info("[cliff] Start app")

	if a.lifecycle != nil {
		a.lifecycle.SetAppHealth(lifecycle.AppHealthStarting, nil)
	}

	err := a.mqtt.Start(10 * time.Second)
	if err != nil {
		return fmt.Errorf("start MQTT err: %w", err)
	}

	for _, service := range a.services {
		err = service.Start()
		if err != nil {
			return fmt.Errorf("start service err: %w", err)
		}
	}

	err = a.messageRouter.Start()
	if err != nil {
		return fmt.Errorf("start message router err: %w", err)
	}

	for _, topic := range config.Deduplicate(a.topicSubscriptions) {
		err = a.mqtt.Subscribe(topic)
		if err != nil {
			return fmt.Errorf("subscribe topic=%s err: %w", topic, err)
		}
	}

	err = a.taskManager.Start()
	if err != nil {
		return fmt.Errorf("start task manager err: %w", err)
	}

	a.startAuthLossWatcher()

	log.Info("[cliff] App started")

	a.running = true

	return nil
}

// startAuthLossWatcher subscribes to lifecycle auth state changes and publishes
// evt.app.state_report whenever the auth state transitions to LOST. This guarantees
// subscribers are notified when an unexpected logout occurs.
func (a *app) startAuthLossWatcher() {
	if a.lifecycle == nil || a.mqtt == nil || a.resourceName == "" {
		return
	}

	const subID = "auth_lost"

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

				if event.Type != lifecycle.StateTypeAuthState || event.State != lifecycle.AuthStateLost {
					continue
				}

				if err := sendAppStateReport(a.mqtt, a.resourceName, fimptype.ServiceNameT(a.resourceName), a.lifecycle); err != nil {
					log.WithError(err).Error("[cliff] failed to publish app state report on auth loss")
				}

			case <-a.authWatcherStopCh:
				return
			}
		}
	}()
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

	a.stopAuthLossWatcher()

	if a.lifecycle != nil {
		a.lifecycle.SetAppHealth(lifecycle.AppHealthTerminate, nil)
	}

	err := a.taskManager.Stop()
	if err != nil {
		return fmt.Errorf("stop task manager err: %w", err)
	}

	for _, topic := range config.Deduplicate(a.topicSubscriptions) {
		err := a.mqtt.Unsubscribe(topic)
		if err != nil {
			return fmt.Errorf("unsubscribe topic=%s err: %w", topic, err)
		}
	}

	err = a.messageRouter.Stop()
	if err != nil {
		return fmt.Errorf("stop message router err: %w", err)
	}

	for i := len(a.services) - 1; i >= 0; i-- {
		err = a.services[i].Stop()
		if err != nil {
			return fmt.Errorf("stop service err: %w", err)
		}
	}

	a.mqtt.Stop()

	a.running = false
	return nil
}

// doReset performs the application factory reset.
func (a *app) doReset() error {
	log.Info("[cliff] Factory reset of the app data")

	for _, resetter := range a.resetters {
		err := resetter.Reset()
		if err != nil {
			return fmt.Errorf("factory reset app data err: %w", err)
		}
	}

	log.Info("[cliff] Factory reset of the app data completed")

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

func sendAppStateReport(mqtt *fimpgo.MqttTransport, resourceName fimptype.ResourceNameT, appLifecycle *lifecycle.Lifecycle) error {
	msg := fimpgo.NewMessage(
		cliffapp.EvtAppStateReport,
		fimptype.ServiceNameT(resourceName),
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
