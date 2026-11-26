# One policy for answering a failed command

## Why

A scan on 2026-11-25 of ten edge adapters (adax, easee, hoiax, hue, mill, netatmo, sensibo, sonos,
tibber, zaptec), energy guard, cliffhanger, fimpgo, the mobile app, 826 error reports from hub logs
and the FIMP documentation found that a failed command is answered in five different ways: an
`evt.error.report` from the router, a hand-built report without `corid` (zaptec), a report that looks
like success, a typed app report with an error field, or nothing at all.

- Energy guard matches an answer to its command by `corid`. A report without `corid`, or a late
  failure answered as success, leaves it waiting or believing the command worked.
- The value of every router report is the fixed text "failed to process incoming message", so a
  client cannot tell a refused value from a cloud outage without parsing `msg`.
- `msg` carries the raw error chain. Hub logs show chains with request URLs, device serials and
  response bodies in it; on edge-hue-adapter the URL holds the bridge's application key.
- A value outside the advertised range is clamped by some adapters, rejected by others.
- `virtualmeter` accepts commands but does not declare `evt.error.report`; six adapter manifests do
  not declare it either.

The FIMP documentation only recommends an error interface and says nothing about `corid` or about
whether a failure must be answered. Cliffhanger's router is the de facto rule, so the policy is
written here first. This change is a proposal for review: it changes no code. The code follows in
the tasks of this change once the policy is accepted.

## What Changes

The policy, nine rules:

1. Every routed command that fails is answered with exactly one `evt.error.report`. A report that
   looks like success is never sent instead; a state report showing the real state may come before
   it.
2. Only the router sends it; the handler returns the error. Service: the request's service. Topic:
   `resp_to` when the request has one, otherwise the request's address with message type `evt`.
   `corid`: the request's `uid`. Properties `cmd_topic`, `cmd_service`, `cmd_type` and `msg`. A
   report on a device resource carries the `skip` storage strategy.
3. The value type is `string` and the value is an error code: `bad_request`, `out_of_range`,
   `not_supported`, `not_found`, `busy`, `unauthorized`, `device_refused`, `unavailable`, `rate_limited`,
   `internal` (fimpgo `fimptype.ErrorCodeT`, change `error-codes`). An adapter may add its own
   lower_snake codes. An error without a code is reported as `internal`, which replaces the fixed
   text "failed to process incoming message".
4. `msg` of a coded error is one plain sentence with no URL, id, key or response body. `msg` of an
   `internal` error is the error chain with URLs removed. The full chain always goes to the
   adapter's log.
5. A failure found after the command was accepted is answered with an error report carrying the
   command's `corid`. No error report is sent without a `corid`.
6. A value outside the advertised range is rejected with `out_of_range`, a value outside the
   advertised list with `not_supported`; it is never clamped, capped or rounded silently. Two
   exceptions (user, 2026-11-25): a level of `out_lvl_switch` is limited to `[min_lvl, max_lvl]`, and
   a value may be rounded to the supported step; the report then carries the value that was set.
7. App flows keep their typed reports and field names (`evt.app.config_report`,
   `evt.app.uninstall_report`, `evt.app.config_action_report`, `evt.auth.status_report`). Additive
   only: the error text field is filled whenever the outcome is a failure. `evt.error.report` stays
   for a malformed payload and a busy lock.
8. Every device service that accepts commands and every app manifest declares
   `out evt.error.report string`.
9. A failure is logged once, by the router: a coded failure at warning level, an uncoded one
   (`internal`) at error level. An adapter may limit the log line of a repeated refusal to once per
   period; the report is still sent for every command.

Spec deltas:

- `messaging/routing`: MODIFIED "Error Reporting Contract"; ADDED "Coded Errors", "A Late Failure Is
  Reported With The Command's Correlation Id", "One Log Line Per Failure".
- `adapter/network-commands`: MODIFIED "Command Error Reporting", so it points to the contract
  instead of repeating "the original error text".
- `devices/fimp-services`: MODIFIED "Specification Construction" (`virtualmeter` declares the
  report); ADDED "A Value Outside The Advertised Range Is Rejected".
- `app/app-service`: ADDED "A Failed Outcome Fills The Error Text".

Proposed only, not part of the deltas (see `design.md`): a handler panic answered with `internal`; a
command no route matches stays unanswered. The decided exceptions and the remaining open points are
listed in `design.md`.

## Impact

Downstream edge adapters, once the code follows:

- **Changed behaviour, no API break.** The value of every router error report changes from "failed
  to process incoming message" to a code (`internal` for every error that has none). `msg` of an
  uncoded error loses the URLs in its chain. No adapter under `~/proj/go` compares the old value;
  energy guard's tests only use it as fixture text in reports they build themselves.
- **New API.** `router.Error` (code, sentence, cause) and its constructor; `router.NewErrorReport`
  for a late failure. Adapters adopt them per the adapter plan in `design.md`, one change each, on
  the new cliffhanger.
- **Codes on upgrade.** Cliffhanger's own validations (unknown address, malformed payload,
  unsupported mode, current above the maximum, ...) return coded errors, so every adapter gets codes
  for them by bumping cliffhanger.
- **App flows.** No field renamed or removed. `evt.auth.status_report` after `cmd.auth.login` or
  `cmd.auth.set_tokens` also fills `error_text` when the auth state after the operation is
  `NOT_AUTHENTICATED`, `ERROR` or `LOST`.
- `virtualmeter` services declare `evt.error.report` in their inclusion report.

Compatibility:

- Only `router/handler_test.go:69` asserts the old value.
- `test/suite/expectation.go:156` `ExpectError` checks topic, type and service only, so its 281 uses
  in cliffhanger, and those in the adapters, keep passing.
- No production consumer compares the value. Energy guard reads `msg` first and the value only as a
  fallback.
- fimpgo must be released with `ErrorCodeT` before cliffhanger can use it.
