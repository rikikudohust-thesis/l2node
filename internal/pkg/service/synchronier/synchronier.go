package synchronier

import (
	"context"
	"database/sql"
	"fmt"
	"math/big"
	"time"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/database/statedb"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/utils"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/service/txprocessor"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	syncingBlockCacheKey = "syncingBlock"
)

type job struct {
	globalCfg *model.JobConfig
	localCfg  *config
	db        *gorm.DB
	r         model.IService
	sdb       *statedb.StateDB
	processor *txprocessor.TxProcessor
	abi       abi.ABI
}

func NewJob(cfg *model.JobConfig, db *gorm.DB, r model.IService, sdb *statedb.StateDB) model.IJob {
	return &job{
		globalCfg: cfg,
		db:        db,
		r:         r,
		sdb:       sdb,
		abi:       utils.GetAbis("internal/pkg/abis/zkpayment/zkpayment.json"),
	}
}

func (j *job) Run(ctx context.Context) {
	c, existed := configs[j.globalCfg.ChainID]
	if !existed {
		return
	}
	j.localCfg = &c
	j.processor = txprocessor.NewTxProcessor(j.sdb, j.localCfg.Cfg)
	for {
    start := time.Now()
    fmt.Println("Starting")
		j.process(ctx)
    fmt.Println("done, elapsed: ", time.Since(start))
		// time.Sleep(time.Duration(j.localCfg.JobIntervalSec) * time.Second)
		// time.Sleep(time.Duration(500) * time.Millisecond)

	}
}

func (j *job) process(ctx context.Context) {
	var syncingBlock uint64
	if err := j.r.Get(ctx, syncingBlockCacheKey, &syncingBlock); err != nil && !model.IsNilErr(err) {
		fmt.Printf("failed to get syncing block, err: %v \n", err)
		return
	}
	startBlock := j.globalCfg.StartBlock
	if startBlock < syncingBlock {
		startBlock = syncingBlock
	}
	ethBlock := model.Block{
		Num: int64(startBlock),
	}
	fmt.Printf("querying block: %v\n", ethBlock.Num)
	client, err := utils.GetEvmClient(ctx, j.globalCfg.RPCs)
	if err != nil {
		fmt.Printf("failed to get evm client: %v\n", err)
		return
	}
	currentBlock, err := client.BlockNumber(ctx)
	if err != nil {
		fmt.Printf("failed to get current number: %v\n", err)
		return
	}
	confirmedBlock := currentBlock - j.globalCfg.ConfirmedBlocks
	if uint64(ethBlock.Num) > uint64(confirmedBlock) {
		fmt.Printf("wait for block: %v\n", ethBlock)
		return
	}

  currentBatch := j.sdb.CurrentBatch()
	rollupData, syncedBlock, err := j.rollupSync(ctx, &ethBlock)
	if err != nil {
		fmt.Printf("failed to sync rollup, err %v\n", err)
    j.sdb.Reset(currentBatch)
		return
	}
	fmt.Printf("synced: %v\n", syncedBlock)
	if err := j.save(ctx, rollupData); err != nil {
		fmt.Printf("failed to save rollup data, err: %v\n", err)
    j.sdb.Reset(currentBatch)
		return
	}

	if err := j.r.Set(ctx, syncingBlockCacheKey, syncedBlock+1, 0); err != nil && !model.IsNilErr(err) {
		fmt.Printf("failed to set synced, err: %v \n", err)
    j.sdb.Reset(currentBatch)
		return
	}
}

