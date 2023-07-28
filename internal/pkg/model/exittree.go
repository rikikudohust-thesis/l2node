package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	ethCommon "github.com/ethereum/go-ethereum/common"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"github.com/iden3/go-merkletree"
)

// ExitInfo represents the ExitTree Leaf data
type ExitInfo struct {
	BatchNum    BatchNum                        `json:"batch_num"`
	AccountIdx  Idx                             `json:"account_idx"`
	MerkleProof *merkletree.CircomVerifierProof `gorm:"type:jsonb" json:"merkle_proof"`
	Balance     *big.Int                        `gorm:"type:numeric" json:"balance"`
	// InstantWithdrawn is the ethBlockNum in which the exit is withdrawn
	// instantly.  nil means this hasn't happened.
	InstantWithdrawn *int64 `json:"instant_withdrawn"`
	// DelayedWithdrawRequest is the ethBlockNum in which the exit is
	// requested to be withdrawn from the delayedWithdrawn smart contract.
	// nil means this hasn't happened.
	DelayedWithdrawRequest *int64 `json:"delayed_withdraw_request"`
	// DelayedWithdrawn is the ethBlockNum in which the exit is withdrawn
	// from the delayedWithdrawn smart contract.  nil means this hasn't
	// happened.
	DelayedWithdrawn *int64 `json:"delayed_withdrawn"`
}

type ExitTree struct {
	BatchNum    BatchNum                        `json:"batch_num"`
	AccountIdx  Idx                             `json:"account_idx"`
	MerkleProof *merkletree.CircomVerifierProof `gorm:"type:jsonb" json:"merkle_proof"`
	Balance     *big.Int                        `gorm:"type:numeric" json:"balance"`
	// InstantWithdrawn is the ethBlockNum in which the exit is withdrawn
	// instantly.  nil means this hasn't happened.
	InstantWithdrawn *int64 `json:"instant_withdrawn"`
	// DelayedWithdrawRequest is the ethBlockNum in which the exit is
	// requested to be withdrawn from the delayedWithdrawn smart contract.
	// nil means this hasn't happened.
	DelayedWithdrawRequest *int64 `json:"delayed_withdraw_request"`
	// DelayedWithdrawn is the ethBlockNum in which the exit is withdrawn
	// from the delayedWithdrawn smart contract.  nil means this hasn't
	// happened.
	DelayedWithdrawn *int64 `json:"delayed_withdrawn"`
}

// WithdrawInfo represents a withdraw action to the rollup
type WithdrawInfo struct {
	Idx             Idx
	NumExitRoot     BatchNum
	InstantWithdraw bool
	TxHash          ethCommon.Hash // hash of the transaction in which the withdraw happened
	Owner           ethCommon.Address
	Token           ethCommon.Address
}

// Root     *merkletree.Hash   `json:"root"`
// Siblings []*merkletree.Hash `json:"siblings"`
// OldKey   *merkletree.Hash   `json:"oldKey"`
// OldValue *merkletree.Hash   `json:"oldValue"`
// IsOld0   bool               `json:"isOld0"`
// Key      *merkletree.Hash   `json:"key"`
// Values   *merkletree.Hash   `json:"value"`
// Fnc      int                `json:"fnc"` // 0: inclusion, 1: non inclusion

func (eig *ExitInfoGorm) ToExitInfo() *ExitInfo {
	balanceBI, _ := new(big.Int).SetString(eig.Balance, 10)
	return &ExitInfo{
		BatchNum:   eig.BatchNum,
		AccountIdx: eig.AccountIdx,
		MerkleProof: &merkletree.CircomVerifierProof{
			Root:     eig.MerkleProof.Root,
			Siblings: eig.MerkleProof.Siblings,
			OldKey:   eig.MerkleProof.OldKey,
			IsOld0:   eig.MerkleProof.IsOld0,
			Key:      eig.MerkleProof.Key,
			Value:    eig.MerkleProof.Values,
			Fnc:      eig.MerkleProof.Fnc,
		},
		Balance:                balanceBI,
		InstantWithdrawn:       eig.InstantWithdrawn,
		DelayedWithdrawRequest: eig.DelayedWithdrawRequest,
		DelayedWithdrawn:       eig.DelayedWithdrawn,
	}
}

