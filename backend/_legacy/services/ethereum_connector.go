package services

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/xolograph/backend/pkg/models"
)

type EthereumConnector struct {
	rpcEndpoint string
	httpClient  *http.Client
}

func NewEthereumConnector(rpcEndpoint string) *EthereumConnector {
	if rpcEndpoint == "" {
		rpcEndpoint = "https://cloudflare-eth.com"
	}
	return &EthereumConnector{
		rpcEndpoint: rpcEndpoint,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// FormatWeiToETH converts raw wei string into human-readable ETH string
func FormatWeiToETH(weiStr string) string {
	wei, ok := new(big.Int).SetString(weiStr, 10)
	if !ok || wei == nil {
		return "0.000000 ETH"
	}
	fWei := new(big.Float).SetInt(wei)
	ethDiv := new(big.Float).SetFloat64(1e18)
	fEth := new(big.Float).Quo(fWei, ethDiv)
	return fmt.Sprintf("%.6f ETH", fEth)
}

func (c *EthereumConnector) SendJSONRPC(method string, params []interface{}) (json.RawMessage, string, error) {
	reqBody := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, "", err
	}

	// Compute payload hash
	hash := sha256.Sum256(jsonBytes)
	rawHash := hex.EncodeToString(hash[:])

	resp, err := c.httpClient.Post(c.rpcEndpoint, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, rawHash, fmt.Errorf("RPC network request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, rawHash, err
	}

	var rpcResp struct {
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &rpcResp); err != nil {
		return nil, rawHash, err
	}
	if rpcResp.Error != nil {
		return nil, rawHash, fmt.Errorf("RPC Error %d: %s", rpcResp.Error.Code, rpcResp.Error.Message)
	}

	return rpcResp.Result, rawHash, nil
}

// GetTransactionByHash fetches and normalizes an Ethereum transaction into domain model
func (c *EthereumConnector) GetTransactionByHash(txHash string) (*models.Transaction, error) {
	if !strings.HasPrefix(txHash, "0x") || len(txHash) != 66 {
		return nil, fmt.Errorf("invalid Ethereum transaction hash format: %s", txHash)
	}

	resultRaw, payloadHash, err := c.SendJSONRPC("eth_getTransactionByHash", []interface{}{txHash})
	if err != nil {
		return nil, err
	}

	if string(resultRaw) == "null" {
		return nil, fmt.Errorf("transaction not found on network: %s", txHash)
	}

	var rawTx struct {
		Hash        string `json:"hash"`
		BlockNumber string `json:"blockNumber"`
		BlockHash   string `json:"blockHash"`
		From        string `json:"from"`
		To          string `json:"to"`
		Value       string `json:"value"`
		Gas         string `json:"gas"`
		GasPrice    string `json:"gasPrice"`
	}

	if err := json.Unmarshal(resultRaw, &rawTx); err != nil {
		return nil, fmt.Errorf("failed to parse RPC transaction payload: %w", err)
	}

	// Hex to decimal conversions
	blockNum, _ := strconv.ParseUint(strings.TrimPrefix(rawTx.BlockNumber, "0x"), 16, 64)
	valBig := new(big.Int)
	valBig.SetString(strings.TrimPrefix(rawTx.Value, "0x"), 16)
	gasPriceBig := new(big.Int)
	gasPriceBig.SetString(strings.TrimPrefix(rawTx.GasPrice, "0x"), 16)

	normalized := &models.Transaction{
		Hash:           rawTx.Hash,
		Network:        models.NetworkEthereumMainnet,
		BlockNumber:    blockNum,
		BlockHash:      rawTx.BlockHash,
		BlockTimestamp: time.Now(), // Hydrated when block metadata fetched
		FromAddress:    strings.ToLower(rawTx.From),
		ToAddress:      strings.ToLower(rawTx.To),
		Status:         "CONFIRMED",
		ValueWei:       valBig.String(),
		ValueFormatted: FormatWeiToETH(valBig.String()),
		GasPriceWei:    gasPriceBig.String(),
		Provenance: models.DataProvenance{
			Network:          models.NetworkEthereumMainnet,
			DataSource:       "ETHEREUM_JSON_RPC",
			Endpoint:         c.rpcEndpoint,
			RetrievedAt:      time.Now(),
			QueryParameters:  fmt.Sprintf("eth_getTransactionByHash(%s)", txHash),
			NormalizationVer: "1.0.0",
			RawPayloadHash:   payloadHash,
			CoverageLimit:    "Single Transaction Query",
		},
	}

	return normalized, nil
}