func (j *job) rollupSync(ctx context.Context, ethBlock *model.Block) (*model.RollupData, uint64, error) {
	blockNum := ethBlock.Num
	rollupData := model.NewRollupData()

	nextForgeL1TxsNum, err := j.getLastL1TxsNum()
	if err != nil {
		fmt.Printf("failed to get last forge l1 txs num, err: %v\n", err)
		return nil, 0, err
	}

	rollupEvents, err := j.RollupEventsByBlock(ctx, uint64(blockNum))
	if err != nil {
		fmt.Printf("failed to get events, err: %v\n", err)
		return nil, 0, err
	}

	if rollupEvents == nil {
		return nil, uint64(ethBlock.Num), nil
	}

	rollupData.L1UserTxs, err = getL1UserTx(rollupEvents.L1UserTx, blockNum)
	if err != nil {
		fmt.Printf("failed to get L1UserTx, err: %v\n", err)
		return nil, 0, err
	}

	for _, evtForgeBatch := range rollupEvents.ForgeBatch {
		batchData := model.NewBatchData()
		position := 0

		forgeBatchArgs, sender, err := j.RollupForgeBatchArgs(ctx, evtForgeBatch.EthTxHash, evtForgeBatch.L1UserTxsLen)
		if err != nil {
			fmt.Printf("failed to get RollupForgeBatchArgs, err: %v\n", err)
			return nil, 0, err
		}
		ethTxHash := evtForgeBatch.EthTxHash
		gasUsed := evtForgeBatch.GasUsed
		gasPrice := evtForgeBatch.GasPrice
		batchNum := evtForgeBatch.BatchNum

		var l1UserTxs []model.L1Tx
		if forgeBatchArgs.L1Batch {
			l1UserTxs, err = j.getUnforgedL1UserTxs(nextForgeL1TxsNum)
			fmt.Println(l1UserTxs)
			if err != nil {
				return nil, 0, err
			}
			position = len(l1UserTxs)
		}

		l1TxsAuth := make([]model.AccountCreationAuth, 0, len(forgeBatchArgs.L1CoordinatorTxsAuths))
		batchData.L1CoordinatorTxs = make([]model.L1Tx, 0, len(forgeBatchArgs.L1CoordinatorTxs))

		for i := range forgeBatchArgs.L1CoordinatorTxs {
			l1CoordinatorTx := forgeBatchArgs.L1CoordinatorTxs[i]
			l1CoordinatorTx.Position = position
			// l1CoordinatorTx.ToForgeL1TxsNum = &forgeL1TxsNum
			l1CoordinatorTx.UserOrigin = false
			l1CoordinatorTx.EthBlockNum = uint64(blockNum)
			batchNumUint64 := model.BatchNum(batchNum)
			l1CoordinatorTx.BatchNum = &batchNumUint64
			l1CoordinatorTx.EthTxHash = ethTxHash
			l1Tx, err := model.NewL1Tx(&l1CoordinatorTx)
			if err != nil {
				return nil, 0, err
			}

			batchData.L1CoordinatorTxs = append(batchData.L1CoordinatorTxs, *l1Tx)
			position++

			// Create a slice of account creation auth to be
			// inserted later if not exists
			if l1CoordinatorTx.FromEthAddr != model.RollupConstEthAddressInternalOnly {
				l1CoordinatorTxAuth := forgeBatchArgs.L1CoordinatorTxsAuths[i]
				l1TxsAuth = append(l1TxsAuth, model.AccountCreationAuth{
					EthAddr:   l1CoordinatorTx.FromEthAddr,
					BJJ:       l1CoordinatorTx.FromBJJ,
					Signature: l1CoordinatorTxAuth,
				})
			}
		}

		for i := range forgeBatchArgs.L2TxsData {
			if err := forgeBatchArgs.L2TxsData[i].SetType(); err != nil {
				return nil, 0, err
			}
		}

		poolL2Txs := model.L2TxsToPoolL2Txs(forgeBatchArgs.L2TxsData)

		output, err := j.processor.ProcessTxs(forgeBatchArgs.FeeIdxCoordinator, l1UserTxs, []model.L1Tx{}, poolL2Txs)
		if err != nil {
			fmt.Printf("failed to process txs, err: %v\n", err)
			return nil, 0, err
		}

		// if j.sdb.CurrentBatch() != model.BatchNum(batchNum) {
		// 	fmt.Printf("current not equal, currentBatch: %d, batchNum: %d\n", j.sdb.CurrentBatch(), batchNum)
		// 	return nil, 0, fmt.Errorf("current not equal, currentBatch: %d, batchNum: %d", j.sdb.CurrentBatch(), batchNum)
		// }
		// if j.sdb.MT.Root().BigInt().Cmp(forgeBatchArgs.NewStRoot) != 0 {
		// 	fmt.Printf("root not equal, root: %v, newStRoot: %v\n", j.sdb.MT.Root().String(), forgeBatchArgs.NewStRoot.String())
		// 	return nil, 0, fmt.Errorf("root not equal, root: %v, newStRoot: %v", j.sdb.MT.Root().String(), forgeBatchArgs.NewStRoot.String())
		// }

		l2Txs := make([]model.L2Tx, len(poolL2Txs))
		for i, tx := range poolL2Txs {
			l2Txs[i] = tx.L2Tx()
			if err := l2Txs[i].SetID(); err != nil {
				fmt.Printf("failed to set ID: err: %v\n", err)
				return nil, 0, err
			}

			l2Txs[i].EthBlockNum = uint64(blockNum)
			l2Txs[i].BatchNum = model.BatchNum(batchNum)
			l2Txs[i].Position = position
			position++
		}

		batchData.L2Txs = l2Txs

		for i := range l1UserTxs {
			l1UserTxs[i].BatchNum = model.NewBatchNum(batchNum)
		}
		batchData.L1UserTxs = l1UserTxs
		for i := range output.CreatedAccounts {
			createdAccount := &output.CreatedAccounts[i]
			createdAccount.Nonce = 0
			createdAccount.Balance = big.NewInt(0)
			createdAccount.BatchNum = model.BatchNum(batchNum)
		}
		for i := range output.ExitInfos {
			exit := &output.ExitInfos[i]
			exit.BatchNum = model.BatchNum(batchNum)
		}
		batchData.ExitTree = output.ExitInfos

		batchData.CreatedAccounts = output.CreatedAccounts

		batchData.UpdatedAccounts = make([]model.AccountUpdate, 0,
			len(output.UpdatedAccounts))
		for _, acc := range output.UpdatedAccounts {
			batchData.UpdatedAccounts = append(batchData.UpdatedAccounts,
				model.AccountUpdate{
					EthBlockNum: blockNum,
					BatchNum:    model.BatchNum(batchNum),
					Idx:         acc.Idx,
					Nonce:       acc.Nonce,
					Balance:     acc.Balance,
				})
		}

		batch := model.Batch{
			BatchNum:           model.BatchNum(batchNum),
			EthTxHash:          ethTxHash,
			EthBlockNum:        uint64(blockNum),
			ForgerAddr:         *sender,
			CollectedFees:      output.CollectedFees,
			FeeIdxsCoordinator: forgeBatchArgs.FeeIdxCoordinator,
			StateRoot:          (forgeBatchArgs.NewStRoot),
			NumAccounts:        len(batchData.CreatedAccounts),
			LastIdx:            forgeBatchArgs.NewLastIdx,
			ExitRoot:           (forgeBatchArgs.NewExitRoot),
			GasUsed:            gasUsed,
			GasPrice:           gasPrice,
		}

		nextForgeL1TxsNumCpy := int64(nextForgeL1TxsNum)
		if forgeBatchArgs.L1Batch {
			batch.ForgeL1TxsNum = &nextForgeL1TxsNumCpy
			batchData.L1Batch = true
			nextForgeL1TxsNum++
		}

		batchData.Batch = batch
		rollupData.Batches = append(rollupData.Batches, *batchData)
	}

	for _, addTokenEvt := range rollupEvents.AddToken {
		rollupData.AddedTokens = append(rollupData.AddedTokens, model.Token{
			TokenID:     model.TokenID(addTokenEvt.TokenID),
			EthAddr:     addTokenEvt.TokenAddress,
			EthBlockNum: ethBlock.Num,
		})
	}

	return &rollupData, uint64(ethBlock.Num), nil
}

