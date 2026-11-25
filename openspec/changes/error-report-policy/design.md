# Design: one policy for answering a failed command

The policy and its nine rules are in `proposal.md`. This file records the Go shape the code will
follow, what of it exists already, what is only proposed, what is still open, and the order in which
the adapters follow. Line numbers are those of `develop` at `4db89f4`.

## Go shape

The code reuses what exists and adds one type and one function.

- **Constants stay.** `router.EvtErrorReport`, `router.EvtSuccessReport` and `PropertyMsg`,
  `PropertyCmdTopic`, `PropertyCmdService`, `PropertyCmdType` (`router/routing.go:8-16`) keep their
  names and values.
- **Codes come from fimpgo.** `fimptype.ErrorCodeT` with the nine constants (fimpgo change
  `error-codes`), modelled on `fimptype.ValueTypeT` (`fimptype/value_type.go:3`). Cliffhanger does
  not declare its own copy.
- **`router.Error`** carries a code, the plain sentence for `msg`, and the cause for the log:

  ```go
  type Error struct {
      Code fimptype.ErrorCodeT
      Msg  string
      Err  error
  }

  func NewError(code fimptype.ErrorCodeT, msg string, err error) *Error
  func (e *Error) Error() string // Msg, then ": " and Err when Err is set: the full text for the log
  func (e *Error) Unwrap() error // Err
  ```

  A handler returns it, wrapped with `fmt.Errorf("...: %w", err)` as often as it likes; the router
  finds it with `errors.As`.
- **`handleError`** (`router/handler.go:160-199`) keeps its shape and changes three things:
  - the value at `:181`: the code of the `router.Error` found with `errors.As`, `unavailable` for an
    error wrapping `ErrNotReported` (`:15`), otherwise `internal`;
  - `msg` at `:183`: the `router.Error`'s `Msg` for a coded error; for `internal`, `err.Error()`
    with the URL of a `*url.Error` removed. `url.Error.Error()` is `Op "URL": Err`; the
    `*url.Error` found with `errors.As` is rendered again as `Op: Err` and that text replaces the
    original in the chain's text. Only the first `*url.Error` of the chain is rewritten; a URL an
    adapter writes into its own error text is the adapter's to remove (adapter plan);
  - the log level at `:161-164`: warning for `ErrNotReported` (as today) and for the expected
    refusals (`bad_request`, `out_of_range`, `not_supported`, `not_found`, `busy`, an adapter's own
    code), error for the rest. The log line at `:166-169` gains the code and keeps the full chain.
- **Busy lock.** `Handle` (`router/handler.go:112-126`) returns
  `NewError(fimptype.ErrorCodeBusy, "another operation is already running, skipping message", nil)`
  at `:115` instead of the plain `fmt.Errorf`.
- **Options stay.** `WithSilentErrors`, `WithDefaultAddress`, `WithLock`, `WithExternalLock`,
  `WithSuccessConfirmation` (`router/handler.go:233-266`) keep their meaning. The two
  `WithSilentErrors` users in cliffhanger serve events (`root/routing.go:42`,
  `prime/observer/routing.go:46`), not commands.
- **Late failure.** One exported function, built from the same code as `handleError`:

  ```go
  func NewErrorReport(request *fimpgo.Message, err error) *fimpgo.Message
  ```

  It logs the failure once (as `handleError` does) and returns the report with `Addr` already
  resolved: the parsed `request.Payload.ResponseToTopic` when set, otherwise the request's address
  with message type `evt` (the rule `router/router.go:202-230` and `handler.go:201-217` apply
  together today). The payload is built with `fimpgo.NewMessage(..., request.Payload)`, which sets
  the correlation ID to the request's UID (fimpgo `message.go:364-366`), and gets the skip storage
  strategy on a device resource. The adapter keeps the `*fimpgo.Message` of the accepted command and,
  when the failure is found, publishes `report.Payload` to `report.Addr` with the MQTT transport it
  already holds. `handleError` calls the same unexported builder with its own address, so the two
  cannot drift.
