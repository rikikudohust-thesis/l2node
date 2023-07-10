package model

import (
	"time"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

// tokenIDBytesLen defines the length of the TokenID byte array representation
const tokenIDBytesLen = 4

// Token is a struct that represents an Ethereum token that is supported in Hermez network
type Token struct {
	TokenID     TokenID           `json:"token_id"`
	EthBlockNum int64             `json:"ethereumBlockNum" `
	EthAddr     ethCommon.Address `json:"ethereumAddress" `
	Name        string            `json:"name" `
	Symbol      string            `json:"symbol"`
	Decimals    uint64            `json:"decimals"`
}

// TokenInfo provides the price of the token in USD
type TokenInfo struct {
	TokenID     uint32
	Value       float64
	LastUpdated time.Time
}
