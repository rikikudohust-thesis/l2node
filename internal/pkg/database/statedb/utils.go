package statedb

import (
	"bytes"
	"fmt"

	ethmodel "github.com/ethereum/go-ethereum/common"
  "github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/hermeznetwork/hermez-node/log"
	"github.com/hermeznetwork/tracerr"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"github.com/iden3/go-merkletree/db"
)

func concatEthAddrTokenID(addr ethmodel.Address, tokenID model.TokenID) []byte {
	var b []byte
	b = append(b, addr.Bytes()...)
	b = append(b[:], tokenID.Bytes()[:]...)
	return b
}
func concatEthAddrBJJTokenID(addr ethmodel.Address, pk babyjub.PublicKeyComp,
	tokenID model.TokenID) []byte {
	pkComp := pk
	var b []byte
	b = append(b, addr.Bytes()...)
	b = append(b[:], pkComp[:]...)
	b = append(b[:], tokenID.Bytes()[:]...)
	return b
}

// setIdxByEthAddrBJJ stores the given Idx in the StateDB as follows:
// - key: Eth Address, value: idx
// - key: EthAddr & BabyJubJub PublicKey Compressed, value: idx
// If Idx already exist for the given EthAddr & BJJ, the remaining Idx will be
// always the smallest one.
func (s *StateDB) setIdxByEthAddrBJJ(idx model.Idx, addr ethmodel.Address,
	pk babyjub.PublicKeyComp, tokenID model.TokenID) error {
	oldIdx, err := s.GetIdxByEthAddrBJJ(addr, pk, tokenID)
	if err == nil {
		// EthAddr & BJJ already have an Idx
		// check which Idx is smaller
		// if new idx is smaller, store the new one
		// if new idx is bigger, don't store and return, as the used one will be the old
		if idx >= oldIdx {
			log.Debug("StateDB.setIdxByEthAddrBJJ: Idx not stored because there " +
				"already exist a smaller Idx for the given EthAddr & BJJ")
			return nil
		}
	}

	// store idx for EthAddr & BJJ assuming that EthAddr & BJJ still don't
	// have an Idx stored in the DB, and if so, the already stored Idx is
	// bigger than the given one, so should be updated to the new one
	// (smaller)
	tx, err := s.db.DB().NewTx()
	if err != nil {
		return tracerr.Wrap(err)
	}
	idxBytes, err := idx.Bytes()
	if err != nil {
		return tracerr.Wrap(err)
	}
	// store Addr&BJJ-idx
	k := concatEthAddrBJJTokenID(addr, pk, tokenID)
	err = tx.Put(append(PrefixKeyAddrBJJ, k...), idxBytes[:])
	if err != nil {
		return tracerr.Wrap(err)
	}

	// store Addr-idx
	k = concatEthAddrTokenID(addr, tokenID)
	err = tx.Put(append(PrefixKeyAddr, k...), idxBytes[:])
	if err != nil {
		return tracerr.Wrap(err)
	}
	err = tx.Commit()
	if err != nil {
		return tracerr.Wrap(err)
	}
	return nil
}

// GetIdxByEthAddr returns the smallest Idx in the StateDB for the given
// Ethereum Address. Will return model.Idx(0) and error in case that Idx is
// not found in the StateDB.
func (s *StateDB) GetIdxByEthAddr(addr ethmodel.Address, tokenID model.TokenID) (model.Idx,
	error) {
	k := concatEthAddrTokenID(addr, tokenID)
	b, err := s.db.DB().Get(append(PrefixKeyAddr, k...))
	if err != nil {
		return model.Idx(0), tracerr.Wrap(fmt.Errorf("GetIdxByEthAddr: %s: ToEthAddr: %s, TokenID: %d",
			ErrIdxNotFound, addr.Hex(), tokenID))
	}
	idx, err := model.IdxFromBytes(b)
	if err != nil {
		return model.Idx(0), tracerr.Wrap(fmt.Errorf("GetIdxByEthAddr: %s: ToEthAddr: %s, TokenID: %d",
			err, addr.Hex(), tokenID))
	}
	return idx, nil
}

// GetIdxByEthAddrBJJ returns the smallest Idx in the StateDB for the given
// Ethereum Address AND the given BabyJubJub PublicKey. If `addr` is the zero
// address, it's ignored in the query.  If `pk` is nil, it's ignored in the
// query.  Will return model.Idx(0) and error in case that Idx is not found in
// the StateDB.
func (s *StateDB) GetIdxByEthAddrBJJ(addr ethmodel.Address, pk babyjub.PublicKeyComp,
	tokenID model.TokenID) (model.Idx, error) {
	if !bytes.Equal(addr.Bytes(), model.EmptyAddr.Bytes()) && pk == model.EmptyBJJComp {
		// ToEthAddr
		// case ToEthAddr!=0 && ToBJJ=0
		return s.GetIdxByEthAddr(addr, tokenID)
	} else if !bytes.Equal(addr.Bytes(), model.EmptyAddr.Bytes()) &&
		pk != model.EmptyBJJComp {
		// case ToEthAddr!=0 && ToBJJ!=0
		k := concatEthAddrBJJTokenID(addr, pk, tokenID)
		b, err := s.db.DB().Get(append(PrefixKeyAddrBJJ, k...))
		if tracerr.Unwrap(err) == db.ErrNotFound {
			// return the error (ErrNotFound), so can be traced at upper layers
			return model.Idx(0), tracerr.Wrap(ErrIdxNotFound)
		} else if err != nil {
			return model.Idx(0),
				tracerr.Wrap(fmt.Errorf("GetIdxByEthAddrBJJ: %s: ToEthAddr: %s, ToBJJ: %s, TokenID: %d",
					ErrIdxNotFound, addr.Hex(), pk, tokenID))
		}
		idx, err := model.IdxFromBytes(b)
		if err != nil {
			return model.Idx(0),
				tracerr.Wrap(fmt.Errorf("GetIdxByEthAddrBJJ: %s: ToEthAddr: %s, ToBJJ: %s, TokenID: %d",
					err, addr.Hex(), pk, tokenID))
		}
		return idx, nil
	}
	// rest of cases (included case ToEthAddr==0) are not possible
	return model.Idx(0),
		tracerr.Wrap(
			fmt.Errorf("GetIdxByEthAddrBJJ: Not found, %s: ToEthAddr: %s, ToBJJ: %s, TokenID: %d",
				ErrGetIdxNoCase, addr.Hex(), pk, tokenID))
}

// GetTokenIDsFromIdxs returns a map containing the model.TokenID with its
// respective model.Idx for a given slice of model.Idx
func (s *StateDB) GetTokenIDsFromIdxs(idxs []model.Idx) (map[model.TokenID]model.Idx, error) {
	m := make(map[model.TokenID]model.Idx)
	for i := 0; i < len(idxs); i++ {
		a, err := s.GetAccount(idxs[i])
		if err != nil {
			return nil,
				tracerr.Wrap(fmt.Errorf("GetTokenIDsFromIdxs error on GetAccount with Idx==%d: %s",
					idxs[i], err.Error()))
		}
		m[a.TokenID] = idxs[i]
	}
	return m, nil
}