- **Coded validations in cliffhanger.** The services' own checks return `router.Error`, so every
  adapter gets codes on upgrade without changing its code:

  | check | where | code |
  | --- | --- | --- |
  | no service under the topic ("service not found under the provided address") | every `adapter/service/*/routing.go`, `adapter/routing.go` | `not_found` |
  | payload of the wrong value type ("incorrect format", "invalid format") | the same routing files, `app/routing.go` | `bad_request` |
  | capability the controller lacks ("... is not supported") | e.g. `chargepoint/service.go:543-618`, `diagnostic/service.go`, `devsys/service.go` | `not_supported` |
  | current below the minimum or above `sup_max_current` | `chargepoint/service.go:646-658`, called at `:253` (minimum 0) and `:276` (minimum 6) | `out_of_range` |
  | phase mode, charging mode not listed | `chargepoint/service.go:625-643`, `:661-672` | `not_supported` |
  | mode, setpoint type, playback, scene, unit, event not listed | `fanctrl`, `thermostat`, `waterheater`, `mediaplayer`, `scenectrl`, `numericmeter`, `numericsensor`, `battery`, `alarm` | `not_supported` |
  | setpoint outside `sup_range`/`sup_ranges` | `waterheater/service.go:314-322` | `out_of_range` |
  | parameter outside `min`/`max`, not among the options, read-only | `parameters` | `out_of_range`, `not_supported` |
  | state not reported yet | `ErrNotReported`, `router/handler.go:15` | `unavailable` |

  A specification missing `min_lvl`/`max_lvl` (`outlvlswitch` `levelRange`) is a programming error
  and stays uncoded (`internal`).
- **`virtualmeter`** adds the outgoing `evt.error.report` (`string`) to `requiredInterfaces`
  (`adapter/service/virtualmeter/specification.go:40-67`), as `numericmeter` does
  (`adapter/service/numericmeter/specification.go:60-81`).
- **App flows.** `HandleCmdAuthLogin` (`app/routing.go:441-475`) and `HandleCmdAuthSetTokens`
  (`:517-543`) also fill `ErrorText` (`:461`, `:537`) when the auth state after the operation is in
  `failedAuthStates` (`:478-482`), the set the login handler already uses for the deprecated
  `errors` field (`:468-469`). `ConfigurationReport` and `AuthenticationReport` (`app/model.go:12,18`)
  and `ButtonActionResponse` (`manifest/manifest.go:99`) do not change.

## Rule to code

| rule | today | after |
| --- | --- | --- |
| 1 exactly one report, never success instead | `router/handler.go:120-123` calls `handleError`; `router/router.go:139-165` publishes it. State report before it: chargepoint (change `charge-cmd-error-state-report`) | adapters (zaptec, easee) |
| 2 router sends; service, topic, `corid`, `cmd_*`, `skip` | service `handler.go:179`; topic `handler.go:176,201-217` and `router.go:202-230`; `corid` `handler.go:189` and `router.go:161-163`; props `handler.go:182-187`; `skip` `handler.go:193-196` | unchanged; adapters' hand-built reports go through the router (zaptec) |
| 3 value is a code, `internal` by default | fixed text at `handler.go:181` | new: `router.Error`, `errors.As` in `handleError`; coded service validations |
| 4 `msg`: sentence, or chain without URLs | `err.Error()` at `handler.go:183` | new: `router.Error.Msg`; `*url.Error` rewritten |
| 5 late failure with the command's `corid` | nothing | new: `router.NewErrorReport` |
| 6 out of range rejected, not clamped | `chargepoint/service.go:646-658` rejects; `waterheater/service.go:309-322` rounds, then rejects; `outlvlswitch/service.go:177-181` clamps | codes new; clamp and rounding are open points |
| 7 typed app reports unchanged, error text filled | `app/model.go:12,18`; `manifest/manifest.go:99`; `app/routing.go:393-426` (`op_error` `:412`), reset `:282-301`, login `:441-475`, set_tokens `:517-543`, logout `:558-577` | new: `error_text` on a failed auth state after login and set_tokens |
| 8 every service and manifest declares the report | 18 of 19 service packages | new: `virtualmeter/specification.go:40-67`; manifests in six adapters |
| 9 logged once, warning for expected refusals | `handler.go:161-169` (warning only for `ErrNotReported`) | new: level by code; adapters stop logging what they return |

## Proposed only, no decision

These are for the reviewer. They are not requirements of this change.

- **A handler panic answered with `internal`.** Today `router/router.go:121-126` recovers the panic
  and `handleProcessingPanic` (`:189-199`) logs it with the stack and calls the panic callback; the
  command gets no answer, so the caller waits for its timeout. Proposal: when the message is a
  command, the recover also publishes `NewErrorReport(msg, NewError(internal, "internal error", nil))`.
  The stack stays in the log only.
