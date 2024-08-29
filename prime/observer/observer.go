package observer

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/prime"
)

// Observer represents an observer of the prime components.
type Observer interface {
	Refresh(force bool) error
	Update(notification *prime.Notify) error
	GetDevices() (prime.Devices, error)
	GetThings() (prime.Things, error)
	GetRooms() (prime.Rooms, error)
	GetAreas() (prime.Areas, error)
}

// New creates a new observer of the prime components.
func New(
	client prime.Client,
	eventManager event.Manager,
	refreshInterval time.Duration,
	components ...string,
) (Observer, error) {
	o := &observer{
		client:          client,
		eventManager:    eventManager,
		components:      components,
		strategies:      make(map[string]func(notification *prime.Notify) error),
		lock:            &sync.RWMutex{},
		refreshed:       false,
		refreshInterval: refreshInterval,
		set:             newSet(nil),
	}

	for _, component := range components {
		switch component {
		case prime.ComponentDevice:
			o.strategies[component] = o.updateDevice
		case prime.ComponentThing:
			o.strategies[component] = o.updateThing
		case prime.ComponentRoom:
			o.strategies[component] = o.updateRoom
		case prime.ComponentArea:
			o.strategies[component] = o.updateArea
		case prime.ComponentShortcut:
			o.strategies[component] = o.updateShortcut
		default:
			return nil, fmt.Errorf("prime observer: unsupported component %s", component)
		}
	}

	return o, nil
}

type observer struct {
	client       prime.Client
	eventManager event.Manager

	components      []string
	strategies      map[string]func(notification *prime.Notify) error
	lock            *sync.RWMutex
	refreshInterval time.Duration
	refreshed       bool
	lastRefresh     time.Time
	set             *set
}

// Update updates the observed components based on the vinculum notification.
func (o *observer) Update(notification *prime.Notify) error {
	if !o.isComponentObserved(notification.Component) {
		return nil
	}

	o.lock.Lock()
	defer o.lock.Unlock()

	err := o.strategies[notification.Component](notification)
	if err != nil {
		o.refreshed = false

		return fmt.Errorf("prime observer: failed to process update for component %s: %w", notification.Component, err)
	}

	return nil
}

// GetDevices returns the devices.
func (o *observer) GetDevices() (prime.Devices, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if err := o.doRefresh(false); err != nil {
		return nil, err
	}

	return o.set.getDevices(), nil
}

// GetThings returns the things.
func (o *observer) GetThings() (prime.Things, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if err := o.doRefresh(false); err != nil {
		return nil, err
	}

	return o.set.getThings(), nil
}

// GetRooms returns the rooms.
func (o *observer) GetRooms() (prime.Rooms, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if err := o.doRefresh(false); err != nil {
		return nil, err
	}

	return o.set.getRooms(), nil
}

// GetAreas returns the areas.
func (o *observer) GetAreas() (prime.Areas, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if err := o.doRefresh(false); err != nil {
		return nil, err
	}

	return o.set.getAreas(), nil
}

// Refresh refreshes the observed components.
func (o *observer) Refresh(force bool) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.doRefresh(force)
}

func (o *observer) doRefresh(force bool) error {
	if !force && !o.isRefreshRequired() {
		return nil
	}

	componentSet, err := o.client.GetComponents(o.components...)
	if err != nil {
		return fmt.Errorf("observer: error while refreshing components: %w", err)
	}

	o.set = newSet(componentSet)
	o.refreshed = true
	o.lastRefresh = time.Now()

	o.eventManager.Publish(newRefreshEvent(o.components))

	return nil
}

func (o *observer) isRefreshRequired() bool {
	if !o.refreshed || o.lastRefresh.IsZero() {
		return true
	}

	return time.Since(o.lastRefresh) > o.refreshInterval
}

func (o *observer) isComponentObserved(component string) bool {
	_, ok := o.strategies[component]

	return ok
}

//nolint:dupl
func (o *observer) updateDevice(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		device, err := notification.GetDevice()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add device: %w", err)
		}

		o.set.addDevice(device)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, device.ID))

	case prime.CmdEdit:
		device, err := notification.GetDevice()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update device: %w", err)
		}

		o.set.updateDevice(device)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, device.ID))

	case prime.CmdDelete:
		id, err := notification.ParseIntegerID()
		if err != nil {
			return fmt.Errorf("prime observer: failed to parse ID of a device: %w", err)
		}

		o.set.deleteDevice(id)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, id))
	}

	return nil
}

//nolint:dupl
func (o *observer) updateThing(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		thing, err := notification.GetThing()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add thing: %w", err)
		}

		o.set.addThing(thing)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, thing.ID))

	case prime.CmdEdit:
		thing, err := notification.GetThing()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update thing: %w", err)
		}

		o.set.updateThing(thing)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, thing.ID))

	case prime.CmdDelete:
		id, err := notification.ParseIntegerID()
		if err != nil {
			return fmt.Errorf("prime observer: failed to parse ID of a thing: %w", err)
		}

		o.set.deleteThing(id)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, id))
	}

	return nil
}

//nolint:dupl
func (o *observer) updateRoom(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		room, err := notification.GetRoom()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add room: %w", err)
		}

		o.set.addRoom(room)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, room.ID))

	case prime.CmdEdit:
		room, err := notification.GetRoom()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update room: %w", err)
		}

		o.set.updateRoom(room)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, room.ID))

	case prime.CmdDelete:
		id, err := notification.ParseIntegerID()
		if err != nil {
			return fmt.Errorf("prime observer: failed to parse ID of a room: %w", err)
		}

		o.set.deleteRoom(id)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, id))
	}

	return nil
}

//nolint:dupl
func (o *observer) updateArea(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		area, err := notification.GetArea()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add area: %w", err)
		}

		o.set.addArea(area)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, area.ID))

	case prime.CmdEdit:
		area, err := notification.GetArea()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update area: %w", err)
		}

		o.set.updateArea(area)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, area.ID))

	case prime.CmdDelete:
		id, err := notification.ParseIntegerID()
		if err != nil {
			return fmt.Errorf("prime observer: failed to parse ID of an area: %w", err)
		}

		o.set.deleteArea(id)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, id))
	}

	return nil
}

//nolint:dupl
func (o *observer) updateShortcut(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		shortcut, err := notification.GetShortcut()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add shortcut: %w", err)
		}

		o.set.addShortcut(shortcut)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, shortcut.ID))

	case prime.CmdEdit:
		shortcut, err := notification.GetShortcut()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update shortcut: %w", err)
		}

		o.set.updateShortcut(shortcut)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, shortcut.ID))

	case prime.CmdDelete:
		id, err := notification.ParseIntegerID()
		if err != nil {
			return fmt.Errorf("prime observer: failed to parse ID of a shortcut: %w", err)
		}

		o.set.deleteShortcut(id)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, id))
	}

	return nil
}
