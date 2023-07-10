package model

import (
	"encoding/binary"
	"fmt"
	"math/big"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

const batchNumBytesLen = 8

type Batch struct {
	BatchNum  BatchNum
  EthTxHash ethCommon.Hash `gorm:"-"`
	// Ethereum block in which the batch is forged
	EthBlockNum        uint64
	ForgerAddr         ethCommon.Address
  CollectedFees      map[TokenID]*big.Int `gorm:"-"`
	FeeIdxsCoordinator []Idx                `gorm:"-"`
	StateRoot          *big.Int             `gorm:"type:numeric"`
	NumAccounts        int
	LastIdx            int64
	ExitRoot           *big.Int `gorm:"type:numeric"`
  GasUsed            uint64 `gorm:"-"`
	GasPrice           *big.Int `gorm:"type:numeric"`
	EtherPriceUSD      float64
	// ForgeL1TxsNum is optional, Only when the batch forges L1 txs. Identifier that corresponds
	// to the group of L1 txs forged in the current batch.
	ForgeL1TxsNum *int64
	SlotNum       int64    
	TotalFeesUSD *float64
}

// NewEmptyBatch creates a new empty batch
func NewEmptyBatch() *Batch {
	return &Batch{
		BatchNum:           0,
		EthBlockNum:        0,
		ForgerAddr:         ethCommon.Address{},
		CollectedFees:      make(map[TokenID]*big.Int),
		FeeIdxsCoordinator: make([]Idx, 0),
		StateRoot:          big.NewInt(0),
		NumAccounts:        0,
		LastIdx:            0,
		ExitRoot:           big.NewInt(0),
		ForgeL1TxsNum:      nil,
		// SlotNum:            0,
		TotalFeesUSD: nil,
	}
}

// Bytes returns a byte array of length 4 representing the BatchNum
func (bn BatchNum) Bytes() []byte {
	var batchNumBytes [batchNumBytesLen]byte
	binary.BigEndian.PutUint64(batchNumBytes[:], uint64(bn))
	return batchNumBytes[:]
}

// BigInt returns a *big.Int representing the BatchNum
func (bn BatchNum) BigInt() *big.Int {
	return big.NewInt(int64(bn))
}

// BatchNumFromBytes returns BatchNum from a []byte
func BatchNumFromBytes(b []byte) (BatchNum, error) {
	if len(b) != batchNumBytesLen {
		return 0,
			(fmt.Errorf("can not parse BatchNumFromBytes, bytes len %d, expected %d",
				len(b), batchNumBytesLen))
	}
	batchNum := binary.BigEndian.Uint64(b[:batchNumBytesLen])
	return BatchNum(batchNum), nil
}

type BatchData struct {
	L1Batch bool
	// L1UserTxs that were forged in the batch
	L1UserTxs        []L1Tx
	L1CoordinatorTxs []L1Tx
	L2Txs            []L2Tx
	CreatedAccounts  []Account
	UpdatedAccounts  []AccountUpdate
	ExitTree         []ExitInfo
	Batch            Batch
}

// NewBatchData creates an empty BatchData with the slices initialized.
func NewBatchData() *BatchData {
	return &BatchData{
		L1Batch: false,
		// L1UserTxs:        make([]common.L1Tx, 0),
		L1CoordinatorTxs: make([]L1Tx, 0),
		L2Txs:            make([]L2Tx, 0),
		CreatedAccounts:  make([]Account, 0),
		ExitTree:         make([]ExitInfo, 0),
		Batch:            Batch{},
	}
}
