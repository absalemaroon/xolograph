# Xolograph — verified status

| Area | State |
|---|---|
| Engine: SQLite store, migrations, cases, notes, findings | Written (Stage 1) — run `go mod tidy && go test ./...` |
| Engine: evidence vault (SHA-256, re-hash, custody + audit hash chains) | Written (Stage 1) |
| Engine: loopback token-auth HTTP API + `xolographd` sidecar | Written (Stage 1) |
| Ethereum connector / transaction service | Pre-existing, not yet wired to API (Stage 3) |
| Frontend foundation, theme, missing views | Stage 2 |
| Graph, entities, report export on real data | Stage 3 |
| Tauri shell, installers | Stage 4 |

Run engine (dev): `cd backend && go run ./cmd/xolographd --port 8787 --token dev --data-dir ./.data`
