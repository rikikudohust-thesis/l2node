package model

import "github.com/ethereum/go-ethereum/common"

type TransactionResponse struct {
	ID            TxID           `json:"id"`
	TxHash        TxID           `json:"txHash"`
	Type          bool           `json:"type"`
	TxType        TxType         `json:"txType"`
	State         PoolL2TxState  `json:"state"`
	FromEthAddr   common.Address `json:"fromEthAddr"`
	Sender        common.Address `json:"sender"`
	Receiver      common.Address `json:"receiver"`
	ToIdx         Idx            `json:"toIdx"`
	TokenID       TokenID        `json:"token"`
	DepositAmount string         `json:"depositAmount"`
	Amount        string         `json:"amount"`
	Batch         BatchNum       `json:"batch"`
	L1Batch       BatchNum       `json:"l1Batch"`
}
