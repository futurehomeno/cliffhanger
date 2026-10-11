# root.Builder shares its routing/subscription slices with the built app

`root/builder.go:179-190` (`prepareRouting`, `//nolint:gocritic`) appends the discovery route,
`discovery.Topic` and the gateway reset route/topic directly onto `b.routing` and
`b.topicSubscriptions`. When those slices have spare capacity the built app's router and
subscription list share the builder's backing array, so a later `WithRouting`/`WithTopicSubscription`
on the same builder (or a second `Build`) overwrites the already-built app's discovery route or
topic in place. Rare (builders are normally used once) but silent.

Close with `slices.Concat`/`slices.Clone` in `prepareRouting` and drop the nolint; add a test that
builds, appends a route to the builder, and checks discovery still answers.
