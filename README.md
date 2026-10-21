# Xolograph

Local-first forensic workbench for blockchain investigations (Ethereum mainnet).

- **Evidence vault** — files copied in read-only with SHA-256, re-verification, hash-chained chain of custody and audit log
- **On-chain explorer** — addresses and transactions retrieved live, each record with source, time and raw-payload hash
- **Graph** — value-flow graph from real activity, path finding, export; attach to cases with source snapshots
- **Entities** — labels that must state source, confidence and basis
- **Reports** — Markdown + JSON export, digests recorded in the audit log
- **Backup & restore** — verified single-archive backups (SQLite snapshot + vault + SHA-256 manifest); restore keeps previous data
- **Signed releases & auto-update** — optional, secret-driven; see `docs/RELEASING.md`

```
backend/     Go engine (xolographd): SQLite store, chain layer, loopback token-protected API
frontend/    React + TypeScript UI (npm workspace)
src-tauri/   Tauri 2 desktop shell (Rust)
scripts/     build-engine.mjs (compiles the Go engine as the Tauri sidecar)
```

## Develop
```
cd backend  && go mod tidy && go test ./... && go run ./cmd/xolographd --port 8787 --token dev --data-dir ./.data
npm install && npm run ui:dev                         # from the repo root; browser at http://localhost:3000
```

## Desktop app (Tauri 2)
Prerequisites: Node 20+, Go 1.22+, Rust (stable), and the [Tauri system dependencies](https://v2.tauri.app/start/prerequisites/).
```
npm install              # once, from the repo root
npm run desktop:dev      # builds the engine sidecar, then launches the app
npm run desktop:build    # installers in src-tauri/target/release/bundle
```
`npm run tauri dev` alone fails until `npm run engine:build` has produced `src-tauri/binaries/xolographd-<target>`.
To regenerate icons: `npm run tauri icon frontend/src/assets/xolograph.svg`.

## Lockfiles
On the first push to `main`, CI generates and commits `backend/go.sum` and `package-lock.json` (commit message has `[skip ci]`). Afterwards run `git pull --rebase` before pushing again.

## Security model
- Engine listens on `127.0.0.1` only, with a random per-run bearer token handed to the UI through the shell.
- The engine exits when the app closes (stdin watch), even after a crash.
- File dialogs are opened by the shell; the UI cannot write to arbitrary paths.
- Evidence and case data live in the OS app-data folder (shown in Settings); never commit it.

## Releases
Push a tag `vX.Y.Z` to build draft installers for Windows, macOS and Linux. Code-signing and signed auto-update turn on when you add the secrets described in [`docs/RELEASING.md`](docs/RELEASING.md); without them builds are unsigned.
