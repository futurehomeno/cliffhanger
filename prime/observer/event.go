package observer

import (
	"github.com/futurehomeno/cliffhanger/event"
	"github.com/futurehomeno/cliffhanger/prime"
)

const Domain = "prime"

type Event struct {
	Component string
	Command   string
	ID        int
}

func newEvent(component string, command string, id int) *event.Event {
	return &event.Event{
		Domain: Domain,
		Payload: &Event{
			Component: component,
			Command:   command,
			ID:        id,
		},
	}
}

func WaitForDeviceChange() event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		WaitForComponent(prime.ComponentDevice),
		WaitForCommand(prime.CmdAdd, prime.CmdDelete, prime.CmdEdit),
	)
}

func WaitForThingChange() event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		WaitForComponent(prime.ComponentThing),
		WaitForCommand(prime.CmdAdd, prime.CmdDelete, prime.CmdEdit),
	)
}

func WaitForRoomChange() event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		WaitForComponent(prime.ComponentRoom),
		WaitForCommand(prime.CmdAdd, prime.CmdDelete, prime.CmdEdit),
	)
}

func WaitForAreaChange() event.Filter {
	return event.And(
		event.WaitForDomain(Domain),
		WaitForComponent(prime.ComponentArea),
		WaitForCommand(prime.CmdAdd, prime.CmdDelete, prime.CmdEdit),
	)
}

func WaitForComponent(component string) event.Filter {
	return event.FilterFn(func(event *event.Event) bool {
		e, ok := event.Payload.(*Event)
		if !ok {
			return false
		}

		return e.Component == component
	})
}

func WaitForCommand(commands ...string) event.Filter {
	return event.FilterFn(func(event *event.Event) bool {
		e, ok := event.Payload.(*Event)
		if !ok {
			return false
		}

		for _, command := range commands {
			if e.Command == command {
				return true
			}
		}

		return false
	})
}