func (j *job) RollupEventsByBlock(ctx context.Context, blockNum uint64) (*RollupEvents, error) {
	rollupEvent := NewRollUpEvents()
	client, err := utils.GetEvmClient(ctx, j.globalCfg.RPCs)
	if err != nil {
		fmt.Printf("failed to get evm client, err: %v\n", err)
		return nil, err
	}

	eventsQuery := ethereum.FilterQuery{
		FromBlock: big.NewInt(int64(blockNum)),
		ToBlock:   big.NewInt(int64(blockNum)),
		Addresses: []common.Address{j.globalCfg.Contracts.ZKPayment},
		Topics:    [][]common.Hash{},
	}

	logs, err := client.FilterLogs(ctx, eventsQuery)
	if err != nil {
		fmt.Printf("failed to query logs data, err: %v\n", err)
	}

	if len(logs) == 0 {
		return nil, nil
	}

	for _, log := range logs {
		switch log.Topics[0] {
		case j.localCfg.L1TxEvent:
			var l1UserTxAux rollupEventL1UserTxAux
			var l1UserTx RollupEventL1UserTx
			if err := j.abi.UnpackIntoInterface(&l1UserTxAux, "L1UserTxEvent", log.Data); err != nil {
				fmt.Printf("failed to unpack l1txevent, err: %v\n", err)
				return nil, err
			}

			l1Tx, err := model.L1UserTxFromBytes(l1UserTxAux.L1UserTx)
			if err != nil {
				fmt.Printf("failed to unmarshal l1tx from bytes, err: %v\n", err)
			}
			toForgeL1TxsNum := new(big.Int).SetBytes(log.Topics[1][:]).Int64()
			l1Tx.ToForgeL1TxsNum = &toForgeL1TxsNum
			l1Tx.Position = int(new(big.Int).SetBytes(log.Topics[2][:]).Int64())
			l1Tx.UserOrigin = true
			l1UserTx.L1UserTx = *l1Tx
			rollupEvent.L1UserTx = append(rollupEvent.L1UserTx, l1UserTx)
		case j.localCfg.AddToken:
			var addToken RollupEventAddToken
			if err := j.abi.UnpackIntoInterface(&addToken, "AddToken", log.Data); err != nil {
				fmt.Printf("failed to unpack add token event, err: %v\n", err)
				return nil, err
			}
			addToken.TokenAddress = common.BytesToAddress(log.Topics[1].Bytes())
			rollupEvent.AddToken = append(rollupEvent.AddToken, addToken)

		case j.localCfg.ForgeBatch:
			var forgeBatch RollupEventForgeBatch
			if err := j.abi.UnpackIntoInterface(&forgeBatch, "ForgeBatch", log.Data); err != nil {
				fmt.Printf("failed to unpack forge batch logs, err: %v\n", err)
				return nil, err
			}
			forgeBatch.BatchNum = new(big.Int).SetBytes(log.Topics[1][:]).Int64()
			forgeBatch.EthTxHash = log.TxHash
			tx, _, err := client.TransactionByHash(ctx, log.TxHash)
			if err != nil {
				fmt.Printf("failed to get tx, err: %v\n ", err)
				return nil, err
			}
			forgeBatch.GasPrice = tx.GasPrice()
			txReceipt, err := client.TransactionReceipt(ctx, log.TxHash)
			if err != nil {
				fmt.Printf("failed to get receipt, err: %v\n ", err)
				return nil, err
			}
			forgeBatch.GasUsed = txReceipt.GasUsed
			rollupEvent.ForgeBatch = append(rollupEvent.ForgeBatch, forgeBatch)
		case j.localCfg.UpdateForgeL1L2BatchTimeout:
			var updateForgeL1L2BatchTimeout struct {
				NewForgeL1L2BatchTimeout uint8
			}
			if err := j.abi.UnpackIntoInterface(&updateForgeL1L2BatchTimeout, "UpdateForgeL1L2BatchTimeout", log.Data); err != nil {
				return nil, err
			}
			rollupEvent.UpdateForgeL1L2BatchTimeout = append(rollupEvent.UpdateForgeL1L2BatchTimeout, RollupEventUpdateForgeL1L2BatchTimeout{
				NewForgeL1L2BatchTimeout: int64(updateForgeL1L2BatchTimeout.NewForgeL1L2BatchTimeout),
			})
		case j.localCfg.UpdateFeeAddToken:
			var updateFeeAddToken RollupEventUpdateFeeAddToken
			if err := j.abi.UnpackIntoInterface(&updateFeeAddToken, "UpdateFeeAddToken", log.Data); err != nil {
				return nil, err
			}
			rollupEvent.UpdateFeeAddToken = append(rollupEvent.UpdateFeeAddToken, updateFeeAddToken)
		}
	}

	return rollupEvent, nil
}

