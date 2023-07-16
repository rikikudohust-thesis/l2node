package transaction

import (
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model/nonce"
	"github.com/rikikudohust-thesis/l2node/pkg/context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"gorm.io/gorm"
)

const (
	chainID = 1
)

func SetupRouter(router *gin.RouterGroup, db *gorm.DB, r model.IService) {
	router.POST("/transactions", postTx(db))
	router.GET("/transactions", getTx(db))
}

func postTxVer2(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.New(c).WithLogPrefix("post-transactions-v2-api")
		var receivedTx PoolL2Tx
		if err := c.ShouldBindJSON(&receivedTx); err != nil {
			ctx.Infof("failed to bin params, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}
		byteSignRaw, err := hex.DecodeString(receivedTx.Signature)
		if err != nil {
			ctx.Infof("failed to decode signature, err:%v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		var byteSign [64]byte
		copy(byteSign[:], byteSignRaw)
		amount := new(big.Int)
		amountBI, validAmount := amount.SetString(receivedTx.Amount, 10)
		if !validAmount {
			ctx.Errorf("invalid amount, err:%v", err)
			ctx.AbortWith400("invalid amount")
			return
		}
		pooll2tx := model.PoolL2Tx{
			FromIdx:   receivedTx.FromIdx,
			ToEthAddr: receivedTx.ToEthAddr,
			Amount:    amountBI,
			TokenID:   receivedTx.TokenID,
			Nonce:     receivedTx.Nonce,
			Type:      model.TxType(receivedTx.Type),
			State:     model.PoolL2TxStatePending,
			Signature: byteSign,
		}

		if err := pooll2tx.SetID(); err != nil {
			ctx.Infof("failed to Set id, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		if err := pooll2tx.SetType(); err != nil {
			ctx.Infof("failed to set type, err: &v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		valid, err := validationL2Tx(ctx, db, pooll2tx)
		if err != nil {
			ctx.Infof("failed to validate tx, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		if !valid {
			ctx.Infoln("failed to validate tx")
			return
		}

		if err := storeL2Tx(ctx, db, pooll2tx); err != nil {
			ctx.Infof("failed to store tx, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}

	}
}

func postTx(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.New(c).WithLogPrefix("post-transactions-api")
		var receivedTx PoolL2Tx

		if err := c.ShouldBindJSON(&receivedTx); err != nil {
			ctx.Infof("failed to bin params, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}

    ctx.Infof("receive: %+v", receivedTx)
		byteSignRaw, err := hex.DecodeString(receivedTx.Signature)
		if err != nil {
			ctx.Infof("failed to decode signature, err:%v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		var byteSign [64]byte
		copy(byteSign[:], byteSignRaw)

		// amount := new(big.Int)
		// amount, _ = amount.SetString("10000000000000000000", 10)
		amount := new(big.Int)
		amountBI, validAmount := amount.SetString(receivedTx.Amount, 10)
		if !validAmount {
			ctx.Errorf("invalid amount, err:%v", err)
			ctx.AbortWith400("invalid amount")
			return
		}

		pooll2tx := model.PoolL2Tx{
			FromIdx:   receivedTx.FromIdx,
			ToIdx:     receivedTx.ToIdx,
			Amount:    amountBI,
			TokenID:   receivedTx.TokenID,
			Nonce:     receivedTx.Nonce,
			Type:      model.TxType(receivedTx.Type),
			State:     model.PoolL2TxStatePending,
			Signature: byteSign,
		}
		if err := pooll2tx.SetID(); err != nil {
			ctx.Infof("failed to Set id, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		if err := pooll2tx.SetType(); err != nil {
			ctx.Infof("failed to set type, err: &v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		valid, err := validationL2Tx(ctx, db, pooll2tx)
		if err != nil {
			ctx.Infof("failed to validate tx, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		if !valid {
			ctx.Infoln("failed to validate tx")
			return
		}

		if err := storeL2Tx(ctx, db, pooll2tx); err != nil {
			ctx.Infof("failed to store tx, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}
	}
}

func getTx(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.New(c).WithLogPrefix("get-transactions-api")
		ctx.Infof("get all l2 transactions")
	}
}

func validationL2Tx(ctx context.Context, db *gorm.DB, tx model.PoolL2Tx) (bool, error) {
	_, err := model.NewPoolL2Tx(&tx)
	if err != nil {
		ctx.Errorf("failed to create new pool l2 tx, err: %v", err)
		return false, err
	}

	account, err := getAccount(ctx, db, tx.FromIdx)
	if err != nil {
		ctx.Errorf("failed to get account, err: %v", err)
		return false, err
	}

	if tx.TokenID != account.TokenID {
		return false, fmt.Errorf("tx.TokenID (%v) != account.TokenID(%v)", tx.TokenID, account.TokenID)
	}

	if tx.Nonce < account.Nonce {
		return false, fmt.Errorf("tx.Nonce(%v) < account.Nonce(%v)", tx.Nonce, account.Nonce)
	}

	ctx.Infof("account data: %+v", account)
	fmt.Printf("tx data: %+v", tx)
	if !tx.VerifySignature(chainID, account.BJJ) {
		return false, fmt.Errorf("wrong signature")
	}

	switch tx.Type {
	case model.TxTypeTransfer:
		// var toAccount *model.Account
		// if tx.ToEthAddr != model.EmptyAddr {
		// 	toAccount, err = getAccountByEthAddrAndToken(ctx, db, &tx.ToEthAddr, tx.TokenID)
		// 	if err != nil {
		// 		ctx.Errorf("failed to get to account, err: %v", err)
		// 		return false, err
		// 	}
		// } else {
		toAccount, err := getAccount(ctx, db, tx.ToIdx)
		if err != nil {
			ctx.Errorf("failed to get to account, err: %v", err)
			return false, err
			// }
		}

		if tx.TokenID != toAccount.TokenID {
			return false, fmt.Errorf("tx.TokenID (%v) != account.TokenID(%v)", tx.TokenID, toAccount.TokenID)
		}
	case model.TxTypeExit:
	}

	return true, nil
}

func storeL2Tx(ctx context.Context, db *gorm.DB, tx model.PoolL2Tx) error {
	var (
		toEthAddr *common.Address
		toBJJ     *babyjub.PublicKeyComp
		// Info (always nil)
		// Rq fields, nil unless tx.RqFromIdx != 0
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
	f := new(big.Float).SetInt(tx.Amount)
	amountFloat, _ := f.Float64()

	if tx.ToEthAddr != model.EmptyAddr {
		toEthAddr = &tx.ToEthAddr
	}

	if tx.ToBJJ != model.EmptyBJJComp {
		toBJJ = &tx.ToBJJ
	}

	if tx.MaxNumBatch != 0 {
		maxNumBatch = &tx.MaxNumBatch
	}
	// Rq fields
	if tx.RqFromIdx != 0 {
		// RqFromIdx
		rqFromIdx = &tx.RqFromIdx
		// RqToIdx
		if tx.RqToIdx != 0 {
			rqToIdx = &tx.RqToIdx
		}
		// RqToEthAddr
		if tx.RqToEthAddr != model.EmptyAddr {
			rqToEthAddr = &tx.RqToEthAddr
		}
		// RqToBJJ
		if tx.RqToBJJ != model.EmptyBJJComp {
			rqToBJJ = &tx.RqToBJJ
		}
		// RqTokenID
		rqTokenID = &tx.RqTokenID
		// RqAmount
		if tx.RqAmount != nil {
			rqAmount = tx.RqAmount
		}
		// RqFee
		rqFee = &tx.RqFee
		// RqNonce
		rqNonce = &tx.RqNonce
		// RqOffset
		rqOffset = &tx.RqOffset
		// AtomicGroupID
		atomicGroupID = &tx.AtomicGroupID
	}

	info := "l2tx"
	txStore := &PoolL2TxStore{
		TxID:          tx.TxID,
		FromIdx:       tx.FromIdx,
		ToIdx:         tx.ToIdx,
		ToEthAddr:     toEthAddr,
		ToBJJ:         toBJJ,
		TokenID:       tx.TokenID,
		Amount:        tx.Amount,
		Fee:           tx.Fee,
		Nonce:         tx.Nonce,
		State:         tx.State,
		Info:          info,
		Signature:     tx.Signature,
		RqFromIdx:     rqFromIdx,
		RqToIdx:       rqToIdx,
		RqToEthAddr:   rqToEthAddr,
		RqToBJJ:       rqToBJJ,
		RqTokenID:     rqTokenID,
		RqAmount:      rqAmount,
		RqFee:         rqFee,
		RqNonce:       rqNonce,
		Type:          tx.Type,
		AmountF:       &amountFloat,
		ClientIP:      tx.ClientIP,
		RqOffset:      rqOffset,
		AtomicGroupID: atomicGroupID,
		MaxNumBatch:   maxNumBatch,
	}
	if err := db.Table("tx_pool").Create(&txStore).Error; err != nil {
		ctx.Errorf("failed to save tx to tx_pool, err: %v", err)
	}
	return nil
}

func getAccount(ctx context.Context, db *gorm.DB, idx model.Idx) (*model.Account, error) {
	type fullAccount struct {
		Idx      model.Idx
		TokenID  model.TokenID
		BatchNum model.BatchNum
		BJJ      babyjub.PublicKeyComp
		EthAddr  common.Address
		Nonce    nonce.Nonce
		Balance  string `gorm:"type:numeric"`
	}

	account := &fullAccount{}
	queryRaw := `SELECT distinct on (a.idx) a.idx, a.token_id, a.batch_num, a.bjj, 
			a.eth_addr, coalesce(au.nonce, '0') as nonce, coalesce(au.balance, '0') as balance
		FROM accounts_l2 a
	    LEFT JOIN account_updates_l2 au
			ON a.idx = au.idx
	   	WHERE a.idx = ?
	   	ORDER BY a.idx, au.eth_block_num desc;
  `

	if err := db.Raw(queryRaw, idx).Find(account).Error; err != nil {
		ctx.Errorf("faild to full account data, err: %v", err)
		return nil, err
	}

	balanceBI, _ := new(big.Int).SetString(account.Balance, 10)
	return &model.Account{
		Idx:      account.Idx,
		TokenID:  account.TokenID,
		BatchNum: account.BatchNum,
		BJJ:      account.BJJ,
		EthAddr:  account.EthAddr,
		Nonce:    account.Nonce,
		Balance:  balanceBI,
	}, nil
}

func getAccountByEthAddrAndToken(ctx context.Context, db *gorm.DB, ethAddr *common.Address, tokenID model.TokenID) (*model.Account, error) {
	type fullAccount struct {
		Idx      model.Idx
		TokenID  model.TokenID
		BatchNum model.BatchNum
		BJJ      babyjub.PublicKeyComp
		EthAddr  common.Address
		Nonce    nonce.Nonce
		Balance  string `gorm:"type:numeric"`
	}

	account := &fullAccount{}
	queryRaw := `SELECT distinct on (a.idx) a.idx, a.token_id, a.batch_num, a.bjj, 
			a.eth_addr, coalesce(au.nonce, '0') as nonce, coalesce(au.balance, '0') as balance
		FROM accounts_l2 a
	    LEFT JOIN account_updates_l2 au
			ON a.idx = au.idx
	   	WHERE a.eth_addr = ? and a.token_id = ?
	   	ORDER BY a.idx, au.eth_block_num desc;
  `

	if err := db.Raw(queryRaw, ethAddr, tokenID).Find(account).Error; err != nil {
		ctx.Errorf("faild to full account data, err: %v", err)
		return nil, err
	}

	balanceBI, _ := new(big.Int).SetString(account.Balance, 10)
	return &model.Account{
		Idx:      account.Idx,
		TokenID:  account.TokenID,
		BatchNum: account.BatchNum,
		BJJ:      account.BJJ,
		EthAddr:  account.EthAddr,
		Nonce:    account.Nonce,
		Balance:  balanceBI,
	}, nil

}
