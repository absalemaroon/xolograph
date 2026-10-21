package models

import "time"

type TransferDirection string

const (
	DirectionIncoming TransferDirection = "INCOMING"
	DirectionOutgoing TransferDirection = "OUTGOING"
	DirectionSelf     TransferDirection = "SELF"
)

type TransactionFilter struct {
	Address         string            `json:"address"`
	Network         NetworkID         `json:"network"`
	Direction       TransferDirection `json:"direction,omitempty"` // INCOMING, OUTGOING
	MinEthValue     float64           `json:"minEthValue,omitempty"`
	TokenType       string            `json:"tokenType,omitempty"`   // "NATIVE", "ERC20", "ALL"
	FromBlock       uint64            `json:"fromBlock,omitempty"`
	ToBlock         uint64            `json:"toBlock,omitempty"`
	Page            int               `json:"page"`
	PageSize        int               `json:"pageSize"`
}

type PaginatedTransactions struct {
	Address        string         `json:"address"`
	TotalCount     int            `json:"totalCount"`
	Page           int            `json:"page"`
	PageSize       int            `json:"pageSize"`
	HasMore        bool           `json:"hasMore"`
	Transactions   []Transaction  `json:"transactions"`
	Provenance     DataProvenance `json:"provenance"`
	CoverageNote   string         `json:"coverageNote"`
}