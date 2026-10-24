# Xolograph — status

| Area | State |
|---|---|
| Engine: SQLite store, cases, notes, findings, evidence vault, hash-chained custody + audit | Stage 1 |
| Frontend foundation: design system, theme, cases, vault, audit, settings | Stage 2 |
| Chain layer: Ethereum JSON-RPC (balance, tx, receipt, token transfers) + Blockscout history/labels, all with provenance | Stage 3 (written, unverified against live APIs) |
| Entity labels (source + confidence + basis required), report export (Markdown + JSON with SHA-256, audited) | Stage 3 |
| Explorer, Graph (Sigma, expand, path, export), Entities, Reports screens | Stage 3 |
| Tauri 2 shell: engine sidecar lifecycle, native open/save dialogs, CSP, icons, installers config | Stage 4 (written, unverified) |
| GitHub Actions: CI (go vet/test, npm build/test, `cargo check`) and tagged desktop releases | Stage 4 |
| Entities (v3): multi-wallet records, per-wallet attribution status + history, public identifiers, linked vault evidence; v1 labels migrated conservatively | Phase 2 (written, unverified: no Go toolchain in the authoring sandbox) |
| Graph (v4): multi-seed, entity labels, filters, saved views | Phase 3 (written, unverified) |
| Case board (v5): items, facts vs hypotheses, evidence-backed links | Phase 4 (written, unverified) |
| Multi-chain Explorer (EVM x10, Solana, Bitcoin) + universal search: see docs/NETWORKS.md | Phase 5 (written, unverified; default endpoints never hit from the sandbox) |
| Backup & restore (verified archive, staged restore, pre-restore copy, audited) | Stage 5 |
| Signed auto-update (opt-in feature + secrets), Windows/macOS signing wiring | Stage 5 (needs your keys/certificates) |

## Verify (nothing here was compiled in the authoring sandbox)
```
cd backend && go mod tidy && go vet ./... && go test ./...
npm install && npm run ui:build && npm test      # from the repo root (npm workspace)
# desktop (needs Rust + Tauri prerequisites):
npm run desktop:dev
```
CI runs the first two automatically on every push.

## Run
```
cd backend && go run ./cmd/xolographd --port 8787 --token dev --data-dir ./.data
#   optional: --rpc-url <your node> --history-url <blockscout base url>   (or XOLOGRAPH_RPC_URL / XOLOGRAPH_HISTORY_URL)
npm run ui:dev     # from the repo root; http://localhost:3000
```

## Known limitations
- Address history/labels come from a third-party index (Blockscout v2); response field names were written from its documented shape and are covered only by mock tests until run against the live service.
- Graph expansion loads the most recent page of activity per node, not full history.
- Installers are unsigned until you add signing secrets (docs/RELEASING.md). Evidence stored by earlier versions keeps absolute paths; they are re-based automatically on restore.
- `tauri-plugin-*` / `tauri` APIs were written for Tauri 2 from documentation and not compiled here.
