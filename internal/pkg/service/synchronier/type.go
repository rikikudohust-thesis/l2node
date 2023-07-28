package synchronier

import (
	"math/big"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"

	"github.com/ethereum/go-ethereum/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
)

type RollupEventL1UserTx struct {
	L1UserTx model.L1Tx
}

type rollupEventL1UserTxAux struct {
	ToForgeL1TxsNum uint64
	Position        uint8
	L1UserTx        []byte
}

type RollupEventAddToken struct {
	TokenAddress common.Address
	TokenID      uint32
}

type RollupEventForgeBatch struct {
	BatchNum     int64
	EthTxHash    common.Hash
	L1UserTxsLen uint16
	GasPrice     *big.Int
	GasUsed      uint64
}

type RollupEventUpdateForgeL1L2BatchTimeout struct {
	NewForgeL1L2BatchTimeout int64
}

type RollupEventUpdateFeeAddToken struct {
	NewFeeAddToken *big.Int
}

type RollupEvents struct {
	L1UserTx                    []RollupEventL1UserTx
	AddToken                    []RollupEventAddToken
	ForgeBatch                  []RollupEventForgeBatch
	UpdateForgeL1L2BatchTimeout []RollupEventUpdateForgeL1L2BatchTimeout
	UpdateFeeAddToken           []RollupEventUpdateFeeAddToken
	Withdraw                    []RollupEventWithdraw
}

func NewRollUpEvents() *RollupEvents {
	return &RollupEvents{
		L1UserTx:                    make([]RollupEventL1UserTx, 0),
		AddToken:                    make([]RollupEventAddToken, 0),
		ForgeBatch:                  make([]RollupEventForgeBatch, 0),
		UpdateForgeL1L2BatchTimeout: make([]RollupEventUpdateForgeL1L2BatchTimeout, 0),
		UpdateFeeAddToken:           make([]RollupEventUpdateFeeAddToken, 0),
	}
}

type RollupForgeBatchArgs struct {
	NewLastIdx            int64
	NewStRoot             *big.Int
	NewExitRoot           *big.Int
	L1UserTxs             []model.L1Tx
	L1CoordinatorTxs      []model.L1Tx
	L1CoordinatorTxsAuths [][]byte // Authorization for accountCreations for each L1CoordinatorTx
	L2TxsData             []model.L2Tx
	FeeIdxCoordinator     []model.Idx
	// Circuit selector
	VerifierIdx uint8
	L1Batch     bool
	ProofA      [2]*big.Int
	ProofB      [2][2]*big.Int
	ProofC      [2]*big.Int
}

type rollupForgeBatchArgsAux struct {
	NewLastIdx             *big.Int
	NewStRoot              *big.Int
	NewExitRoot            *big.Int
	EncodedL1CoordinatorTx []byte
	L1L2TxsData            []byte
	FeeIdxCoordinator      []byte
	// Circuit selector
	VerifierIdx uint8
	L1Batch     bool
	ProofA      [2]*big.Int
	ProofB      [2][2]*big.Int
	ProofC      [2]*big.Int
}

type RollupEventWithdraw struct {
	Idx             uint64
	NumExitRoot     uint64
	InstantWithdraw bool
	TxHash          ethCommon.Hash // Hash of the transaction that generated this event
}
