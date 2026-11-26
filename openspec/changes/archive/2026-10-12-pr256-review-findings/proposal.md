# Every reporting task skips a state not reported yet; a nil FilterFn is ignored

From the PR #256 review: `adapter.ErrNotReported` was skipped only by the presence, numericsensor,
scenectrl and battery tasks, so any other service whose controller returns it logged an error every
tick. A typed nil `event.FilterFn` passed the manager's nil check and panicked on the first publish.
