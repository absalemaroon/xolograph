# Xolograph — status

| Area | State |
|---|---|
| Engine: SQLite store, cases, notes, findings, evidence vault, hash-chained custody + audit | Stage 1 |
| Frontend foundation: design system, theme, cases, vault, audit, settings | Stage 2 |
| Chain layer: Ethereum JSON-RPC (balance, tx, receipt, token transfers) + Blockscout history/labels, all with provenance | Stage 3 (written, unverified against live APIs) |
| Entity labels (source + confidence + basis required), report export (Markdown + JSON with SHA-256, audited) | Stage 3 |
| Explorer, Graph (Sigma, expand, path, export), Entities, Reports screens | Stage 3 |
| Tauri shell, native file picker/save dialog, installers | Stage 4 |

## Verify (nothing here was compiled in the authoring sandbox)
```
cd backend && go mod tidy && go vet ./... && go test ./...
cd frontend && npm install && npm run build && npm test
```

## Run
```
cd backend && go run ./cmd/xolographd --port 8787 --token dev --data-dir ./.data
#   optional: --rpc-url <your node> --history-url <blockscout base url>   (or XOLOGRAPH_RPC_URL / XOLOGRAPH_HISTORY_URL)
cd frontend && npm run dev     # http://localhost:3000
```

## Known limitations
- Address history/labels come from a third-party index (Blockscout v2); response field names were written from its documented shape and are covered only by mock tests until run against the live service.
- Graph expansion loads the most recent page of activity per node, not full history.
- Browser-style downloads and typed file paths are replaced by native dialogs in Stage 4.
- `backend/_legacy` and `frontend/src/_legacy` hold superseded Gemini-era code (mock data); excluded from builds.
