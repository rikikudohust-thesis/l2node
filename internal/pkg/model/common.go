package model

import (
	"context"
	"database/sql/driver"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

type IJob interface {
	Run(ctx context.Context)
}

type IService interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, src interface{}) error
}

func IsNilErr(err error) bool {
	return errors.Is(err, redis.Nil)
}

// type BigInt big.Int
//
// func NewBigInt(d int64) *BigInt {
// 	return (*BigInt)(big.NewInt(d))
// }
//
// func NewBigIntFromString(s string) *BigInt {
// 	b, ok := new(big.Int).SetString(s, 0)
// 	if !ok {
// 		b = big.NewInt(0)
// 	}
// 	return (*BigInt)(b)
// }
//
// func NewBigIntFromFloat(f float64) *BigInt {
// 	b, _ := big.NewFloat(f).Int(nil)
// 	return (*BigInt)(b)
// }
//
// func (b *BigInt) Bytes() []byte {
//   return (*big.Int)(b).Bytes()
// }
//
// func (b *BigInt) BI() *big.Int {
// 	return (*big.Int)(b)
// }
//
// func (b *BigInt) Cmp(y *BigInt) int {
// 	return (*big.Int)(b).Cmp((*big.Int)(y))
// }
//
// func (b *BigInt) Add(y *BigInt) *BigInt {
// 	if y == nil || y.Cmp(NewBigInt(0)) == 0 {
// 		return b
// 	}
// 	if b == nil {
// 		r := *y
// 		return &r
// 	}
// 	return (*BigInt)((*big.Int)(b).Add((*big.Int)(b), (*big.Int)(y)))
// }
//
// func (b *BigInt) Sub(y *BigInt) *BigInt {
// 	if y == nil || y.Cmp(NewBigInt(0)) == 0 {
// 		return b
// 	}
// 	if b == nil {
// 		r := *y
// 		return &r
// 	}
// 	return (*BigInt)((*big.Int)(b).Sub((*big.Int)(b), (*big.Int)(y)))
// }
//
// func (b *BigInt) Mul(y *BigInt) *BigInt {
// 	if y == nil || y.Cmp(NewBigInt(1)) == 0 {
// 		return b
// 	}
// 	if b == nil {
// 		r := *y
// 		return &r
// 	}
// 	return (*BigInt)((*big.Int)(b).Mul((*big.Int)(b), (*big.Int)(y)))
// }
//
// func (b *BigInt) Div(y *BigInt) *BigInt {
// 	if y == nil || y.Cmp(NewBigInt(1)) == 0 {
// 		return b
// 	}
// 	if b == nil {
// 		r := *y
// 		return &r
// 	}
// 	return (*BigInt)((*big.Int)(b).Div((*big.Int)(b), (*big.Int)(y)))
// }
//
// func (b *BigInt) F64() float64 {
// 	res, _ := new(big.Float).SetInt((*big.Int)(b)).Float64()
// 	return res
// }
//
// func (b *BigInt) String() string {
// 	return (*big.Int)(b).String()
// }
//
// func (b *BigInt) Hex() string {
// 	h := (*big.Int)(b).Text(HexBase)
// 	if len(h)%2 == 0 {
// 		return "0x" + h
// 	}
// 	return "0x0" + h
// }
//
// func (b *BigInt) MarshalJSON() ([]byte, error) {
// 	return []byte(fmt.Sprintf("%q", b.String())), nil
// }
//
// func (b *BigInt) Scan(value interface{}) error {
// 	stringValue, ok := value.(string)
// 	if !ok {
// 		return fmt.Errorf("failed to get string value: %v", value)
// 	}
// 	(*big.Int)(b).SetString(stringValue, DecBase)
// 	return nil
// }
//
// func (b BigInt) Value() (driver.Value, error) {
// 	return (*big.Int)(&b).String(), nil
// }
//
// func (BigInt) GormDataType() string {
// 	return GormDataTypeText
// }

type Idx uint64

// String returns a string representation of the Idx
func (idx Idx) String() string {
	return strconv.Itoa(int(idx))
}

