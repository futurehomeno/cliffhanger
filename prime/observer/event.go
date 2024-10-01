package observer

import (
	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/prime"
)

const (
	Domain = "prime"
	Class  = "observer"
)

type (
	ComponentEvent struct {
		event.Event

		Component string
		Command   string
		ID        int
	}

	DeleteComponentEvent struct {
		event.Event

		ComponentName string
		Component     any
		ID            int
	}

	RefreshEvent struct {
		event.Event

		Components []string
	}
)

func newComponentEvent(component string, command string, id int) event.Event {
	return &ComponentEvent{
		Event:     event.New(Domain, Class),
		Component: component,
		Command:   command,
		ID:        id,
	}
}

func newDeleteComponentEvent(componentName string, component any, id int) event.Event {
	return &DeleteComponentEvent{
		Event:         event.New(Domain, Class),
		ComponentName: componentName,
		Component:     component,
		ID:            id,
	}
}

func newRefreshEvent(components []string) event.Event {
	return &RefreshEvent{
		Event:      event.New(Domain, Class),
		Components: components,
	}
}

func WaitForDeviceChange() event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		event.Or(
			WaitForRefresh(prime.ComponentDevice),
			WaitForComponent(prime.ComponentDevice, prime.CmdAdd, prime.CmdDelete, prime.CmdEdit),
		),
	)
}

func WaitForThingChange() event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		event.Or(
			WaitForRefresh(prime.ComponentThing),
			WaitForComponent(prime.ComponentThing, prime.CmdAdd, prime.CmdDelete, prime.CmdEdit),
		),
	)
}

func WaitForRoomChange() event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		event.Or(
			WaitForRefresh(prime.ComponentRoom),
			WaitForComponent(prime.ComponentRoom, prime.CmdAdd, prime.CmdDelete, prime.CmdEdit),
		),
	)
}

func WaitForAreaChange() event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		event.Or(
			WaitForRefresh(prime.ComponentArea),
			WaitForComponent(prime.ComponentArea, prime.CmdAdd, prime.CmdDelete, prime.CmdEdit),
		),
	)
}

func WaitForComponent(component string, commands ...string) event.Filter {
	return event.FilterFn(func(event event.Event) bool {
		e, ok := event.(*ComponentEvent)
		if !ok {
			return false
		}

		if e.Component != component {
			return false
		}

		if len(commands) == 0 {
			return true
		}

		for _, command := range commands {
			if e.Command == command {
				return true
			}
		}

		return false
	})
}

func WaitForRefresh(component string) event.Filter {
	return event.FilterFn(func(event event.Event) bool {
		e, ok := event.(*RefreshEvent)
		if !ok {
			return false
		}

		for _, c := range e.Components {
			if component == c {
				return true
			}
		}

		return false
	})
}

func WaitForRoomDeletion() event.Filter {
	return event.FilterFn(func(event event.Event) bool {
		e, ok := event.(*DeleteComponentEvent)
		if !ok {
			return false
		}

		return e.ComponentName == prime.ComponentRoom
	})
}
