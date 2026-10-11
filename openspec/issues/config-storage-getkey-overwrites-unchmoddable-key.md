# security: GetKey regenerates (truncates) an existing key it cannot chmod, and never validates it

`security.GetKey` (security/encryption.go:15) falls back to `writeKeyFile` on *any*
`readKeyFromFile` error, including a failed `os.Chmod` of a key file that was read successfully
(encryption.go:34). `writeKeyFile` then `os.Create`s the path — truncating the existing key — before
its own chmod, so a key file the process can write but not chmod (e.g. owned by another user after a
manual restore) is wiped and everything encrypted with it becomes undecryptable.

The read path also returns the file content verbatim: an empty file (crash between `os.Create` and
`WriteString`) or a trailing newline from a hand edit yields a key that makes every
`Encrypt`/`Decrypt` fail, with no regeneration or error at `GetKey`.

Close by regenerating only when the file does not exist (`errors.Is(err, fs.ErrNotExist)`), returning
chmod errors, and validating the key (hex, 16/24/32 bytes) on read; then extend the Encryption Key
Management requirement and its tests.
