# test/suite processMessage writes Expectation.Publish/Reply without a lock

`test/suite/router.go:240-250` assigns `e.Publish = e.PublishFn()` and `e.Reply = e.ReplyFn()` on
the shared `*Expectation` outside `r.lock`, while the cliffhanger router processes messages with
concurrency 5. Two messages matching the same expectation at once race on those fields (a data race
under `-race`) and one handler can publish/reply with the other's value. Close by using locals
(`publish := e.Publish; if e.PublishFn != nil { publish = e.PublishFn() }`) instead of writing the
fields.