func getL1UserTx(eventsL1UserTx []RollupEventL1UserTx, blockNum int64) ([]model.L1Tx, error) {
	l1Txs := make([]model.L1Tx, len(eventsL1UserTx))
	for i := range eventsL1UserTx {
		eventsL1UserTx[i].L1UserTx.EthBlockNum = uint64(blockNum)
		// Check validity of L1UserTx
		l1Tx, err := model.NewL1Tx(&eventsL1UserTx[i].L1UserTx)
		if err != nil {
			return nil, err
		}
		l1Txs[i] = *l1Tx
	}
	return l1Txs, nil
}

func (j *job) RollupForgeBatchArgs(ctx context.Context, ethTxHash common.Hash, l1UserTxsLen uint16) (*RollupForgeBatchArgs, *common.Address, error) {
	client, err := utils.GetEvmClient(ctx, j.globalCfg.RPCs)
	if err != nil {
		fmt.Printf("failed to get evm client, err: %v\n", err)
		return nil, nil, err
	}
	tx, _, err := client.TransactionByHash(ctx, ethTxHash)
	if err != nil {
		return nil, nil, err
	}

	txData := tx.Data()

	method, err := j.abi.MethodById(txData[:4])
	if err != nil {
		fmt.Printf("failed to get method, err: %v", method)
		return nil, nil, err
	}

	receipt, err := client.TransactionReceipt(ctx, ethTxHash)
	if err != nil {
		fmt.Printf("failed to get receipt, err: %v", err)
		return nil, nil, err
	}

	sender, err := client.TransactionSender(ctx, tx, receipt.Logs[0].BlockHash, receipt.Logs[0].Index)
	if err != nil {
		fmt.Printf("failed to get sender, err: %v", err)
		return nil, nil, err
	}
	var aux rollupForgeBatchArgsAux
	if values, err := method.Inputs.Unpack(txData[4:]); err != nil {
		return nil, nil, err
	} else if err := method.Inputs.Copy(&aux, values); err != nil {
		return nil, nil, err
	}
	rollupForgeBatchArgs := RollupForgeBatchArgs{
		L1Batch:               aux.L1Batch,
		NewExitRoot:           aux.NewExitRoot,
		NewLastIdx:            aux.NewLastIdx.Int64(),
		NewStRoot:             aux.NewStRoot,
		ProofA:                aux.ProofA,
		ProofB:                aux.ProofB,
		ProofC:                aux.ProofC,
		VerifierIdx:           aux.VerifierIdx,
		L1CoordinatorTxs:      []model.L1Tx{},
		L1CoordinatorTxsAuths: [][]byte{},
		L2TxsData:             []model.L2Tx{},
		FeeIdxCoordinator:     []model.Idx{},
	}

	nlevels := NLEVEL
	lenL1L2TxsBytes := int((nlevels/8)*2 + utils.Float40BytesLength + 1)
	numBytesL1TxUser := int(l1UserTxsLen) * lenL1L2TxsBytes
	numTxsL1Coord := len(aux.EncodedL1CoordinatorTx) / model.RollupConstL1CoordinatorTotalBytes
	numBytesL1TxCoord := numTxsL1Coord * lenL1L2TxsBytes
	numBeginL2Tx := numBytesL1TxCoord + numBytesL1TxUser
	l1UserTxsData := []byte{}
	if l1UserTxsLen > 0 {
		l1UserTxsData = aux.L1L2TxsData[:numBytesL1TxUser]
	}

	for i := 0; i < int(l1UserTxsLen); i++ {
		l1Tx, err := model.L1TxFromDataAvailability(l1UserTxsData[i*lenL1L2TxsBytes:(i+1)*lenL1L2TxsBytes], uint32(nlevels))
		if err != nil {
			return nil, nil, err
		}

		rollupForgeBatchArgs.L1UserTxs = append(rollupForgeBatchArgs.L1UserTxs, *l1Tx)
	}

	l2TxsData := []byte{}
	if numBeginL2Tx < len(aux.L1L2TxsData) {
		l2TxsData = aux.L1L2TxsData[numBeginL2Tx:]
	}

	numTxsL2 := len(l2TxsData) / lenL1L2TxsBytes
	for i := 0; i < numTxsL2; i++ {
		l2Tx, err := model.L2TxFromBytesDataAvailability(l2TxsData[i*lenL1L2TxsBytes:(i+1)*lenL1L2TxsBytes], int(nlevels))
		if err != nil {
			return nil, nil, err
		}

		rollupForgeBatchArgs.L2TxsData = append(rollupForgeBatchArgs.L2TxsData, *l2Tx)
	}

	for i := 0; i < numTxsL1Coord; i++ {
		bytesL1Coordinator :=
			aux.EncodedL1CoordinatorTx[i*model.RollupConstL1CoordinatorTotalBytes : (i+1)*model.RollupConstL1CoordinatorTotalBytes] //nolint:lll
		var signature []byte
		v := bytesL1Coordinator[0]
		s := bytesL1Coordinator[1:33]
		r := bytesL1Coordinator[33:65]
		signature = append(signature, r[:]...)
		signature = append(signature, s[:]...)
		signature = append(signature, v)
		l1Tx, err := model.L1CoordinatorTxFromBytes(bytesL1Coordinator, big.NewInt(int64(j.globalCfg.ChainID)), j.globalCfg.Contracts.ZKPayment)
		if err != nil {
			return nil, nil, err
		}
		rollupForgeBatchArgs.L1CoordinatorTxs = append(rollupForgeBatchArgs.L1CoordinatorTxs, *l1Tx)
		rollupForgeBatchArgs.L1CoordinatorTxsAuths =
			append(rollupForgeBatchArgs.L1CoordinatorTxsAuths, signature)
	}

	lenFeeIdxCoordinatorBytes := int(nlevels / 8) //nolint:gomnd
	numFeeIdxCoordinator := len(aux.FeeIdxCoordinator) / lenFeeIdxCoordinatorBytes
	for i := 0; i < numFeeIdxCoordinator; i++ {
		var paddedFeeIdx [6]byte
		if lenFeeIdxCoordinatorBytes < model.IdxBytesLen {
			copy(paddedFeeIdx[6-lenFeeIdxCoordinatorBytes:],
				aux.FeeIdxCoordinator[i*lenFeeIdxCoordinatorBytes:(i+1)*lenFeeIdxCoordinatorBytes])
		} else {
			copy(paddedFeeIdx[:],
				aux.FeeIdxCoordinator[i*lenFeeIdxCoordinatorBytes:(i+1)*lenFeeIdxCoordinatorBytes])
		}
		feeIdxCoordinator, err := model.IdxFromBytes(paddedFeeIdx[:])
		if err != nil {
			return nil, nil, err
		}
		if feeIdxCoordinator != model.Idx(0) {
			rollupForgeBatchArgs.FeeIdxCoordinator =
				append(rollupForgeBatchArgs.FeeIdxCoordinator, feeIdxCoordinator)
		}
	}
	return &rollupForgeBatchArgs, &sender, nil
}

