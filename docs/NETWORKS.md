# Supported networks and data sources (Phase 5)

| Network | Family | Native | State / transactions | History + labels |
|---|---|---|---|---|
| Ethereum (`ethereum-mainnet`) | EVM | ETH | JSON-RPC (`--rpc-url`) | Blockscout v2 (`--history-url`) |
| Base, Optimism, Arbitrum One, Polygon, Gnosis, Scroll, Celo | EVM | ETH / POL / xDAI / CELO | JSON-RPC | Blockscout v2 |
| BNB Smart Chain, Avalanche C-Chain | EVM | BNB / AVAX | JSON-RPC | none by default (no history tab) |
| Solana (`solana`) | SOLANA | SOL | Solana JSON-RPC (`getBalance`, `getAccountInfo`, `getSignaturesForAddress`, `getTransaction`) | derived from parsed instructions, latest 20 transactions per page |
| Bitcoin (`bitcoin`) | BITCOIN | BTC | Esplora REST (mempool.space-compatible) | net effect per transaction, 25 per page |

The default endpoints are public conveniences chosen from documentation; they were not exercised from the authoring sandbox
(no network). Override or disable any of them:

- `XOLOGRAPH_RPC_<ID>` — JSON-RPC endpoint (Esplora base URL for Bitcoin). Set to `off` to disable the network.
- `XOLOGRAPH_HISTORY_<ID>` — Blockscout v2 base URL for EVM history and labels. Set to `off` for none.
- `<ID>` is the network id upper-cased with `-` as `_`, e.g. `XOLOGRAPH_RPC_BASE`, `XOLOGRAPH_HISTORY_BSC`, `XOLOGRAPH_RPC_BITCOIN`.
- Ethereum keeps its flags/env: `--rpc-url` / `XOLOGRAPH_RPC_URL`, `--history-url` / `XOLOGRAPH_HISTORY_URL`.

## Honest limits
- **Solana activity is derived**, not indexed: only parsed System-program and SPL-token transfers in the latest transactions that
  reference the address. Transfers inside other programs, and balance changes that are not transfers, can be missing. SPL
  tokens show their mint address; names are not resolved. Each page costs one RPC call per transaction.
- **Bitcoin is UTXO-based**: an activity row is the net effect of one transaction on the address; the counterparty is the first
  other input/output address and is indicative only. No Ordinals/Runes/BRC-20.
- **No third-party labels** for Solana and Bitcoin.
- Public RPC nodes rate-limit; use your own endpoint for heavy work.
- The value-flow **graph is still Ethereum-mainnet only**. Entities, the case board and the Explorer work on every network.
- The same EVM address exists on every EVM chain; entity attributions are scoped per network and never carry over.

## Universal search
One box detects EVM addresses/hashes, Solana addresses/signatures and Bitcoin addresses/txids, and also searches your own
entities, cases, evidence (name or hash) and case labels (`GET /api/search`). Strings valid on several networks (any EVM
chain; rare Bitcoin/Solana overlaps) ask you to pick the network instead of guessing.
