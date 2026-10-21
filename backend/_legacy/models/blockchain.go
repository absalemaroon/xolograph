package models

import "time"

type NetworkID string

const (
	NetworkEthereumMainnet NetworkID = "ethereum-mainnet"
	NetworkEthereumSepolia NetworkID = "ethereum-sepolia"
)

type DataProvenance struct {
	Network          NetworkID `json:"network"`
	DataSource       string    `json:"dataSource"`       // e.g. "INFURA_RPC", "LOCAL_NODE", "ETHERSCAN"
	Endpoint         string    `json:"endpoint"`         // RPC URL or API base (redacted secrets)
	RetrievedAt      time.Time `json:"retrievedAt"`
	QueryParameters  string    `json:"queryParameters"`
	NormalizationVer string    `json:"normalizationVer"`
	RawPayloadHash   string    `json:"rawPayloadHash"`   // SHA-256 of raw response
	CoverageLimit    string    `json:"coverageLimit"`    // e.g. "Last 10,000 blocks"
}

type TokenTransfer struct {
	TxHash          string `json:"txHash"`
	LogIndex        int64  `json:"logIndex"`
	ContractAddress string `json:"contractAddress"`
	TokenSymbol     string `json:"tokenSymbol,omitempty"`
	TokenDecimals   int    `json:"tokenDecimals,omitempty"`
	FromAddress     string `json:"fromAddress"`
	ToAddress       string `json:"toAddress"`
	RawValue        string `json:"rawValue"`        // Exact wei/raw integer
	FormattedValue  string `json:"formattedValue"`  // Human-readable decimal representation
}

type Transaction struct {
	Hash             string          `json:"hash"`
	Network          NetworkID       `json:"network"`
	BlockNumber      uint64          `json:"blockNumber"`
	BlockHash        string          `json:"blockHash"`
	BlockTimestamp   time.Time       `json:"blockTimestamp"`
	FromAddress      string          `json:"fromAddress"`
	ToAddress        string          `json:"toAddress"`
	Status           string          `json:"status"` // CONFIRMED, FAILED, PENDING
	ValueWei         string          `json:"valueWei"`
	ValueFormatted   string          `json:"valueFormatted"`
	GasUsed          uint64          `json:"gasUsed"`
	GasPriceWei      string          `json:"gasPriceWei"`
	TxFeeFormatted   string          `json:"txFeeFormatted"`
	TokenTransfers   []TokenTransfer `json:"tokenTransfers,omitempty"`
	Provenance       DataProvenance  `json:"provenance"`
}

type AddressProfile struct {
	Address             string         `json:"address"`
	Network             NetworkID      `json:"network"`
	FirstObservedAt     *time.Time     `json:"firstObservedAt,omitempty"`
	LastObservedAt      *time.Time     `json:"lastObservedAt,omitempty"`
	TxCountTotal        uint64         `json:"txCountTotal"`
	IncomingTxCount     uint64         `json:"incomingTxCount"`
	OutgoingTxCount     uint64         `json:"outgoingTxCount"`
	TotalETHReceivedWei string         `json:"totalETHReceivedWei"`
	TotalETHSentWei     string         `json:"totalETHSentWei"`
	Labels              []string       `json:"labels,omitempty"`
	Provenance          DataProvenance `json:"provenance"`
}