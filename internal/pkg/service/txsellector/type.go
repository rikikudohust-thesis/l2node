package txsellector

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"

	"github.com/hermeznetwork/tracerr"
)

type Proof struct {
	PiA      [3]*big.Int    `json:"pi_a"`
	PiB      [3][2]*big.Int `json:"pi_b"`
	PiC      [3]*big.Int    `json:"pi_c"`
	Protocol string         `json:"protocol"`
}

type bigInt big.Int

func (b *bigInt) UnmarshalText(text []byte) error {
	_, ok := (*big.Int)(b).SetString(string(text), 10)
	if !ok {
		return tracerr.Wrap(fmt.Errorf("invalid big int: \"%v\"", string(text)))
	}
	return nil
}

// UnmarshalJSON unmarshals the proof from a JSON encoded proof with the big
// ints as strings
func (p *Proof) UnmarshalJSON(data []byte) error {
	proof := struct {
		PiA      [3]*bigInt    `json:"pi_a"`
		PiB      [3][2]*bigInt `json:"pi_b"`
		PiC      [3]*bigInt    `json:"pi_c"`
		Protocol string        `json:"protocol"`
	}{}
	if err := json.Unmarshal(data, &proof); err != nil {
		return tracerr.Wrap(err)
	}
	p.PiA[0] = (*big.Int)(proof.PiA[0])
	p.PiA[1] = (*big.Int)(proof.PiA[1])
	p.PiA[2] = (*big.Int)(proof.PiA[2])
	if p.PiA[2].Int64() != 1 {
		return tracerr.Wrap(fmt.Errorf("Expected PiA[2] == 1, but got %v", p.PiA[2]))
	}
	p.PiB[0][0] = (*big.Int)(proof.PiB[0][0])
	p.PiB[0][1] = (*big.Int)(proof.PiB[0][1])
	p.PiB[1][0] = (*big.Int)(proof.PiB[1][0])
	p.PiB[1][1] = (*big.Int)(proof.PiB[1][1])
	p.PiB[2][0] = (*big.Int)(proof.PiB[2][0])
	p.PiB[2][1] = (*big.Int)(proof.PiB[2][1])
	if p.PiB[2][0].Int64() != 1 || p.PiB[2][1].Int64() != 0 {
		return tracerr.Wrap(fmt.Errorf("Expected PiB[2] == [1, 0], but got %v", p.PiB[2]))
	}
	p.PiC[0] = (*big.Int)(proof.PiC[0])
	p.PiC[1] = (*big.Int)(proof.PiC[1])
	p.PiC[2] = (*big.Int)(proof.PiC[2])
	if p.PiC[2].Int64() != 1 {
		return tracerr.Wrap(fmt.Errorf("Expected PiC[2] == 1, but got %v", p.PiC[2]))
	}
	p.Protocol = proof.Protocol
	return nil
}

// PublicInputs are the public inputs of the proof
type PublicInputs []*big.Int

// UnmarshalJSON unmarshals the JSON into the public inputs where the bigInts
// are in decimal as quoted strings
func (p *PublicInputs) UnmarshalJSON(data []byte) error {
	pubInputs := []*bigInt{}
	if err := json.Unmarshal(data, &pubInputs); err != nil {
		return tracerr.Wrap(err)
	}
	*p = make([]*big.Int, len(pubInputs))
	for i, v := range pubInputs {
		([]*big.Int)(*p)[i] = (*big.Int)(v)
	}
	return nil
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

type failedAtomicGroup struct {
	id         model.AtomicGroupID
	failedTxID model.TxID // ID of the tx that made the entire atomic group fail
	reason     model.TxSelectorError
}
