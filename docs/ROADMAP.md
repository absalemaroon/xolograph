# Xolograph roadmap (from the v3 master prompt)

| Phase | Scope | State |
|---|---|---|
| 1 | Audit, remove dead `_legacy` code, document direction | Done (audit in chat; legacy removed) |
| 2 | Entity foundation: multi-wallet, attribution status/history, identifiers, linked evidence, profile UI, explorer attributions | Written; run CI to verify |
| 3 | Entity wallet graph: multi-seed, entity labels on nodes, filters, inspector, saved views | Written; run CI to verify |
| 4 | Case relationship board (facts vs hypotheses, evidence-backed edges) | Planned |
| 5 | Multi-chain explorer, universal search, provider adapters | Planned |
| 6 | Portfolio and qualified P&L | Planned |
| 7 | Tools: file and metadata inspector | Planned |
| 8 | Hardening, packaging, docs | Planned |

## Entity model (v3 migration)
- `entities`: profile + identity-level attribution status and confidence.
- `entity_addresses`: one claim per (entity, network, address) with status, confidence, source, basis; `entity_address_history` records every status change.
- `entity_identifiers`: public handles/URLs; default UNVERIFIED; VERIFIED/PUBLICLY_LISTED need a source.
- `entity_evidence`: links to existing vault items (no copies).
- Strong claims (VERIFIED, PUBLICLY_REPORTED) require a source; INVESTIGATOR_ATTRIBUTED requires a basis or source.
- v1 `entity_labels` are untouched. They are copied into entities grouped by identical name; a "verified source" label becomes PUBLICLY_REPORTED, never VERIFIED.
- Chains: EVM networks and Solana can be recorded; only Ethereum mainnet has an explorer/graph adapter.

## Graph (v4 migration, Phase 3)
- Multi-seed graphs (up to 10). An entity's "Graph wallets" button seeds the graph with all its queryable wallets.
- Entity names are looked up live from local records (`POST /api/entities/lookup`). They are never derived from chain data. The strongest non-rejected claim labels a node; rejected claims never do.
- Filters (minimum transfers, ETH/token) hide data on screen only.
- Saved views (`graph_snapshots`): positions plus the aggregates loaded at save time. Restoring does not re-query the chain and is read-only (expanding would double-count transfers); entity names are refreshed.
