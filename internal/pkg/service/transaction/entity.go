package transaction

import (
	"math/big"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model/nonce"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iden3/go-iden3-crypto/babyjub"
)

type PoolL2Tx struct {
	// Stored in DB: mandatory fields

	// TxID (12 bytes) for L2Tx is:
	// bytes:  |  1   |    6    |   5   |
	// values: | type | FromIdx | Nonce |
	TxID    model.TxID `json:""`
	FromIdx model.Idx  `json:"fromIdx"`
	ToIdx   model.Idx  `json:"toIdx"`
	// AuxToIdx is only used internally at the StateDB to avoid repeated
	// computation when processing transactions (from Synchronizer,
	// TxSelector, BatchBuilder)
	ToEthAddr   common.Address      `json:"toEthAddr"`
	ToBJJ       string              `json:"toBjj"`
	TokenID     model.TokenID       `json:"tokenID"`
	Amount      string              `json:"amount"`
	Fee         model.FeeSelector   `json:""`
	Nonce       nonce.Nonce         `json:"nonce"`
	State       model.PoolL2TxState `json:""`
	MaxNumBatch uint32              `json:""`
	Signature   string              `json:"signature"`
	Timestamp   uint64              `json:""`
	Type        string              `json:"type"`
	// Info contains information about the status & State of the
	// transaction. As for example, if the Tx has not been selected in the
	// last batch due not enough Balance at the Sender account, this reason
	// would appear at this parameter.
}
type PoolL2TxStore struct {
	// Stored in DB: mandatory fields

	// TxID (12 bytes) for L2Tx is:
	// bytes:  |  1   |    6    |   5   |
	// values: | type | FromIdx | Nonce |
	TxID    model.TxID
	FromIdx model.Idx
	ToIdx   model.Idx
	// AuxToIdx is only used internally at the StateDB to avoid repeated
	// computation when processing transactions (from Synchronizer,
	// TxSelector, BatchBuilder)
	AuxToIdx    model.Idx `gorm:"-"`
	ToEthAddr   *common.Address
	ToBJJ       *babyjub.PublicKeyComp
	TokenID     model.TokenID
	Amount      *big.Int `gorm:"type:numeric"`
	Fee         model.FeeSelector
	Nonce       nonce.Nonce
	State       model.PoolL2TxState `gorm:"column:state"`
	MaxNumBatch *uint32
	// Info contains information about the status & State of the
	// transaction. As for example, if the Tx has not been selected in the
	// last batch due not enough Balance at the Sender account, this reason
	// would appear at this parameter.
	Info      string
	ErrorCode int
	ErrorType string
	Signature babyjub.SignatureComp
	Timestamp uint64
	// Stored in DB: optional fields, may be uninitialized
	AmountF           *float64 
	AtomicGroupID     *model.AtomicGroupID
	RqFromIdx         *model.Idx
	RqToIdx           *model.Idx
	RqToEthAddr       *common.Address
	RqToBJJ           *babyjub.PublicKeyComp
	RqTokenID         *model.TokenID
	RqAmount          *big.Int `gorm:"type:numeric"`
	RqFee             *model.FeeSelector
	RqNonce           *nonce.Nonce
	AbsoluteFee       float64      `gorm:"-"`
	AbsoluteFeeUpdate time.Time    `gorm:"-"`
	Type              model.TxType `gorm:"column:tx_type"`
	RqOffset          *uint8
	// Extra DB write fields (not included in JSON)
	ClientIP string
	// Extra metadata, may be uninitialized
	RqTxCompressedData []byte `gorm:"-"`
	TokenSymbol        string `gorm:"-"`
}