// Bytes returns a byte array representing the Idx
func (idx Idx) Bytes() ([6]byte, error) {
	if idx > maxIdxValue {
		return [6]byte{}, (ErrIdxOverflow)
	}
	var idxBytes [8]byte
	binary.BigEndian.PutUint64(idxBytes[:], uint64(idx))
	var b [6]byte
	copy(b[:], idxBytes[2:])
	return b, nil
}

// BigInt returns a *big.Int representing the Idx
func (idx Idx) BigInt() *big.Int {
	return big.NewInt(int64(idx))
}

// IdxFromBytes returns Idx from a byte array
func IdxFromBytes(b []byte) (Idx, error) {
	if len(b) != IdxBytesLen {
		return 0, (fmt.Errorf("can not parse Idx, bytes len %d, expected %d",
			len(b), IdxBytesLen))
	}
	var idxBytes [8]byte
	copy(idxBytes[2:], b[:])
	idx := binary.BigEndian.Uint64(idxBytes[:])
	return Idx(idx), nil
}

// IdxFromBigInt converts a *big.Int to Idx type
func IdxFromBigInt(b *big.Int) (Idx, error) {
	if b.Int64() > maxIdxValue {
		return 0, (ErrNumOverflow)
	}
	return Idx(uint64(b.Int64())), nil
}

type TokenID uint32 // current implementation supports up to 2^32 tokens

// Bytes returns a byte array of length 4 representing the TokenID
func (t TokenID) Bytes() []byte {
	var tokenIDBytes [4]byte
	binary.BigEndian.PutUint32(tokenIDBytes[:], uint32(t))
	return tokenIDBytes[:]
}

// BigInt returns the *big.Int representation of the TokenID
func (t TokenID) BigInt() *big.Int {
	return big.NewInt(int64(t))
}

// TokenIDFromBytes returns TokenID from a byte array
func TokenIDFromBytes(b []byte) (TokenID, error) {
	if len(b) != tokenIDBytesLen {
		return 0, fmt.Errorf("can not parse TokenID, bytes len %d, expected 4",
			len(b))
	}
	tid := binary.BigEndian.Uint32(b[:4])
	return TokenID(tid), nil
}

// TokenIDFromBigInt returns a TokenID with the value of the given *big.Int
func TokenIDFromBigInt(b *big.Int) TokenID {
	return TokenID(b.Int64())
}

type BatchNum uint64

func NewBatchNum(batchNum int64) *BatchNum {
  batchNumCV := BatchNum(uint64(batchNum))
  return &batchNumCV
}

type TxID [TxIDLen]byte

// Scan implements Scanner for database/sql.
func (txid *TxID) Scan(src interface{}) error {
	srcB, ok := src.([]byte)
	if !ok {
		return (fmt.Errorf("can't scan %T into TxID", src))
	}
	if len(srcB) != TxIDLen {
		return (fmt.Errorf("can't scan []byte of len %d into TxID, need %d",
			len(srcB), TxIDLen))
	}
	copy(txid[:], srcB)
	return nil
}

// Value implements valuer for database/sql.
func (txid TxID) Value() (driver.Value, error) {
	return txid[:], nil
}

// String returns a string hexadecimal representation of the TxID
func (txid TxID) String() string {
	return "0x" + hex.EncodeToString(txid[:])
}

// NewTxIDFromString returns a string hexadecimal representation of the TxID
func NewTxIDFromString(idStr string) (TxID, error) {
	txid := TxID{}
	idStr = strings.TrimPrefix(idStr, "0x")
	decoded, err := hex.DecodeString(idStr)
	if err != nil {
		return TxID{}, (err)
	}
	if len(decoded) != TxIDLen {
		return txid, errors.New("Invalid idStr")
	}
	copy(txid[:], decoded)
	return txid, nil
}

// MarshalText marshals a TxID
func (txid TxID) MarshalText() ([]byte, error) {
	return []byte(txid.String()), nil
}

// UnmarshalText unmarshalls a TxID
func (txid *TxID) UnmarshalText(data []byte) error {
	idStr := string(data)
	id, err := NewTxIDFromString(idStr)
	if err != nil {
		return err
	}
	*txid = id
	return nil
}

// TxType is a string that represents the type of a Hermez network transaction
type TxType string