func (j *job) addNewL1Tx(ctx context.Context, l1txs []model.L1Tx) error {
	txs := make([]model.Tx, 0, len(l1txs))
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
			BatchNum:           l1tx.BatchNum,
			EthBlockNum:        l1tx.EthBlockNum,
			ToForgeL1TxsNum:    l1tx.ToForgeL1TxsNum,
			UserOrigin:         &l1tx.UserOrigin,
			FromEthAddr:        l1tx.FromEthAddr,
			FromBJJ:            l1tx.FromBJJ,
			DepositAmount:      l1tx.DepositAmount,
			DepositAmountFloat: &depositAmountFloat,
			// EthTxHash:          &l1tx.EthTxHash,
			// L1Fee:              l1tx.L1Fee,
		})
	}
	if err := j.db.Clauses(clause.OnConflict{
		DoNothing: true,
	}).CreateInBatches(txs, len(txs)).Error; err != nil {
		fmt.Printf("failed to save L1Tx Pending %v\n", err)
	}
	return nil
}

func (j *job) save(ctx context.Context, rollupData *model.RollupData) error {
	if rollupData == nil {
		return nil
	}
	txs := make([]model.Tx, 0, len(rollupData.L1UserTxs))
	for _, l1tx := range rollupData.L1UserTxs {
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
			BatchNum:           l1tx.BatchNum,
			EthBlockNum:        l1tx.EthBlockNum,
			ToForgeL1TxsNum:    l1tx.ToForgeL1TxsNum,
			UserOrigin:         &l1tx.UserOrigin,
			FromEthAddr:        l1tx.FromEthAddr,
			FromBJJ:            l1tx.FromBJJ,
			DepositAmount:      l1tx.DepositAmount,
			DepositAmountFloat: &depositAmountFloat,
			// EthTxHash:          &l1tx.EthTxHash,
			// L1Fee:              l1tx.L1Fee,
		})
	}
	fmt.Printf("txs: %+v\n", txs)

	tx := j.db.Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	if err := tx.Error; err != nil {
		tx.Rollback()
		fmt.Printf("transaction failed: %+v\n", err)
		return err
	}

	if err := tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}).CreateInBatches(txs, len(txs)).Error; err != nil {
		fmt.Printf("failed to save L1Tx Pending %v\n", err)
		tx.Rollback()
		return err
	}

	if err := tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}).Table("txes_l2").CreateInBatches(txs, len(txs)).Error; err != nil {
		fmt.Printf("failed to save L1Tx Pending %v\n", err)
		tx.Rollback()
		return err
	}

	for i := range rollupData.Batches {
		batch := &rollupData.Batches[i]

		if err := tx.Table("batches").Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(batch.Batch).Error; err != nil {
			fmt.Printf("failed to save Batch %v\n", err)
			tx.Rollback()
			return err
		}

		if len(batch.CreatedAccounts) != 0 {
			if err := tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(batch.CreatedAccounts).Error; err != nil {
				fmt.Printf("failed to save new account %v\n", err)
				tx.Rollback()
				return err
			}
		}

		if len(batch.UpdatedAccounts) != 0 {
			if err := tx.Clauses(clause.OnConflict{
				DoNothing: true,
			}).Create(batch.UpdatedAccounts).Error; err != nil {
				fmt.Printf("failed to save account update %v\n", err)
				tx.Rollback()
				return err
			}
		}

		fmt.Printf("forged l1: %+v\n", batch.L1UserTxs)
		if len(batch.L1UserTxs) != 0 {

			forged_txs := make([]model.Tx, 0, len(batch.L1UserTxs))
			for _, l1tx := range batch.L1UserTxs {
				af := new(big.Float).SetInt(l1tx.Amount)
				amountFloat, _ := af.Float64()
				laf := new(big.Float).SetInt(l1tx.DepositAmount)
				depositAmountFloat, _ := laf.Float64()
				forged_txs = append(forged_txs, model.Tx{
					IsL1:               true,
					TxID:               l1tx.TxID,
					Type:               l1tx.Type,
					Position:           l1tx.Position,
					FromIdx:            l1tx.FromIdx,
					ToIdx:              l1tx.ToIdx,
					Amount:             l1tx.Amount,
					AmountFloat:        amountFloat,
					TokenID:            l1tx.TokenID,
					BatchNum:           l1tx.BatchNum,
					EthBlockNum:        l1tx.EthBlockNum,
					ToForgeL1TxsNum:    l1tx.ToForgeL1TxsNum,
					UserOrigin:         &l1tx.UserOrigin,
					FromEthAddr:        l1tx.FromEthAddr,
					FromBJJ:            l1tx.FromBJJ,
					DepositAmount:      l1tx.DepositAmount,
					DepositAmountFloat: &depositAmountFloat,
					// EthTxHash:          &l1tx.EthTxHash,
					// L1Fee:              l1tx.L1Fee,
				})
			}

			fmt.Printf("forge_txs: %+v\n", forged_txs)
			if len(forged_txs) != len(batch.L1UserTxs) {
				tx.Rollback()
				return nil
			}
			queryUpdateL1tx := tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "id"}},
				DoUpdates: clause.AssignmentColumns([]string{"amount_success", "deposit_amount_success", "effective_from_idx", "batch_num"}),
			})

			if err := queryUpdateL1tx.Create(forged_txs).Error; err != nil {
				fmt.Printf("failed to save l1tx %v\n", err)
				tx.Rollback()
				return err
			}
		}

		//update l2 txs
		l2txs := batch.L2Txs
		txsL2 := make([]*model.Tx, len(batch.L2Txs))
		pooll2txs := model.L2TxsToPoolL2Txs(l2txs)
		for i := 0; i < len(batch.L2Txs); i++ {
			f := new(big.Float).SetInt(l2txs[i].Amount)
			amountFloat, _ := f.Float64()
			txsL2 = append(txsL2, &model.Tx{
				IsL1:     false,
				TxID:     l2txs[i].TxID,
				Type:     l2txs[i].Type,
				Position: l2txs[i].Position,
				FromIdx:  l2txs[i].FromIdx,
				// EffectiveFromIdx: &l2txs[i].FromIdx,
				ToIdx:       l2txs[i].ToIdx,
				TokenID:     l2txs[i].TokenID,
				Amount:      l2txs[i].Amount,
				AmountFloat: amountFloat,
				BatchNum:    &l2txs[i].BatchNum,
				EthBlockNum: l2txs[i].EthBlockNum,
				// L2
				Fee:   &l2txs[i].Fee,
				Nonce: &l2txs[i].Nonce,
			})

			pooll2txs[i].State = model.PoolL2TxStateForged
		}
		queryAddL2Txs := tx.Clauses(clause.OnConflict{
			DoNothing: true,
		})
		if err := queryAddL2Txs.CreateInBatches(txsL2, len(txsL2)).Error; err != nil {
			fmt.Printf("failed to save l2tx, %v\n", err)
			tx.Rollback()
			return err
		}

		//update pooll2tx
		queryUpdatePoolL2Tx := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "tx_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"state"}),
		})

		if err := queryUpdatePoolL2Tx.Table("tx_pool").CreateInBatches(pooll2txs, len(pooll2txs)).Error; err != nil {
			fmt.Printf("failed to save pool_l2_txes %v\n", err)
			tx.Rollback()
			return err
		}
	}

	if err := tx.Clauses(clause.OnConflict{
		DoNothing: true,
	}).CreateInBatches(rollupData.AddedTokens, len(rollupData.AddedTokens)).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (j *job) getLastL1TxsNum() (uint64, error) {
	query := j.db.Select("MAX(forge_l1_txs_num)")
	query = query.Table("batches")
	var lastL1TxsNumPtr sql.NullInt64
	if err := query.Scan(&lastL1TxsNumPtr).Error; err != nil {
		return 0, err
	}
	lastL1TxsNum := uint64(0)
	if lastL1TxsNumPtr.Valid {
		lastL1TxsNum = uint64(lastL1TxsNumPtr.Int64) + 1
	}
	return lastL1TxsNum, nil
}

func (j *job) getUnforgedL1UserTxs(nextForgeL1TxsNum uint64) ([]model.L1Tx, error) {
	var txs []model.L1TxGorm
	query := j.db.Select("id, to_forge_l1_txs_num, position, user_origin, from_idx, from_eth_addr, from_bjj, to_idx, token_id, amount, NULL AS effective__amount, deposit_amount, NULL AS effective_deposit_amount, eth_block_num, type, batch_num")
	query = query.Table("txes")
	query = query.Where("batch_num IS NULL AND to_forge_l1_txs_num = $1", nextForgeL1TxsNum)
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
