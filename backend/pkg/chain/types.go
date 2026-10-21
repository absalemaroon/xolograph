// Package chain retrieves and normalises real blockchain data (Ethereum mainnet).
// Every record carries provenance: source, retrieval time and a SHA-256 of the raw response.
package chain

import "time"

type Provenance struct {
	Source        string    `json:"source"` // ETHEREUM_JSON_RPC | BLOCKSCOUT_V2
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
	BalanceWei       string            `json:"balanceWei"`
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
	Notes           []string        `json:"notes"`
	Provenance      []Provenance    `json:"provenance"`
}
