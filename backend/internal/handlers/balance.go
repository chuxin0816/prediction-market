package handlers

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/prediction-market/backend/internal/models"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type BalanceHandler struct {
	db              *gorm.DB
	ethRPCURL       string
	contractAddress string
}

func NewBalanceHandler(db *gorm.DB, ethRPCURL, contractAddress string) *BalanceHandler {
	return &BalanceHandler{
		db:              db,
		ethRPCURL:       ethRPCURL,
		contractAddress: contractAddress,
	}
}

// SyncBalance reads the on-chain contract balance and syncs it to the database.
// It tracks a "synced_onchain" amount to compute the delta correctly:
//   DB.available += (onchain_now - last_synced_onchain)
// This way AMM trades (which only change DB) are preserved.
func (h *BalanceHandler) SyncBalance(c *gin.Context) {
	userAddress, ok := c.Get("user_address")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userAddr, ok := userAddress.(string)
	if !ok || userAddr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user address"})
		return
	}

	// Read on-chain balance via eth_call
	onchainBalance, err := h.readOnchainBalance(userAddr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read on-chain balance: " + err.Error()})
		return
	}

	// Convert from uint256 (6 decimals for USDC) to decimal
	onchainDec := decimal.NewFromBigInt(onchainBalance, -6)

	tx := h.db.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to start transaction"})
		return
	}

	// Get or create user balance with lock
	var balance models.UserBalance
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
		FirstOrCreate(&balance, models.UserBalance{UserAddress: userAddr}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Calculate delta: what changed on-chain since last sync
	// SyncedOnchain tracks the last known on-chain balance
	delta := onchainDec.Sub(balance.SyncedOnchain)

	if !delta.IsZero() {
		balance.Available = balance.Available.Add(delta)
		// Clamp to zero in case of rounding issues
		if balance.Available.IsNegative() {
			balance.Available = decimal.Zero
		}
		balance.SyncedOnchain = onchainDec

		if err := tx.Save(&balance).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// Log the balance change
		balanceLog := models.BalanceLog{
			UserAddress:  userAddr,
			ChangeType:   "sync",
			Amount:       delta,
			BalanceAfter: balance.Available,
		}
		if err := tx.Create(&balanceLog).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available":       balance.Available,
		"onchain_balance": onchainDec,
		"delta":           delta,
	})
}

// GetBalance returns the user's database trading balance.
func (h *BalanceHandler) GetBalance(c *gin.Context) {
	userAddress, ok := c.Get("user_address")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}
	userAddr, ok := userAddress.(string)
	if !ok || userAddr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid user address"})
		return
	}

	var balance models.UserBalance
	if err := h.db.FirstOrCreate(&balance, models.UserBalance{UserAddress: userAddr}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"available": balance.Available,
	})
}

// readOnchainBalance calls contract.balances(address) via eth_call JSON-RPC
func (h *BalanceHandler) readOnchainBalance(address string) (*big.Int, error) {
	// balances(address) selector = keccak256("balances(address)")[:4] = 0x27e235e3
	addr := strings.TrimPrefix(strings.ToLower(address), "0x")
	// Pad address to 32 bytes
	paddedAddr := fmt.Sprintf("%064s", addr)
	callData := "0x27e235e3" + paddedAddr

	rpcReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "eth_call",
		"params": []interface{}{
			map[string]string{
				"to":   h.contractAddress,
				"data": callData,
			},
			"latest",
		},
		"id": 1,
	}

	body, err := json.Marshal(rpcReq)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	resp, err := http.Post(h.ethRPCURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("rpc call: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var rpcResp struct {
		Result string `json:"result"`
		Error  *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}
	if rpcResp.Error != nil {
		return nil, fmt.Errorf("rpc error: %s", rpcResp.Error.Message)
	}

	// Parse hex result
	resultHex := strings.TrimPrefix(rpcResp.Result, "0x")
	if resultHex == "" || resultHex == "0x" {
		return big.NewInt(0), nil
	}

	resultBytes, err := hex.DecodeString(resultHex)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}

	return new(big.Int).SetBytes(resultBytes), nil
}
