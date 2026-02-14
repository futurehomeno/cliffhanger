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

	"github.com/futurehomeno/fimpgo"
	log "github.com/sirupsen/logrus"

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
	topicSubscriptions []string
	messageRouter      router.Router
	taskManager        task.Manager
	services           []Service
	resetters          []Resetter
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
		a.lifecycle.SetAppState(lifecycle.AppStateStarting, nil)
	}

	err := a.mqtt.Start()
	if err != nil {
		return fmt.Errorf("start the MQTT broker err: %w", err)
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

	log.Debug("[cliff] App started")

	a.running = true

	return nil
}

// doStop performs the application shutdown.
func (a *app) doStop() error {
	if !a.running {
		return nil
	}

	if a.lifecycle != nil {
		a.lifecycle.SetAppState(lifecycle.AppStateTerminate, nil)
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