type ExitInfoGorm struct {
	BatchNum    BatchNum             `json:"batch_num"`
	AccountIdx  Idx                  `json:"account_idx"`
	MerkleProof *CircomVerifierProof `gorm:"type:jsonb" json:"merkle_proof"`
	Balance     string               `gorm:"type:numeric" json:"balance"`
	// InstantWithdrawn is the ethBlockNum in which the exit is withdrawn
	// instantly.  nil means this hasn't happened.
	InstantWithdrawn *int64 `json:"instant_withdrawn"`
	// DelayedWithdrawRequest is the ethBlockNum in which the exit is
	// requested to be withdrawn from the delayedWithdrawn smart contract.
	// nil means this hasn't happened.
	DelayedWithdrawRequest *int64 `json:"delayed_withdraw_request"`
	// DelayedWithdrawn is the ethBlockNum in which the exit is withdrawn
	// from the delayedWithdrawn smart contract.  nil means this hasn't
	// happened.
	DelayedWithdrawn *int64 `json:"delayed_withdrawn"`
}

type ExitInfoGormV2 struct {
	BatchNum    BatchNum             `json:"batch_num"`
	AccountIdx  Idx                  `json:"account_idx"`
	MerkleProof *CircomVerifierProof `gorm:"type:jsonb" json:"merkle_proof"`
	Balance     string               `gorm:"type:numeric" json:"balance"`
	// InstantWithdrawn is the ethBlockNum in which the exit is withdrawn
	// instantly.  nil means this hasn't happened.
	InstantWithdrawn *int64 `json:"instant_withdrawn"`
	// DelayedWithdrawRequest is the ethBlockNum in which the exit is
	// requested to be withdrawn from the delayedWithdrawn smart contract.
	// nil means this hasn't happened.
	DelayedWithdrawRequest *int64 `json:"delayed_withdraw_request"`
	// DelayedWithdrawn is the ethBlockNum in which the exit is withdrawn
	// from the delayedWithdrawn smart contract.  nil means this hasn't
	// happened.
	DelayedWithdrawn *int64                `json:"delayed_withdrawn"`
	EthAddr          common.Address        `json:"ethAddr"`
	BJJ              babyjub.PublicKeyComp `json:"bjj"`
	TokenID          TokenID               `json:"tokenId"`
}

type CircomVerifierProof struct {
	Root     *merkletree.Hash   `json:"root"`
	Siblings []*merkletree.Hash `json:"siblings"`
	OldKey   *merkletree.Hash   `json:"oldKey"`
	OldValue *merkletree.Hash   `json:"oldValue"`
	IsOld0   bool               `json:"isOld0"`
	Key      *merkletree.Hash   `json:"key"`
	Values   *merkletree.Hash   `json:"value"`
	Fnc      int                `json:"fnc"` // 0: inclusion, 1: non inclusion
}

func (proof *CircomVerifierProof) Scan(value interface{}) error {
	// Check if the value is nil
	if value == nil {
		*proof = CircomVerifierProof{} // Set the proof to an empty struct
		return nil
	}

	// Check if the value is of type []byte
	bytesValue, ok := value.([]byte)
	if !ok {
		return errors.New("Invalid Scan value for CircomVerifierProof")
	}

	// Deserialize the byte slice into the CircomVerifierProof struct
	if err := json.Unmarshal(bytesValue, proof); err != nil {
		return err
	}

	return nil
}

func (proof CircomVerifierProof) Value() (driver.Value, error) {
	// Serialize the CircomVerifierProof struct into a byte slice
	bytesValue, err := json.Marshal(proof)
	if err != nil {
		return nil, err
	}

	return bytesValue, nil
}
