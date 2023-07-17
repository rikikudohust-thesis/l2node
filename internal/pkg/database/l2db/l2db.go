package l2db

import (
	"fmt"
	"math/big"
	"time"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model/nonce"

	"github.com/ethereum/go-ethereum/common"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const queryTxRaw = `
select txes.id, l2.*, a_to.eth_addr as receiver, a_from.eth_addr as sender, txes.batch_num as l1_batch from 
txes full outer join 
(select txes_l2.id as tx_hash, txes_l2.type as tx_type ,txes_l2.is_l1 as "type" , tx_pool.state as "state" ,txes_l2.from_eth_addr, txes_l2.from_idx, txes_l2.to_idx, txes_l2.token_id, txes_l2.deposit_amount as deposit_amount, txes_l2.amount as amount, txes_l2.batch_num as batch  from txes_l2 full outer join tx_pool on txes_l2.id = tx_pool.tx_id) as l2
on txes.id = l2.tx_hash
left join accounts_l2 as a_to on a_to.idx = l2.to_idx
left join accounts_l2 as a_from on a_from.idx = l2.from_idx
where l2.from_eth_addr = ? or a_from.eth_addr = ? or a_to.eth_addr = ?
`

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
	Info      *string
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

func StorePoolL2Txs(db *gorm.DB, txs []model.PoolL2Tx) error {
	fmt.Println("HERE")
	l2txs := make([]PoolL2TxStore, 0, len(txs))
	for i := range txs {
		// Format extra DB fields and nullables
		var (
			toEthAddr *common.Address
			toBJJ     *babyjub.PublicKeyComp
			// Info (always nil)
			// Rq fields, nil unless tx.RqFromIdx != 0
			info          *string
			rqFromIdx     *model.Idx
			rqToIdx       *model.Idx
			rqToEthAddr   *common.Address
			rqToBJJ       *babyjub.PublicKeyComp
			rqTokenID     *model.TokenID
			rqAmount      *big.Int
			rqFee         *model.FeeSelector
			rqNonce       *nonce.Nonce
			rqOffset      *uint8
			atomicGroupID *model.AtomicGroupID
			maxNumBatch   *uint32
		)
		// AmountFloat
		f := new(big.Float).SetInt((*big.Int)(txs[i].Amount))
		amountF, _ := f.Float64()
		// ToEthAddr
		if txs[i].ToEthAddr != model.EmptyAddr {
			toEthAddr = &txs[i].ToEthAddr
		}
		// ToBJJ
		if txs[i].ToBJJ != model.EmptyBJJComp {
			toBJJ = &txs[i].ToBJJ
		}
		// MAxNumBatch
		if txs[i].MaxNumBatch != 0 {
			maxNumBatch = &txs[i].MaxNumBatch
		}
		// Rq fields
		if txs[i].RqFromIdx != 0 {
			// RqFromIdx
			rqFromIdx = &txs[i].RqFromIdx
			// RqToIdx
			if txs[i].RqToIdx != 0 {
				rqToIdx = &txs[i].RqToIdx
			}
			// RqToEthAddr
			if txs[i].RqToEthAddr != model.EmptyAddr {
				rqToEthAddr = &txs[i].RqToEthAddr
			}
			// RqToBJJ
			if txs[i].RqToBJJ != model.EmptyBJJComp {
				rqToBJJ = &txs[i].RqToBJJ
			}
			// RqTokenID
			rqTokenID = &txs[i].RqTokenID
			// RqAmount
			if txs[i].RqAmount != nil {
				rqAmount = txs[i].RqAmount
			}
			// RqFee
			rqFee = &txs[i].RqFee
			// RqNonce
			rqNonce = &txs[i].RqNonce
			// RqOffset
			rqOffset = &txs[i].RqOffset
			// AtomicGroupID
			atomicGroupID = &txs[i].AtomicGroupID
		}
		l2txs = append(l2txs, PoolL2TxStore{
			TxID:          txs[i].TxID,
			FromIdx:       txs[i].FromIdx,
			ToIdx:         txs[i].ToIdx,
			ToEthAddr:     toEthAddr,
			ToBJJ:         toBJJ,
			TokenID:       txs[i].TokenID,
			Amount:        txs[i].Amount,
			Fee:           txs[i].Fee,
			Nonce:         txs[i].Nonce,
			State:         txs[i].State,
			Info:          info,
			Signature:     txs[i].Signature,
			RqFromIdx:     rqFromIdx,
			RqToIdx:       rqToIdx,
			RqToEthAddr:   rqToEthAddr,
			RqToBJJ:       rqToBJJ,
			RqTokenID:     rqTokenID,
			RqAmount:      rqAmount,
			RqFee:         rqFee,
			RqNonce:       rqNonce,
			Type:          txs[i].Type,
			AmountF:       &amountF,
			ClientIP:      txs[i].ClientIP,
			RqOffset:      rqOffset,
			AtomicGroupID: atomicGroupID,
			MaxNumBatch:   maxNumBatch,
		})
	}

	if len(l2txs) > 0 {
		query := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tx_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"state", "batch_num"}),
		})
		if err := query.Table("tx_pool").Create(&l2txs).Error; err != nil {
			fmt.Printf("failed to save tx to tx_pool, err: %v", err)
			return err
		}
	}

	return nil
}

func AddExitTree(db *gorm.DB, exittree []model.ExitInfo, batchNum model.BatchNum) error {
	eigs := make([]*model.ExitInfoGorm, 0, len(exittree))
	for i := range exittree {
		eigs = append(eigs, &model.ExitInfoGorm{
			BatchNum:   batchNum,
			AccountIdx: exittree[i].AccountIdx,
			MerkleProof: &model.CircomVerifierProof{
				Root:     exittree[i].MerkleProof.Root,
				Siblings: exittree[i].MerkleProof.Siblings,
				OldKey:   exittree[i].MerkleProof.OldKey,
				IsOld0:   exittree[i].MerkleProof.IsOld0,
				Key:      exittree[i].MerkleProof.Key,
				Values:   exittree[i].MerkleProof.Value,
				Fnc:      exittree[i].MerkleProof.Fnc,
			},
			Balance:                exittree[i].Balance.String(),
			InstantWithdrawn:       exittree[i].InstantWithdrawn,
			DelayedWithdrawRequest: exittree[i].DelayedWithdrawRequest,
			DelayedWithdrawn:       exittree[i].DelayedWithdrawn,
		})
	}
	if len(exittree) > 0 {
		if err := db.Table("exit_trees").Create(&eigs).Error; err != nil {
			return err
		}
	}
	return nil
}

func GetExitTree(db *gorm.DB, idx model.Idx) ([]*model.ExitInfoGorm, error) {
	var eig []*model.ExitInfoGorm
	query := db.Table("exit_trees")
	query = query.Where("account_idx = ?", idx)
	if err := query.Find(&eig).Error; err != nil {
		return nil, err
	}
	return eig, nil
}

func GetTx(db *gorm.DB, ethAddr *common.Address) ([]*model.TransactionResponse, error) {
	var txs []*model.TransactionResponse
  if err := db.Raw(queryTxRaw, ethAddr, ethAddr, ethAddr).Find(&txs).Error; err != nil {
		return nil, err
	}
	return txs, nil
}
