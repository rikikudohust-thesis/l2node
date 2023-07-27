package txsellector

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/database/l2db"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/database/statedb"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/service/txprocessor"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/utils"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	syncingBlockCacheKey = "syncingBlock"
	l1l2timeout          = 5000
	maxTimeout           = 1000000
)

type job struct {
	globalCfg *model.JobConfig
	localCfg  *config
	db        *gorm.DB
	r         model.IService
	sdb       *statedb.StateDB
	syncDB    *statedb.StateDB
}

func NewJob(cfg *model.JobConfig, db *gorm.DB, r model.IService, sdb *statedb.StateDB, syncDB *statedb.StateDB) model.IJob {
	return &job{
		globalCfg: cfg,
		db:        db,
		r:         r,
		sdb:       sdb,
		syncDB:    syncDB,
	}
}

func (j *job) Run(ctx context.Context) {
	// j.sdb.Reset(2)
	// j.syncDB.Reset(2)
	// return
	c, existed := configs[j.globalCfg.ChainID]
	if !existed {
		return
	}
	j.localCfg = &c
	for {
		start := time.Now()
		fmt.Println("start")
		j.process(ctx)
		fmt.Println("done, elapsed: ", time.Since(start))
		time.Sleep(time.Duration(j.localCfg.JobIntervalSec) * time.Second)
	}
}

func (j *job) process(ctx context.Context) {
	currentBatch := j.sdb.CurrentBatch()
	if j.sdb.CurrentBatch() == model.BatchNum(0) {
		tp := txprocessor.NewTxProcessor(j.sdb, j.localCfg.Cfg)
		_, err := tp.ProcessTxs(nil, nil, nil, nil)

		if err != nil {
			j.sdb.Reset(0)
			return
		}
		tpSync := txprocessor.NewTxProcessor(j.syncDB, j.localCfg.Cfg)
		_, err = tpSync.ProcessTxs(nil, nil, nil, nil)
		if err != nil {
			fmt.Printf("failed to process processtxs")
			j.sdb.Reset(0)
			j.syncDB.Reset(0)
			return
		}
		if err := j.db.Create(&model.BlockL2{BatchNum: 1, EthBlockNum: uint64(0), IsL1: true}).Error; err != nil {
			fmt.Printf("failed to save genesis block l2, err: %v\n", err)
			j.sdb.Reset(0)
			j.syncDB.Reset(0)
			return
		}
		return
	}
	fmt.Println("Start checking l1l2 Bacth")
	l1Batch, requireL1Batch, err := j.checkL1L2Batch(ctx)
	if err != nil {
		fmt.Printf("failed to check l1l2 batch, err: %v\n", err)
		return
	}

	fmt.Println("Start get unforge l1")
	l1txs, err := j.getL1TxUnforged(ctx, l1Batch)
	if err != nil {
		fmt.Printf("failed to get unforge l1tx, err: %v\n", err)
		return
	}

	selectableL2TxLength := uint64(j.localCfg.Cfg.MaxTx) - uint64(len(l1txs))
	selectedL2Tx, err := j.getPendingTxs(ctx, selectableL2TxLength)
	if err != nil {
		fmt.Printf("failed to get l2 tx pool, err: %v\n", err)
		return
	}

	if l1Batch {
		isL1Pending, err := l2db.IsL1Pending(j.db)
		if err != nil {
			fmt.Printf("failed to check l1 pending, err: %v", err)
			return
		}
		if isL1Pending {
			requireL1Batch = isL1Pending
		}
		fmt.Println("l1batch: ", requireL1Batch)
		if len(l1txs)+len(selectedL2Tx) == 0 && !requireL1Batch {
			fmt.Printf("wait tx\n")
			return
		}
	} else {
		if len(selectedL2Tx) == 0 {
			fmt.Printf("wait tx\n")
			return
		}
	}

	if err := j.buildBatchL2(ctx, l1txs, selectedL2Tx, l1Batch); err != nil {
		fmt.Printf("failed to buil batch l2, err: %v\n", err)
		j.sdb.Reset(currentBatch)
		j.syncDB.Reset(currentBatch)
		return
	}
}

