# prime: GetAttribute*Value relies on unspecified return evaluation order

`prime/type.go:588-658` — each helper is `return val, d.GetAttributeObjectValue(..., &val)`. The Go
spec leaves the order between evaluating the plain operand `val` and the call unspecified; gc happens
to run the call first, so the decoded value is returned. A compiler change (or gccgo) could return
the zero value with a valid timestamp. `prime/type_test.go` (TestStateDevice_GetAttributeValue) would
catch it, so this is latent, not live.

Close by assigning the time first (`t := d.GetAttributeObjectValue(...); return val, t`).
