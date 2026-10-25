# Xolograph roadmap (from the v3 master prompt)

| Phase | Scope | State |
|---|---|---|
| 1 | Audit, remove dead `_legacy` code, document direction | Done (audit in chat; legacy removed) |
| 2 | Entity foundation: multi-wallet, attribution status/history, identifiers, linked evidence, profile UI, explorer attributions | Written; run CI to verify |
| 3 | Entity wallet graph: multi-seed, entity labels on nodes, filters, inspector, saved views | Written; run CI to verify |
| 4 | Case relationship board (facts vs hypotheses, evidence-backed links) | Written; run CI to verify |
| 5 | Multi-chain explorer (EVM family, Solana, Bitcoin), universal search, provider registry | Written; run CI to verify (see docs/NETWORKS.md) |
| 6 | Portfolio, valuation and qualified flows (no P&L figure; see docs/PORTFOLIO.md) | Written; run CI to verify |
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

## Case board (v5 migration, Phase 4)
- Per-case canvas under the case's **Board** tab. Items: entity, wallet, evidence item, note, event. Referenced records are shown live and never copied.
- Links are drawn only by a person. Chain activity never creates a link.
- Every link is a **FACT** (solid line) or a **HYPOTHESIS** (dashed, marked "(?)"). A fact needs at least one linked evidence item from the same case or a stated source; the last support cannot be removed from a fact, and a link cannot be promoted without support.
- Layout (positions) is saved per item and is not audited; adding, editing and removing items, links and evidence links are audited.
- Evidence whose integrity check failed is flagged on the board.
