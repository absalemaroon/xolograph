// Package chain retrieves and normalises real blockchain data (EVM networks, Solana, Bitcoin).
// Every record carries provenance: source, retrieval time and a SHA-256 of the raw response.
package chain

import "time"

type Provenance struct {
	Source        string    `json:"source"` // ETHEREUM_JSON_RPC | BLOCKSCOUT_V2 | SOLANA_JSON_RPC | ESPLORA_REST
	Endpoint      string    `json:"endpoint"`
	Query         string    `json:"query"`
	RetrievedAt   time.Time `json:"retrievedAt"`
	PayloadSHA256 string    `json:"payloadSha256"`
	Coverage      string    `json:"coverage"`
}

type ThirdPartyLabel struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`   // CONTRACT_NAME | PUBLIC_TAG
	Source string `json:"source"` // always a third party; never an Xolograph assertion
}

type AddressSummary struct {
	Address          string            `json:"address"`
	Network          string            `json:"network"`
	NativeSymbol     string            `json:"nativeSymbol"`
	NativeDecimals   int               `json:"nativeDecimals"`
	BalanceWei       string            `json:"balanceWei"` // smallest native unit (wei, lamports, satoshi)
	Nonce            uint64            `json:"nonce"`
	IsContract       bool              `json:"isContract"`
	ThirdPartyLabels []ThirdPartyLabel `json:"thirdPartyLabels"`
	Notes            []string          `json:"notes"`
	Provenance       []Provenance      `json:"provenance"`
}

// Transfer is a normalised value movement: a native transaction or a token transfer event.
type Transfer struct {
	TxHash       string    `json:"txHash"`
	LogIndex     int64     `json:"logIndex"` // -1 for native transfers
	Kind         string    `json:"kind"`     // NATIVE | ERC20 | ERC721 | ERC1155
	From         string    `json:"from"`
	To           string    `json:"to"` // empty for contract creation
	Value        string    `json:"value"`
	Decimals     int       `json:"decimals"` // -1 when unknown
	Symbol       string    `json:"symbol"`
	TokenAddress string    `json:"tokenAddress,omitempty"`
	TokenID      string    `json:"tokenId,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
	BlockNumber  uint64    `json:"blockNumber"`
	Status       string    `json:"status"` // SUCCESS | FAILED | UNKNOWN
}

type ActivityPage struct {
	Address    string     `json:"address"`
	Kind       string     `json:"kind"` // native | token
	Items      []Transfer `json:"items"`
	NextCursor string     `json:"nextCursor"`
	Provenance Provenance `json:"provenance"`
}

type TokenTransfer struct {
	LogIndex int64  `json:"logIndex"`
	Standard string `json:"standard"` // ERC20 | ERC721
	Token    string `json:"token"`
	Symbol   string `json:"symbol"`
	Decimals int    `json:"decimals"` // -1 when unknown
	From     string `json:"from"`
	To       string `json:"to"`
	Value    string `json:"value"`
	TokenID  string `json:"tokenId,omitempty"`
}

type Transaction struct {
	Hash            string          `json:"hash"`
	Network         string          `json:"network"`
	NativeSymbol    string          `json:"nativeSymbol"`
	NativeDecimals  int             `json:"nativeDecimals"`
	Status          string          `json:"status"` // SUCCESS | FAILED | PENDING | UNKNOWN
	BlockNumber     uint64          `json:"blockNumber"`
	BlockHash       string          `json:"blockHash,omitempty"`
	Timestamp       *time.Time      `json:"timestamp,omitempty"`
	From            string          `json:"from"`
	To              string          `json:"to"`
	ContractCreated string          `json:"contractCreated,omitempty"`
	ValueWei        string          `json:"valueWei"`
	GasUsed         uint64          `json:"gasUsed"`
	GasPriceWei     string          `json:"gasPriceWei"`
	FeeWei          string          `json:"feeWei"`
	TokenTransfers  []TokenTransfer `json:"tokenTransfers"`
	Inputs          []TxIO          `json:"inputs,omitempty"`  // UTXO chains
	Outputs         []TxIO          `json:"outputs,omitempty"` // UTXO chains
	Notes           []string        `json:"notes"`
	Provenance      []Provenance    `json:"provenance"`
}

// TxIO is one input or output of a UTXO transaction.
type TxIO struct {
	Address string `json:"address"` // empty for non-standard scripts (e.g. OP_RETURN)
	Value   string `json:"value"`   // satoshi
	Type    string `json:"type,omitempty"`
}

// NetworkInfo describes one supported network and what its adapter can do.
type NetworkInfo struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Family         string `json:"family"` // EVM | SOLANA | BITCOIN
	NativeSymbol   string `json:"nativeSymbol"`
	NativeDecimals int    `json:"nativeDecimals"`
	HasHistory     bool   `json:"hasHistory"` // address activity available
	HasLabels      bool   `json:"hasLabels"`  // third-party labels available
	Endpoint       string `json:"endpoint"`   // scheme://host only
}