func (j *job) buildBatchL2(ctx context.Context, l1txs []model.L1Tx, pooll2txs []model.PoolL2Tx, l1Batch bool) error {
	ethBlockNum := uint64(0)
	if l1Batch {
		client, _ := utils.GetEvmClient(ctx, j.globalCfg.RPCs)
		ethBlockNum, _ = client.BlockNumber(ctx)
	}
	sdbBatch := j.sdb.CurrentBatch()
	for i := 0; i < len(pooll2txs); i++ {
		accSender, err := j.sdb.GetAccount(pooll2txs[i].FromIdx)
		if err != nil {
			fmt.Printf("faild to get accSender, err: %v\n", err)
			return err
		}
		pooll2txs[i].TokenID = accSender.TokenID
		// enoughBalance, balance, _ := tp.CheckEnoughBalance(pooll2txs[i])
		// if !enoughBalance {
		// 	return fmt.Errorf("not enough balance, balance: %v", balance)
		// }

		if pooll2txs[i].Nonce != accSender.Nonce {
			return fmt.Errorf("nonce not equal, tx.nonce: %d, account.nonce: %d", pooll2txs[i].Nonce, accSender.Nonce)
		}
	}
	tp := txprocessor.NewTxProcessor(j.sdb, j.localCfg.Cfg)
	tpSync := txprocessor.NewTxProcessor(j.syncDB, j.localCfg.Cfg)
	_, err := tp.ProcessTxs(nil, l1txs, nil, pooll2txs)
	if err != nil {
		fmt.Printf("failed to process processtxs")
		j.sdb.Reset(sdbBatch)
		return err
	}
	pout, err := tpSync.ProcessTxs(nil, l1txs, nil, pooll2txs)
	if err != nil {
		fmt.Printf("failed to process processtxs")
		j.sdb.Reset(sdbBatch)
		return err
	}

	fmt.Printf("exit infor :%+v\n", pout.ExitInfos)
	for i := range pout.ExitInfos {
		fmt.Printf("exit infor :%+v\n", pout.ExitInfos[i].MerkleProof.Siblings)
	}

	nextBatch := sdbBatch + 1
	txs := make([]model.Tx, 0, len(l1txs)+len(pooll2txs))
	for _, l1tx := range l1txs {
		af := new(big.Float).SetInt(l1tx.Amount)
		amountFloat, _ := af.Float64()
		laf := new(big.Float).SetInt(l1tx.DepositAmount)
		depositAmountFloat, _ := laf.Float64()
		txs = append(txs, model.Tx{
			IsL1:               true,
			TxID:               l1tx.TxID,
			Type:               l1tx.Type,
			Position:           l1tx.Position,
			FromIdx:            l1tx.FromIdx,
			ToIdx:              l1tx.ToIdx,
			Amount:             l1tx.Amount,
			AmountFloat:        amountFloat,
			TokenID:            l1tx.TokenID,
			BatchNum:           &nextBatch,
			EthBlockNum:        l1tx.EthBlockNum,
			ToForgeL1TxsNum:    l1tx.ToForgeL1TxsNum,
			UserOrigin:         &l1tx.UserOrigin,
			FromEthAddr:        l1tx.FromEthAddr,
			FromBJJ:            l1tx.FromBJJ,
			DepositAmount:      l1tx.DepositAmount,
			DepositAmountFloat: &depositAmountFloat,
		})
	}

	for i, pooll2tx := range pooll2txs {
		l2tx := pooll2tx.L2Tx()
		f := new(big.Float).SetInt(l2tx.Amount)
		amountFloat, _ := f.Float64()
		txs = append(txs, model.Tx{
			IsL1:     false,
			TxID:     l2tx.TxID,
			Type:     l2tx.Type,
			Position: l2tx.Position,
			FromIdx:  l2tx.FromIdx,
			// EffectiveFromIdx: &l2tx.FromIdx,
			ToIdx:       l2tx.ToIdx,
			TokenID:     l2tx.TokenID,
			Amount:      l2tx.Amount,
			AmountFloat: amountFloat,
			BatchNum:    &nextBatch,
			EthBlockNum: l2tx.EthBlockNum,
			// L2
			Fee:   &l2tx.Fee,
			Nonce: &l2tx.Nonce,
		})

		pooll2txs[i].State = model.PoolL2TxStateForging
	}
	createdAccounts := pout.CreatedAccounts
	tx := j.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			j.sdb.Reset(sdbBatch)
		}
	}()

	if err := tx.Error; err != nil {
		fmt.Printf("failed to create transaction, err: %v\n", err)
		return err
	}

	if len(createdAccounts) > 0 {
		if err := tx.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Table("accounts_l2").CreateInBatches(createdAccounts, len(createdAccounts)).Error; err != nil {
			fmt.Printf("failed to save new account l2db %v\n", err)
			tx.Rollback()
			return err
		}
	}

	updatedAccounts := make([]model.AccountUpdate, 0, len(pout.UpdatedAccounts))
	for _, acc := range pout.UpdatedAccounts {
		updatedAccounts = append(updatedAccounts,
			model.AccountUpdate{
				EthBlockNum: int64(ethBlockNum),
				BatchNum:    model.BatchNum(nextBatch),
				Idx:         acc.Idx,
				Nonce:       acc.Nonce,
				Balance:     acc.Balance,
			})
	}
	if len(updatedAccounts) > 0 {
		if err := tx.Clauses(clause.OnConflict{
			DoNothing: true,
		}).Table("account_updates_l2").Create(updatedAccounts).Error; err != nil {
			fmt.Printf("failed to save account update, err: %v\n", err)
			tx.Rollback()
			return err
		}
	}
	queryUpdateData := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"amount_success", "deposit_amount_success", "effective_from_idx", "batch_num"}),
	})

	if len(txs) > 0 {
		if err := queryUpdateData.Table("txes_l2").Create(txs).Error; err != nil {
			fmt.Printf("failed to save tx %v\n", err)
			tx.Rollback()
			return err
		}
	}
	queryBlockL2 := tx.Clauses(clause.OnConflict{
		DoNothing: true,
	})

	if err := queryBlockL2.Create(&model.BlockL2{BatchNum: nextBatch, EthBlockNum: uint64(ethBlockNum), IsL1: l1Batch}).Error; err != nil {
		fmt.Printf("failed to save block l2, err: %v\n", err)
		queryBlockL2.Rollback()
		j.sdb.Reset(sdbBatch)
		j.syncDB.Reset(sdbBatch)

		return err
	}

	if err := l2db.AddExitTree(tx, pout.ExitInfos, nextBatch); err != nil {
		fmt.Printf("failed to save exit tree, err: %v\n", err)
		tx.Rollback()
		j.sdb.Reset(sdbBatch)
		j.syncDB.Reset(sdbBatch)
		return err
	}

	if len(pooll2txs) > 0 {

		if err := l2db.StorePoolL2Txs(j.db, pooll2txs); err != nil {
			fmt.Printf("failed to save pool_l2_txes %v\n", err)
			tx.Rollback()
			j.sdb.Reset(sdbBatch)
			j.syncDB.Reset(sdbBatch)
			return err
		}

		// if err := queryUpdateL1tx.Table("tx_pool").Create(pooll2txs).Error; err != nil {
		// 	fmt.Printf("failed to save pool_l2_txes %v\n", err)
		// 	tx.Rollback()
		// 	j.sdb.Reset(sdbBatch)
		// 	j.syncDB.Reset(sdbBatch)
		//
		// 	return err
		// }
	}
	if err := tx.Commit().Error; err != nil {
		j.sdb.Reset(sdbBatch)
		j.syncDB.Reset(sdbBatch)
		return err
	}

	return nil
}

