# event.Manager.Subscribe panics on a nil Filter

`event/manager.go:44` converts each `Filter` with `predicates[i] = f.Filter`; taking the method value
of a nil interface panics, so `manager.Subscribe(id, n, nil)` (or a `Listener` handler built with a
nil filter) panics at subscribe time. `Bus.Subscribe` deliberately ignores nil predicates
(`event/bus.go` `matches`), and the main spec (tasks-and-events, Subscription Registry Semantics)
said "a nil filter SHALL be ignored" without scoping it to `Bus` — the delta now scopes it.

Close by skipping nil filters in `manager.Subscribe` (and a test), then widen the requirement back
to the manager.
