package model

import (
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/iden3/go-iden3-crypto/babyjub"
)

const (
	ServiceL1Tx         = "l1tx"
	ServiceSelector     = "selector"
	ServiceSynchronier  = "synchronier"
	ServiceBatchBuilder = "batchbuilder"
)

const (
	GormDataTypeJSON = "json"
	GormDataTypeText = "text"
)

const (
	DecBase = 10
	HexBase = 16
)

const (
	SecPerHour = 3600
	SecPerDay  = SecPerHour * 24
	SecPerWeek = SecPerDay * 7
)

const (
	ChainIDEthereum = 1
)

const (
	// TxIDPrefixL1UserTx is the prefix that determines that the TxID is for
	// a L1UserTx
	//nolinter:gomnd
	TxIDPrefixL1UserTx = byte(0)

	// TxIDPrefixL1CoordTx is the prefix that determines that the TxID is
	// for a L1CoordinatorTx
	//nolinter:gomnd
	TxIDPrefixL1CoordTx = byte(1)

	// TxIDPrefixL2Tx is the prefix that determines that the TxID is for a
	// L2Tx (or PoolL2Tx)
	//nolinter:gomnd
	TxIDPrefixL2Tx = byte(2)

	// TxIDLen is the length of the TxID byte array
	TxIDLen = 33
)

var ChainNames = map[uint64]string{
	ChainIDEthereum: "ethereum",
}

const (
	// TxTypeExit represents L2->L1 token transfer.  A leaf for this account appears in the exit
	// tree of the block
	TxTypeExit TxType = "Exit"
	// TxTypeTransfer represents L2->L2 token transfer
	TxTypeTransfer TxType = "Transfer"
	// TxTypeDeposit represents L1->L2 transfer
	TxTypeDeposit TxType = "Deposit"
	// TxTypeCreateAccountDeposit represents creation of a new leaf in the state tree
	// (newAcconut) + L1->L2 transfer
	TxTypeCreateAccountDeposit TxType = "CreateAccountDeposit"
	// TxTypeCreateAccountDepositTransfer represents L1->L2 transfer + L2->L2 transfer
	TxTypeCreateAccountDepositTransfer TxType = "CreateAccountDepositTransfer"
	// TxTypeDepositTransfer TBD
	TxTypeDepositTransfer TxType = "DepositTransfer"
	// TxTypeForceTransfer TBD
	TxTypeForceTransfer TxType = "ForceTransfer"
	// TxTypeForceExit TBD
	TxTypeForceExit TxType = "ForceExit"
	// TxTypeTransferToEthAddr TBD
	TxTypeTransferToEthAddr TxType = "TransferToEthAddr"
	// TxTypeTransferToBJJ TBD
	TxTypeTransferToBJJ TxType = "TransferToBJJ"
)

const (
	// NLeafElems is the number of elements for a leaf
	NLeafElems = 4

	// maxBalanceBytes is the maximum bytes that can use the
	// Account.Balance *big.Int
	maxBalanceBytes = 24

	// IdxBytesLen idx bytes
	IdxBytesLen = 6
	// maxIdxValue is the maximum value that Idx can have (48 bits:
	// maxIdxValue=2**48-1)
	maxIdxValue = 0xffffffffffff

	// UserThreshold determines the threshold from the User Idxs can be
	UserThreshold = 32
	// IdxUserThreshold is a Idx type value that determines the threshold
	// from the User Idxs can be
	IdxUserThreshold = Idx(UserThreshold)
)

var (
	// FFAddr is used to check if an ethereum address is 0xff..ff
	FFAddr = ethCommon.HexToAddress("0xffffffffffffffffffffffffffffffffffffffff")
	// EmptyAddr is used to check if an ethereum address is 0
	EmptyAddr = ethCommon.HexToAddress("0x0000000000000000000000000000000000000000")

	EmptyBJJComp = babyjub.PublicKeyComp([32]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})

	SignatureConstantBytes = []byte{198, 11, 230, 15}
)
