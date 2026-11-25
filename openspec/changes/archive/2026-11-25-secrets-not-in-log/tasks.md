## 1. Secrets load error

- [x] 1.1 Failing test: a secrets file that does not unmarshal and holds a fake secret (cut short, number overflow, `,string` field, `time.Time` field) gives a `Load` error without the secret, naming the path, the kind and the size
- [x] 1.2 Failing test: the backup fallback log line does not carry the secret
- [x] 1.3 `loadFile` builds the secrets error without the `encoding/json` text
- [x] 1.4 `go build`, `go vet`, `golangci-lint run`, storage tests
