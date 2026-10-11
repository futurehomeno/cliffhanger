# A nil task in task.NewManager leaves the manager unstoppable

`task/manager.go:45` does `wg.Add(len(r.tasks))` and sets `stopCh` before the loop, then
`task.duration` (line 48) dereferences each task. A nil `*Task` panics out of `Start` with the lock
released, `stopCh` set and the WaitGroup counting goroutines that were never spawned, so a later
`Stop` blocks forever. The `task == nil` guard in `run` (line 112) is dead code because `Start`
panics first. Nil tasks arrive easily through `task.Combine` of optional slices built by adapters.

Close by dropping (and logging) nil tasks in `NewManager`/`Start` before `wg.Add`, removing the dead
guard, and relaxing the "nil task SHALL panic out of Start" sentence in Task Panic Isolation.
