package services

import (
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/xolograph/backend/pkg/models"
)

type TransactionExplorerService struct {
	ethConnector *EthereumConnector
}

func NewTransactionExplorerService(ethConnector *EthereumConnector) *TransactionExplorerService {
	return &TransactionExplorerService{ethConnector: ethConnector}
}

// FilterAndPaginateTransactions applies direction, value, token, and page boundaries over normalized transaction sets
func (s *TransactionExplorerService) FilterAndPaginateTransactions(
	allTxs []models.Transaction,
	targetAddress string,
	filter models.TransactionFilter,
) models.PaginatedTransactions {
	cleanTarget := strings.ToLower(targetAddress)
	var filtered []models.Transaction

	for _, tx := range allTxs {
		fromMatch := strings.ToLower(tx.FromAddress) == cleanTarget
		toMatch := strings.ToLower(tx.ToAddress) == cleanTarget

		// 1. Direction Filter
		if filter.Direction == models.DirectionIncoming && !toMatch {
			continue
		}
		if filter.Direction == models.DirectionOutgoing && !fromMatch {
			continue
		}

		// 2. Value Threshold Filter (ETH)
		if filter.MinEthValue > 0 {
			valWei := new(big.Int)
			valWei.SetString(tx.ValueWei, 10)
			fWei := new(big.Float).SetInt(valWei)
			ethDiv := new(big.Float).SetFloat64(1e18)
			fEth := new(big.Float).Quo(fWei, ethDiv)
			ethFloat, _ := fEth.Float64()

			if ethFloat < filter.MinEthValue {
				continue
			}
		}

		// 3. Token Type Filter
		if filter.TokenType == "ERC20" && len(tx.TokenTransfers) == 0 {
			continue
		}
		if filter.TokenType == "NATIVE" && len(tx.TokenTransfers) > 0 {
			continue
		}

		filtered = append(filtered, tx)
	}

	// Pagination Math
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 {
		pageSize = 10
	}

	total := len(filtered)
	startIdx := (page - 1) * pageSize
	endIdx := startIdx + pageSize

	var paged []models.Transaction
	hasMore := false

	if startIdx < total {
		if endIdx > total {
			endIdx = total
		} else {
			hasMore = endIdx < total
		}
		paged = filtered[startIdx:endIdx]
	}

	return models.PaginatedTransactions{
		Address:      targetAddress,
		TotalCount:   total,
		Page:         page,
		PageSize:     pageSize,
		HasMore:      hasMore,
		Transactions: paged,
		Provenance: models.DataProvenance{
			Network:          filter.Network,
			DataSource:       "LOCAL_INDEXER_RPC",
			Endpoint:         s.ethConnector.rpcEndpoint,
			RetrievedAt:      time.Now(),
			QueryParameters:  fmt.Sprintf("address=%s&page=%d&size=%d", targetAddress, page, pageSize),
			NormalizationVer: "1.0.0",
			RawPayloadHash:   "paginated_dataset_provenance_hash",
			CoverageLimit:    "Query window limited by node RPC state archive limits.",
		},
		CoverageNote: "Notice: Historical depth is bounded by configured archive RPC node limits.",
	}
}