package task

import (
	"github.com/futurehomeno/cliffhanger/lifecycle"
)

// Voter is an interface representing a task execution voter.
type Voter interface {
	// Vote provides with a binary answer whether the task should be executed or not at the current time.
	Vote() bool
}

// VoterFn is an adapter allowing usage of anonymous function as a service meeting Voter interface.
type VoterFn func() bool

// Vote provides with a binary answer whether the task should be executed or not at the current time.
func (f VoterFn) Vote() bool {
	return f()
}

// WhenNot inverts provided input voter.
func WhenNot(v Voter) Voter {
	return VoterFn(func() bool {
		return !v.Vote()
	})
}

// WhenAppIsStarting is a task voter allowing a task to run only if relevant state is met.
func WhenAppIsStarting(l *lifecycle.Lifecycle) Voter {
	return VoterFn(func() bool {
		return l.AppState() == lifecycle.AppStateStarting
	})
}

// WhenAppIsNotConfigured is a task voter allowing a task to run only if relevant state is met.
func WhenAppIsNotConfigured(l *lifecycle.Lifecycle) Voter {
	return VoterFn(func() bool {
		return l.AppState() == lifecycle.AppStateNotConfigured
	})
}

// WhenAppIsRunning is a task voter allowing a task to run only if relevant state is met.
func WhenAppIsRunning(l *lifecycle.Lifecycle) Voter {
	return VoterFn(func() bool {
		return l.AppState() == lifecycle.AppStateRunning
	})
}

// WhenAppIsTerminating is a task voter allowing a task to run only if relevant state is met.
func WhenAppIsTerminating(l *lifecycle.Lifecycle) Voter {
	return VoterFn(func() bool {
		return l.AppState() == lifecycle.AppStateTerminate
	})
}

// WhenAppEncounteredStartupError is a task voter allowing a task to run only if relevant state is met.
func WhenAppEncounteredStartupError(l *lifecycle.Lifecycle) Voter {
	return VoterFn(func() bool {
		return l.AppState() == lifecycle.AppStateStartupError
	})
}

// WhenAppEncounteredError is a task voter allowing a task to run only if relevant state is met.
func WhenAppEncounteredError(l *lifecycle.Lifecycle) Voter {
	return VoterFn(func() bool {
		return l.AppState() == lifecycle.AppStateError
	})
}

// WhenAppIsConnected is a task voter allowing a task to run only if relevant state is met.
func WhenAppIsConnected(l *lifecycle.Lifecycle) Voter {
	return VoterFn(func() bool {
		return l.ConnectionState() == lifecycle.ConnStateConnected
	})
}

// WhenAppIsDisconnected is a task voter allowing a task to run only if relevant state is met.
func WhenAppIsDisconnected(l *lifecycle.Lifecycle) Voter {
	return VoterFn(func() bool {
		return l.ConnectionState() == lifecycle.ConnStateDisconnected
	})
}