- **A command no route matches stays unanswered.** The router walks every routing
  (`router/router.go:113-115`) and skips those whose voters refuse (`:128-130`). Several apps share
  the broker and subscribe to wildcard topics, so a router cannot know that no other process handles
  the command; answering would send false errors.

## Open points

- **`outlvlswitch` level clamp against rule 6.** `adapter/service/outlvlswitch/service.go:177-181`
  clamps the level into `[min_lvl, max_lvl]`. Its comment gives the reason: "a controller reached by
  an out of range level bounds it itself (edge-hue clamps into [0, max]), so rejecting would turn a
  command that used to change the device into an error and no action at all." Spec requirement
  "Level Switch Commands" pins it with scenario "level above the maximum" (150 → 100). Rule 6 says
  reject with `out_of_range`. Undecided: reject, or keep the clamp as a written exception.
- **Water heater step rounding before the range check.** `waterheater/service.go:309-312` rounds to
  `sup_step` before the range check at `:314-322`, so 80.2 with `sup_range` up to 80 and step 0.5 is
  accepted as 80. Spec requirement "Water Heater Setpoint Normalisation" pins the rounding. Rule 6
  would reject 80.2; whether rounding a value inside the range to the step counts as "rounded
  silently" is also open.
- **zaptec "reported once per episode".** zaptec's spec `charging-control` treats Zaptec's `527`
  refusals (eco mode, scheduled power management) of an offered-current write as "not a command
  failure": the command returns no error and the adapter publishes one hand-built `evt.error.report`
  (`eco_mode`, `scheduled_power_management`) per episode, until the next accepted write. Rule 1 asks
  for one report per failed command and rule 5 for the command's `corid`. Undecided: whether every
  refused write is answered through the router with its `corid` (a refusal repeated in one episode
  answered each time), or whether these refusals stay outside rule 1 because the charger keeps
  charging.
- **Producers outside cliffhanger's router.** zigbee-ad, ecollector, CloudBridge and fhbutler send
  free text in the value. Consumers must keep accepting any string in the value; the FIMP
  documentation PR carries a compatibility note. These producers are outside this pass.
- **Which codes count as expected refusals** (warning level). This design picks `bad_request`,
  `out_of_range`, `not_supported`, `not_found`, `busy`, an adapter's own codes and `ErrNotReported`.
- **`ErrNotReported` reported as `unavailable`.** Without a code it would be `internal`, which reads
  like a bug.
- **Chargepoint minimum current.** Spec requirement "Chargepoint Command Validation" says "reject a
  requested current below 6 A", but the code applies 6 A only to `cmd.max_current.set`
  (`chargepoint/service.go:276`); the offered current accepts anything from 0 (`:253`). Energy guard,
  easee and zaptec repeat the "under 6 A" statement (adapter plan).

## Adapter plan

Only after the policy is accepted: fimpgo releases `ErrorCodeT`, cliffhanger releases the tasks of
this change, then one change per adapter on the new cliffhanger, checked by one shared conformance
test (energy guard issue 086 asks for adapter conformance suites).

| Repo | Work |
| --- | --- |
| all ten adapters | codes and plain sentences for their main failures; no URL, serial or body in error text; failures logged once |
| zaptec, mill, hoiax, tibber, adax, sensibo | manifest declares `evt.error.report` |
| mill, hoiax, netatmo, tibber, sonos, sensibo | `error_text` filled when the check after authorizing fails |
| zaptec | three hand-built reports go through the router with `corid`; current above the maximum rejected, not capped; own reports not at debug level |
| easee | a refused deferred write answered with the command's `corid`; HTTP failures logged once |
| mill, adax, sensibo | setpoint out of range rejected, not clamped or rounded |
| hoiax | response body in `msg` bounded |
| energy guard | `thing-charger` spec: the wrong "cliffhanger rejects under 6 A" statement, the rule for a current above the maximum; `get_schedule_report` failure |
| easee, zaptec | the same wrong "under 6 A" comment and spec text |
| outside this pass | zigbee-ad, ecollector, CloudBridge, fhbutler (free text in the value); mobile app defects (button `op_error` vs `error_text`, logout counts any reply as success, failed delete shown as timeout): written up for their owners |
