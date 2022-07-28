package observer

import (
	"fmt"
	"sync"
	"time"

	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/prime"
)

type Observer interface {
	Refresh(force bool) error
	Update(notification *prime.Notify) error
	GetDevices() (prime.Devices, error)
	GetThings() (prime.Things, error)
	GetRooms() (prime.Rooms, error)
	GetAreas() (prime.Areas, error)
}

func New(
	client prime.Client,
	eventManager event.Manager,
	refreshInterval time.Duration,
	components ...string,
) (Observer, error) {
	o := &observer{
		client:          client,
		eventManager:    eventManager,
		components:      make(map[string]func(notification *prime.Notify) error),
		lock:            &sync.RWMutex{},
		refreshed:       false,
		refreshInterval: refreshInterval,
	}

	for _, component := range components {
		switch component {
		case prime.ComponentDevice:
			o.components[component] = o.updateDevice
		case prime.ComponentThing:
			o.components[component] = o.updateThing
		case prime.ComponentRoom:
			o.components[component] = o.updateRoom
		case prime.ComponentArea:
			o.components[component] = o.updateArea
		default:
			return nil, fmt.Errorf("prime observer: unsupported component %s", component)
		}
	}

	return o, nil
}

type observer struct {
	client       prime.Client
	eventManager event.Manager

	components      map[string]func(notification *prime.Notify) error
	lock            *sync.RWMutex
	refreshInterval time.Duration
	refreshed       bool
	lastRefresh     time.Time
	set             *set
}

func (o *observer) Update(notification *prime.Notify) error {
	if !o.isComponentObserved(notification.Component) {
		return nil
	}

	o.lock.Lock()
	defer o.lock.Unlock()

	err := o.components[notification.Component](notification)
	if err != nil {
		o.refreshed = false

		return fmt.Errorf("prime observer: failed to process update for component %s: %w", notification.Component, err)
	}

	return nil
}

func (o *observer) isComponentObserved(component string) bool {
	_, ok := o.components[component]

	return ok
}

func (o *observer) observedComponents() []string {
	var components []string

	for component := range o.components {
		components = append(components, component)
	}

	return components
}

// nolint:dupl
func (o *observer) updateDevice(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		device, err := notification.GetDevice()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add device with ID %d: %w", device.ID, err)
		}

		o.set.addDevice(device)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, device.ID))

	case prime.CmdEdit:
		device, err := notification.GetDevice()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update device with ID %d: %w", device.ID, err)
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

// nolint:dupl
func (o *observer) updateThing(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		thing, err := notification.GetThing()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add thing with ID %d: %w", thing.ID, err)
		}

		o.set.addThing(thing)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, thing.ID))

	case prime.CmdEdit:
		thing, err := notification.GetThing()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update thing with ID %d: %w", thing.ID, err)
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

// nolint:dupl
func (o *observer) updateRoom(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		room, err := notification.GetRoom()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add room with ID %d: %w", room.ID, err)
		}

		o.set.addRoom(room)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, room.ID))

	case prime.CmdEdit:
		room, err := notification.GetRoom()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update room with ID %d: %w", room.ID, err)
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

// nolint:dupl
func (o *observer) updateArea(notification *prime.Notify) error {
	switch notification.Cmd {
	case prime.CmdAdd:
		area, err := notification.GetArea()
		if err != nil {
			return fmt.Errorf("prime observer: failed to add area with ID %d: %w", area.ID, err)
		}

		o.set.addArea(area)
		o.eventManager.Publish(newComponentEvent(notification.Component, notification.Cmd, area.ID))

	case prime.CmdEdit:
		area, err := notification.GetArea()
		if err != nil {
			return fmt.Errorf("prime observer: failed to update area with ID %d: %w", area.ID, err)
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

func (o *observer) Refresh(force bool) error {
	o.lock.Lock()
	defer o.lock.Unlock()

	return o.doRefresh(force)
}

func (o *observer) doRefresh(force bool) error {
	if !force && !o.isRefreshRequired() {
		return nil
	}

	componentSet, err := o.client.GetComponents(o.observedComponents()...)
	if err != nil {
		return fmt.Errorf("observer: error while refreshing components: %w", err)
	}

	o.set = &set{ComponentSet: componentSet}
	o.refreshed = true
	o.lastRefresh = time.Now()

	o.eventManager.Publish(newRefreshEvent(o.observedComponents()))

	return nil
}

func (o *observer) isRefreshRequired() bool {
	if !o.refreshed || o.lastRefresh.IsZero() {
		return true
	}

	return time.Since(o.lastRefresh) > o.refreshInterval
}

func (o *observer) GetDevices() (prime.Devices, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if err := o.doRefresh(false); err != nil {
		return nil, err
	}

	return o.set.getDevices(), nil
}

func (o *observer) GetThings() (prime.Things, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if err := o.doRefresh(false); err != nil {
		return nil, err
	}

	return o.set.getThings(), nil
}

func (o *observer) GetRooms() (prime.Rooms, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if err := o.doRefresh(false); err != nil {
		return nil, err
	}

	return o.set.getRooms(), nil
}

func (o *observer) GetAreas() (prime.Areas, error) {
	o.lock.Lock()
	defer o.lock.Unlock()

	if err := o.doRefresh(false); err != nil {
		return nil, err
	}

	return o.set.getAreas(), nil
}
