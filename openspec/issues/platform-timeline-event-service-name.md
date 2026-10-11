# notification: Timeline.Event is sent with service kind_owl

`notification/timeline.go:49` builds the `cmd.timeline.set` message for `Timeline.Event` with service
`kind_owl`, while `Timeline.Message` (`timeline.go:66`) uses `time_owl`, and both go to
`pt:j1/mt:cmd/rt:app/rn:time_owl/ad:1`. It looks like a copy from `notification.go`. If time_owl
filters on the service name, timeline events are dropped silently.

Close by confirming what time_owl expects (hub team / time_owl source) and either switching Event to
`time_owl` or recording `kind_owl` as deliberate in the diagnostics spec; the existing test
(`notification/timeline_test.go`) pins the current value.
