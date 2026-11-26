## ADDED Requirements

### Requirement: Phase Mode Capabilities
`utils.SupportedPhaseModes(grid, utilizedPhases, phaseMode)` SHALL return the phase modes a charger
wired with `utilizedPhases` can switch between: for a TN grid with 3 utilized phases `NL1`, `NL2`,
`NL3` and `NL1L2L3`; for an IT or TT grid with 3 utilized phases `L1L2`, `L2L3`, `L3L1` and
`L1L2L3`; for a TN, IT or TT grid with any other count only the given `phaseMode`; and
`[PhaseModeUnknown]` for any other grid type. `types.PhaseMode.Phases()` SHALL return the lines a
mode uses (`L3L1` yields `L1`, `L3`), and `EffectivePhasesCnt()` SHALL return 3 for `NL1L2L3` and
`L1L2L3`, 2 for `NL1L2` and `NL2L3`, and 1 for the single-line neutral modes and the two-line
phase-to-phase modes. An unknown mode SHALL be treated as three-phase by both: all three lines and
a count of 3.

#### Scenario: three-phase charger on an IT grid
- **WHEN** `SupportedPhaseModes(GridTypeIT, 3, PhaseModeL1L2L3)` is called
- **THEN** `L1L2`, `L2L3`, `L3L1` and `L1L2L3` are returned

#### Scenario: phase-to-phase mode counts one effective phase
- **WHEN** `EffectivePhasesCnt` is called on `PhaseModeL1L2`
- **THEN** 1 is returned, while `Phases` returns `L1` and `L2`
