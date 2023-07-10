package model

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/model/nonce"

	"github.com/ethereum/go-ethereum/common"
	"github.com/hermeznetwork/tracerr"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"github.com/iden3/go-iden3-crypto/poseidon"
	cryptoUtils "github.com/iden3/go-iden3-crypto/utils"
)

type Account struct {
	Idx      Idx                   `gorm:"column:idx" json:"idx"`
	TokenID  TokenID               `gorm:"column:token_id" json:"token_id"`
	BatchNum BatchNum              `json:"batch_num"`
	BJJ      babyjub.PublicKeyComp `json:"bjj"`
	EthAddr  common.Address        `json:"eth_addr"`
  Nonce    nonce.Nonce           `gorm:"-" json:"nonce"`
	Balance  *big.Int              `gorm:"-"`
}

func (a *Account) String() string {
	buf := bytes.NewBufferString("")
	fmt.Fprintf(buf, "Idx: %v, ", a.Idx)
	fmt.Fprintf(buf, "BJJ: %s..., ", a.BJJ.String()[:10])
	fmt.Fprintf(buf, "EthAddr: %s..., ", a.EthAddr.String()[:10])
	fmt.Fprintf(buf, "TokenID: %v, ", a.TokenID)
	fmt.Fprintf(buf, "Nonce: %d, ", a.Nonce)
	fmt.Fprintf(buf, "Balance: %s, ", a.Balance.String())
	fmt.Fprintf(buf, "BatchNum: %v, ", a.BatchNum)
	return buf.String()
}

func (a *Account) Bytes() ([32 * NLeafElems]byte, error) {
	var b [32 * NLeafElems]byte

	if a.Nonce > nonce.MaxNonceValue {
		return b, tracerr.Wrap(fmt.Errorf("%s Nonce", ErrNumOverflow))
	}
	if len(a.Balance.Bytes()) > maxBalanceBytes {
		return b, tracerr.Wrap(fmt.Errorf("%s Balance", ErrNumOverflow))
	}

	nonceBytes, err := a.Nonce.Bytes()
	if err != nil {
		return b, tracerr.Wrap(err)
	}

	copy(b[28:32], a.TokenID.Bytes())
	copy(b[23:28], nonceBytes[:])

	pkSign, pkY := babyjub.UnpackSignY(a.BJJ)
	if pkSign {
		b[22] = 1
	}
	balanceBytes := a.Balance.Bytes()
	copy(b[64-len(balanceBytes):64], balanceBytes)
	// Check if there is possibility of finite field overflow
	ayBytes := pkY.Bytes()
	if len(ayBytes) == 32 { //nolint:gomnd
		ayBytes[0] = ayBytes[0] & 0x3f //nolint:gomnd
		pkY = big.NewInt(0).SetBytes(ayBytes)
	}
	finiteFieldMod, ok := big.NewInt(0).SetString("21888242871839275222246405745257275088548364400416034343698204186575808495617", 10) //nolint:gomnd
	if !ok {
		return b, errors.New("error setting bjj finite field")
	}
	pkY = pkY.Mod(pkY, finiteFieldMod)
	ayBytes = pkY.Bytes()
	copy(b[96-len(ayBytes):96], ayBytes)
	copy(b[108:128], a.EthAddr.Bytes())
	return b, nil
}

// BigInts returns the [5]*big.Int, where each *big.Int is inside the Finite Field
func (a *Account) BigInts() ([NLeafElems]*big.Int, error) {
	e := [NLeafElems]*big.Int{}

	b, err := a.Bytes()
	if err != nil {
		return e, tracerr.Wrap(err)
	}

	e[0] = new(big.Int).SetBytes(b[0:32])
	e[1] = new(big.Int).SetBytes(b[32:64])
	e[2] = new(big.Int).SetBytes(b[64:96])
	e[3] = new(big.Int).SetBytes(b[96:128])

	return e, nil
}

func (a *Account) HashValue() (*big.Int, error) {
	bi, err := a.BigInts()
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	return poseidon.Hash(bi[:])
}

func AccountFromBytes(b [32 * NLeafElems]byte) (*Account, error) {
	tokenID, err := TokenIDFromBytes(b[28:32])
	if err != nil {
		return nil, tracerr.Wrap(err)
	}
	var nonceBytes5 [5]byte
	copy(nonceBytes5[:], b[23:28])
	nonce := nonce.FromBytes(nonceBytes5)
	sign := b[22] == 1

	balance := new(big.Int).SetBytes(b[40:64])
	// Balance is max of 192 bits (24 bytes)
	if !bytes.Equal(b[32:40], []byte{0, 0, 0, 0, 0, 0, 0, 0}) {
		return nil, tracerr.Wrap(fmt.Errorf("%s Balance", ErrNumOverflow))
	}
	ay := new(big.Int).SetBytes(b[64:96])
	publicKeyComp := babyjub.PackSignY(sign, ay)
	ethAddr := common.BytesToAddress(b[108:128])

	if !cryptoUtils.CheckBigIntInField(balance) {
		return nil, tracerr.Wrap(ErrNotInFF)
	}
	if !cryptoUtils.CheckBigIntInField(ay) {
		return nil, tracerr.Wrap(ErrNotInFF)
	}

	a := Account{
		TokenID: TokenID(tokenID),
		Nonce:   nonce,
		Balance: balance,
		BJJ:     publicKeyComp,
		EthAddr: ethAddr,
	}
	return &a, nil
}

type AccountUpdate struct {
	EthBlockNum int64       
	BatchNum    BatchNum    
	Idx         Idx         
	Nonce       nonce.Nonce 
	Balance     *big.Int    `gorm:"type:numeric"`
}
