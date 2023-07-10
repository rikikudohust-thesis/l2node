package account

import (
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model/nonce"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iden3/go-iden3-crypto/babyjub"
)

type accountAPI struct {
	ItemID       uint64                `json:"itemID"`
	Idx          uint64                `json:"idx"`
	BatchNum     uint64                `json:"batchNum"`
	BJJ          babyjub.PublicKeyComp `json:"bjj"`
	EthAddr      common.Address        `json:"ethAddr"`
	TokenID      uint64                `json:"tokenID"`
	TokenItemID  uint64                `json:"tokenItemID"`
	TokenBlock   uint64                `json:"tokenBlock"`
	TokenEthAddr common.Address        `json:"tokenEthAddr"`
	Name         string                `json:"name"`
	Symbol       string                `json:"symbol"`
	Decimals     uint64                `json:"decimals"`
	Nonce        nonce.Nonce           `json:"nonce"`
	Balance      string                `json:"balance"`
	TotalItems   uint64                `json:"totalItems"`
}

type accountResponse struct {
	ItemID       uint64         `json:"itemID"`
	Idx          uint64         `json:"idx"`
	BatchNum     uint64         `json:"batchNum"`
	BJJ          string         `json:"bjj"`
	EthAddr      common.Address `json:"ethAddr"`
	TokenID      uint64         `json:"tokenID"`
	TokenItemID  uint64         `json:"tokenItemID"`
	TokenBlock   uint64         `json:"tokenBlock"`
	TokenEthAddr common.Address `json:"tokenEthAddr"`
	Name         string         `json:"name"`
	Symbol       string         `json:"symbol"`
	Decimals     uint64         `json:"decimals"`
	Nonce        nonce.Nonce    `json:"nonce"`
	Balance      string         `json:"balance"`
	TotalItems   uint64         `json:"totalItems"`
}