func (j *job) getL1TxUnforged(ctx context.Context, l1Batch bool) ([]model.L1Tx, error) {
	if !l1Batch {
		return []model.L1Tx{}, nil
	}
	lastL1Forge := uint64(0)
	query1 := j.db.Select("COUNT(is_l1)")
	query1 = query1.Table("block_l2")
	query1 = query1.Where("is_l1 = true")
	if err := query1.Find(&lastL1Forge).Error; err != nil {
		fmt.Printf("failed to get last l1 forge, %v\n", err)
	}

	var txs []*model.L1TxGorm

	query := j.db.Select("id, is_l1, to_forge_l1_txs_num, position, user_origin, from_idx, from_eth_addr, from_bjj, to_idx, token_id, amount, NULL AS effective_amount, deposit_amount, NULL AS effective_deposit_amount, eth_block_num, type, batch_num")
	query = query.Table("txes_l2")
	query = query.Where("batch_num IS NULL and to_forge_l1_txs_num = ?", lastL1Forge)
	query = query.Order("position")
	if err := query.Find(&txs).Error; err != nil {
		return []model.L1Tx{}, err
	}
	l1txs := make([]model.L1Tx, 0, len(txs))
	for _, l1g := range txs {
		l1txs = append(l1txs, *l1g.ToL1Tx())
	}
	return l1txs, nil
}

