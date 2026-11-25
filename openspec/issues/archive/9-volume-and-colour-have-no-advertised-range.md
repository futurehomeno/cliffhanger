# Media player volume and colour components are not checked against a range

Rule 6 of the error report policy rejects a value outside the range a service advertises.
`mediaplayer` advertises no volume range and `SetVolume` passes any integer to the controller;
`colorctrl` passes the colour map on unchecked: no range for a component value, and a component
missing from the advertised `sup_components` is not rejected either. So `out_of_range` is never
answered by these two services, and a volume of 500, a `red` of 999 or a component the light does
not list reaches the controller. Found while coding the validation errors (`error-report-policy`); no check was added,
because none exists today and no range is advertised.

Reproduce: `cmd.volume.set` with 500 on a media player; `cmd.color.set` with `{"red": 999}`.

Closes when: it is decided whether the two services check a fixed range (volume 0–100, colour
0–255) and answer `out_of_range`, whether `colorctrl` answers `not_supported` for a component outside
`sup_components`, or the controller stays the only judge.

Resolved (2026-11-25, user decision): `colorctrl` rejects a component missing from
`sup_components` with `not_supported` before the controller is called; component values get no
fixed range (a colour temperature is not 0–255). `mediaplayer` rejects a volume outside 0–100 with
`out_of_range`, because the FIMP media player page states "Sets playback volume within 0-100 range."
The spec delta requirement "A Value Outside The Advertised Range Is Rejected" carries both.
