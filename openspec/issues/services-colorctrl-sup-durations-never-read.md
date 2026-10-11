# colorctrl SupportedDurations always returns nil

`colorctrl.Specification` stores `sup_durations` as a `map[string]int`
(`adapter/service/colorctrl/specification.go:32`), but `SupportedDurations()`
(`colorctrl/service.go:143`) reads it with `PropertyIntegers`, which expects a `[]int` and returns nil
when the JSON cast from a map fails. The exported getter can never return the configured durations.

Close by deciding the intended shape of `sup_durations` (map of component → duration, or list) and
making the getter and its return type match; add a test. `SupportedDurations` is currently untested
(0% coverage).
