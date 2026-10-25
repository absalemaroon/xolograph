# Portfolio and flows (Phase 6)

## What it does
- **Holdings and value** (Explorer address page, any network): native balance plus token balances, each valued at a market
  reference price when one exists. EVM tokens come from the network's Blockscout (`/token-balances`), Solana SPL tokens from
  `getTokenAccountsByOwner` (Token and Token-2022 programs, summed per mint), Bitcoin is native only.
- **Entity portfolio** (Entities > Portfolio tab): every non-rejected wallet is valued separately (max 12, one after another).
  Totals are reported **by attribution strength**: "verified + publicly reported" and "all non-rejected claims" are different
  numbers on purpose, so a suspected wallet never silently inflates what an entity "holds".
- **Net flows (loaded window)** (Explorer address page): incoming and outgoing native-asset movements over the most recent 1-4
  pages of activity, each valued at the historical price nearest its time.
- Any result can be attached to a case as a finding with its full source snapshot.

## What it deliberately does not do: profit and loss
There is no realized/unrealized P&L figure. A correct one needs a complete history including fees, internal and
contract-initiated transfers, token swaps and everything before the first transaction, plus a stated cost-basis method.
The data sources used here do not guarantee that (Blockscout may omit internal transactions, Solana activity is derived from
parsed instructions, only recent pages are loaded). A number that looks like P&L but silently omits those would mislead, so the
flows panel is named "net flows", lists exactly what it excludes, and says whether the full history was loaded.

## Rules the code enforces
- A missing price is **unpriced**, never zero. Unpriced assets are listed and excluded from totals, with a count.
- A price is used for a movement only if it is within 36 hours of that movement; otherwise the movement is unvalued (native totals still include it).
- Spam or unlisted tokens normally have no reference price and so are unpriced; contract-supplied token names are labelled unverified.
- Prices are third-party aggregated references, not execution prices; each response carries source, timestamp and payload hash.
- A failed price lookup degrades the result (unpriced + a note); it never fails the whole request.

## Configuration
- `XOLOGRAPH_PRICES=off` disables valuation (holdings are still listed, all unpriced).
- `XOLOGRAPH_PRICE_URL` (default `https://api.coingecko.com/api/v3`) and `XOLOGRAPH_PRICE_KEY` (optional CoinGecko demo/pro key).
- The public CoinGecko API is rate limited; spot responses are cached for 60 seconds. Coin ids and token platform ids per
  network are in `backend/pkg/prices/prices.go` (written from documentation; not exercised from the authoring sandbox).