func (j *job) getPendingTxs(ctx context.Context, availableLen uint64) ([]model.PoolL2Tx, error) {
	var txsGorm []*model.PoolL2TxGorm
	query := j.db.Select(`tx_pool.tx_id, from_idx, to_idx, tx_pool.to_eth_addr, 
tx_pool.to_bjj, tx_pool.token_id, tx_pool.amount, tx_pool.fee, tx_pool.nonce, 
tx_pool.state, tx_pool.info, tx_pool.signature, tx_pool.timestamp, rq_from_idx, 
rq_to_idx, tx_pool.rq_to_eth_addr, tx_pool.rq_to_bjj, tx_pool.rq_token_id, tx_pool.rq_amount, 
tx_pool.rq_fee, tx_pool.rq_nonce, tx_pool.tx_type, tx_pool.rq_offset, tx_pool.atomic_group_id, tx_pool.max_num_batch`)
	query = query.Table("tx_pool INNER JOIN tokens ON tx_pool.token_id = tokens.token_id")
	query = query.Where("state = ?", model.PoolL2TxStatePending)
	query = query.Order("tx_pool.item_id")
	query = query.Limit(int(availableLen))

	if err := query.Find(&txsGorm).Error; err != nil {
		fmt.Printf("failed to get l2 pending, %v\n", err)
		return []model.PoolL2Tx{}, err
	}

	txs := make([]model.PoolL2Tx, 0, len(txsGorm))
	for _, txg := range txsGorm {
		txs = append(txs, *txg.ToPoolL2Tx())
	}
	return txs, nil
}

func (j *job) checkL1L2Batch(ctx context.Context) (bool, bool, error) {
	var ethBlockNumPtr sql.NullInt64
	query := j.db.Select("max(eth_block_num)")
	query = query.Table("block_l2")
	query = query.Where("is_l1 = true")
	if err := query.Scan(&ethBlockNumPtr).Error; err != nil {
		fmt.Println("failed to scan ethBlockNumPtr, err: %v", err)
		return false, false, err
	}

	ethBlockNumL1L2 := uint64(0)
	if ethBlockNumPtr.Valid {
		ethBlockNumL1L2 = uint64(ethBlockNumPtr.Int64)
	}

	client, err := utils.GetEvmClient(ctx, j.globalCfg.RPCs)
	if err != nil {
		fmt.Printf("failed to get evm client, err: %v\n", err)
		return false, false, err
	}

	currentBlock, err := client.BlockNumber(ctx)
	if err != nil {
		fmt.Printf("failed to get current block, err: %v\n", err)
		return false, false, err
	}

	return ethBlockNumL1L2+l1l2timeout < currentBlock, ethBlockNumL1L2+maxTimeout < currentBlock, nil
}
