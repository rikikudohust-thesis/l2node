package model

import (
	ethCommon "github.com/ethereum/go-ethereum/common"
)

// Block represents of an Ethereum block
type Block struct {
  Num        int64          `gorm:"column:eth_block_num"`
	Timestamp  uint64         
	Hash       ethCommon.Hash 
	ParentHash ethCommon.Hash `gorm:"-" json:"-"`
}

// RollupData contains information returned by the Rollup smart contract
type RollupData struct {
	// L1UserTxs that were submitted in the block
	L1UserTxs            []L1Tx
	Batches              []BatchData
	AddedTokens          []Token
	Withdrawals          []WithdrawInfo
	UpdateBucketWithdraw []BucketUpdate
	Vars                 *RollupVariables
}

// NewRollupData creates an empty RollupData with the slices initialized.
func NewRollupData() RollupData {
	return RollupData{
		L1UserTxs:   make([]L1Tx, 0),
		Batches:     make([]BatchData, 0),
		AddedTokens: make([]Token, 0),
		Withdrawals: make([]WithdrawInfo, 0),
		Vars:        nil,
	}
}

// WDelayerTransfer represents a transfer (either deposit or withdrawal) in the
// WDelayer smart contract
// type WDelayerTransfer struct {
// 	Owner  ethCommon.Address
// 	Token  ethCommon.Address
// 	Amount *big.Int
// 	// TxHash ethCommon.Hash // hash of the transaction in which the wdelayer transfer happened
// }

// WDelayerData contains information returned by the WDelayer smart contract
// type WDelayerData struct {
// 	Vars     *WDelayerVariables
// 	Deposits []WDelayerTransfer
// 	// We use an array because there can be multiple deposits in a single eth transaction
// 	DepositsByTxHash       map[ethCommon.Hash][]*WDelayerTransfer
// 	Withdrawals            []WDelayerTransfer
// 	EscapeHatchWithdrawals []WDelayerEscapeHatchWithdrawal
// }
//
// // NewWDelayerData creates an empty WDelayerData.
// func NewWDelayerData() WDelayerData {
// 	return WDelayerData{
// 		Vars:             nil,
// 		Deposits:         make([]WDelayerTransfer, 0),
// 		DepositsByTxHash: make(map[ethCommon.Hash][]*WDelayerTransfer),
// 		Withdrawals:      make([]WDelayerTransfer, 0),
// 	}
// }

// BlockData contains the information of a Block
type BlockData struct {
	Block  Block
	Rollup RollupData
	// Auction  AuctionData
	// WDelayer WDelayerData
}
