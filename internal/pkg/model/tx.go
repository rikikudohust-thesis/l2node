package model

import (
	"bytes"
	"fmt"
	"math/big"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/model/nonce"

	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/iden3/go-iden3-crypto/babyjub"
)

var (
	// EmptyTxID is used to check if a TxID is 0
	EmptyTxID = TxID([TxIDLen]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
)

// Tx is a struct used by the TxSelector & BatchBuilder as a generic type generated from L1Tx &
// PoolL2Tx
type Tx struct {
	// Generic
	IsL1        bool     `gorm:"column:is_l1"`
	TxID        TxID     `gorm:"column:id"`
	Type        TxType   `gorm:"column:type"`
	Position    int      `gorm:"column:position"`
	FromIdx     Idx      `gorm:"column:from_idx"`
	ToIdx       Idx      `gorm:"column:to_idx"`
	Amount      *big.Int `gorm:"type:numeric"`
	AmountFloat float64  `gorm:"column:amount_f"`
	TokenID     TokenID  `gorm:"column:token_id"`
	USD         *float64 `gorm:"column:amount_usd"`
	// BatchNum in which this tx was forged. If the tx is L2, this must be != 0
	BatchNum *BatchNum `gorm:"column:batch_num"`
	// Ethereum Block Number in which this L1Tx was added to the queue
	EthBlockNum uint64 `gorm:"column:eth_block_num"`
	// L1
	// ToForgeL1TxsNum in which the tx was forged / will be forged
	ToForgeL1TxsNum *int64 `gorm:"column:to_forge_l1_txs_num"`
	// UserOrigin is set to true if the tx was originated by a user, false if it was aoriginated
	// by a coordinator. Note that this differ from the spec for implementation simplification
	// purpposes
	UserOrigin         *bool                 `gorm:"column:user_origin"`
	FromEthAddr        ethCommon.Address     `gorm:"column:from_eth_addr"`
	FromBJJ            babyjub.PublicKeyComp `gorm:"column:from_bjj"`
	DepositAmount      *big.Int              `gorm:"type:numeric"`
	DepositAmountFloat *float64              `gorm:"column:deposit_amount_f"`
	DepositAmountUSD   *float64              `gorm:"column:deposit_amount_usd"`
	// L2
	Fee    *FeeSelector `gorm:"column:fee"`
	FeeUSD *float64     `gorm:"column:fee_usd"`
	Nonce  *nonce.Nonce `gorm:"column:nonce"`
}

type TxGorm struct {
	// Generic
	IsL1        bool     `gorm:"column:is_l1"`
	TxID        TxID     `gorm:"column:id"`
	Type        TxType   `gorm:"column:type"`
	Position    int      `gorm:"column:position"`
	FromIdx     Idx      `gorm:"column:from_idx"`
	ToIdx       Idx      `gorm:"column:to_idx"`
	Amount      string   `gorm:"type:numeric"`
	AmountFloat float64  `gorm:"column:amount_f"`
	TokenID     TokenID  `gorm:"column:token_id"`
	USD         *float64 `gorm:"column:amount_usd"`
	// BatchNum in which this tx was forged. If the tx is L2, this must be != 0
	BatchNum *BatchNum `gorm:"column:batch_num"`
	// Ethereum Block Number in which this L1Tx was added to the queue
	EthBlockNum uint64 `gorm:"column:eth_block_num"`
	// L1
	// ToForgeL1TxsNum in which the tx was forged / will be forged
	ToForgeL1TxsNum *int64 `gorm:"column:to_forge_l1_txs_num"`
	// UserOrigin is set to true if the tx was originated by a user, false if it was aoriginated
	// by a coordinator. Note that this differ from the spec for implementation simplification
	// purpposes
	UserOrigin         *bool                 `gorm:"column:user_origin"`
	FromEthAddr        ethCommon.Address     `gorm:"column:from_eth_addr"`
	FromBJJ            babyjub.PublicKeyComp `gorm:"column:from_bjj"`
	DepositAmount      string                `gorm:"type:numeric"`
	DepositAmountFloat *float64              `gorm:"column:deposit_amount_f"`
	DepositAmountUSD   *float64              `gorm:"column:deposit_amount_usd"`
	// L2
	Fee    *FeeSelector `gorm:"column:fee"`
	FeeUSD *float64     `gorm:"column:fee_usd"`
	Nonce  *nonce.Nonce `gorm:"column:nonce"`
}

func (txg *TxGorm) ToTx() *Tx {
	amount := new(big.Int)
	amountBI, _ := amount.SetString(txg.Amount, 10)
	dAmount := new(big.Int)
	dAmountBI, _ := dAmount.SetString(txg.Amount, 10)

	return &Tx{
		IsL1:        txg.IsL1,
		TxID:        txg.TxID,
		Type:        txg.Type,
		Position:    txg.Position,
		FromIdx:     txg.FromIdx,
		ToIdx:       txg.ToIdx,
		Amount:      amountBI,
		AmountFloat: txg.AmountFloat,
		TokenID:     txg.TokenID,
		USD:         txg.USD,
		// BatchNum in which this tx was forged. If the tx is L2, this must be != 0
		BatchNum: txg.BatchNum,
		// Ethereum Block Number in which this L1Tx was added to the queue
		EthBlockNum: txg.EthBlockNum,
		// L1
		// ToForgeL1TxsNum in which the tx was forged / will be forged
		ToForgeL1TxsNum: txg.ToForgeL1TxsNum,
		// UserOrigin is set to true if the tx was originated by a user, false if it was aoriginated
		// by a coordinator. Note that this differ from the spec for implementation simplification
		// purpposes
		UserOrigin:         txg.UserOrigin,
		FromEthAddr:        txg.FromEthAddr,
		FromBJJ:            txg.FromBJJ,
		DepositAmount:      dAmountBI,
		DepositAmountFloat: txg.DepositAmountFloat,
		DepositAmountUSD:   txg.DepositAmountUSD,
		// L2
		Fee:    txg.Fee,
		FeeUSD: txg.FeeUSD,
		Nonce:  txg.Nonce,
	}
}

func (tx *Tx) String() string {
	buf := bytes.NewBufferString("")
	fmt.Fprintf(buf, "Type: %s, ", tx.Type)
	fmt.Fprintf(buf, "FromIdx: %s, ", tx.FromIdx)
	if tx.Type == TxTypeTransfer ||
		tx.Type == TxTypeDepositTransfer ||
		tx.Type == TxTypeCreateAccountDepositTransfer {
		fmt.Fprintf(buf, "ToIdx: %s, ", tx.ToIdx)
	}
	if tx.Type == TxTypeDeposit ||
		tx.Type == TxTypeDepositTransfer ||
		tx.Type == TxTypeCreateAccountDepositTransfer {
		fmt.Fprintf(buf, "DepositAmount: %d, ", tx.DepositAmount)
	}
	if tx.Type != TxTypeDeposit {
		fmt.Fprintf(buf, "Amount: %s, ", tx.Amount)
	}
	if tx.Type == TxTypeTransfer ||
		tx.Type == TxTypeDepositTransfer ||
		tx.Type == TxTypeCreateAccountDepositTransfer {
		fmt.Fprintf(buf, "Fee: %d, ", tx.Fee)
	}
	fmt.Fprintf(buf, "TokenID: %d", tx.TokenID)

	return buf.String()
}

// L1Tx returns a *L1Tx from the Tx
func (tx *Tx) L1Tx() (*L1Tx, error) {
	return &L1Tx{
		TxID:            tx.TxID,
		ToForgeL1TxsNum: tx.ToForgeL1TxsNum,
		Position:        tx.Position,
		UserOrigin:      *tx.UserOrigin,
		FromIdx:         tx.FromIdx,
		FromEthAddr:     tx.FromEthAddr,
		FromBJJ:         tx.FromBJJ,
		ToIdx:           tx.ToIdx,
		TokenID:         tx.TokenID,
		Amount:          tx.Amount,
		DepositAmount:   tx.DepositAmount,
		EthBlockNum:     tx.EthBlockNum,
		Type:            tx.Type,
		BatchNum:        tx.BatchNum,
	}, nil
}
