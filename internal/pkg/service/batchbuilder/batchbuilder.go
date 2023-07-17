package batchbuilder

import (
	"context"
	"crypto/ecdsa"
	"database/sql"
	"math/big"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/crypto"

	"encoding/hex"
	"encoding/json"
	"fmt"

	"time"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/database/statedb"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/service/txprocessor"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/utils"

	"github.com/iden3/go-rapidsnark/prover"
	"github.com/iden3/go-rapidsnark/witness"
	"gorm.io/gorm"
)

const (
	forgingBlockCacheKey = "forging"
	l1l2BatchTimeout     = 10
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
type ProofService struct {
	ZKEY              []byte
	WitnessCalculator *witness.Circom2WitnessCalculator
}

func NewProofService() *ProofService {
	return &ProofService{
		ZKEY:              utils.GetZkey("internal/pkg/zkey/circuit_16_8_8_4.zkey"),
		WitnessCalculator: utils.NewWasmCalculate("internal/pkg/wasm/circuit_16_8_8_4.wasm"),
	}
}

func (ps *ProofService) CalculateProof(input *model.ZKInputs) (*Proof, *PublicInputs, error) {
	inputBytes, err := input.MarshalJSON()
	if err != nil {
		return nil, nil, err
	}
	zkInput, err := witness.ParseInputs(inputBytes)
	if err != nil {
		fmt.Printf("failed to parse input, err: %v\n", err)
		return nil, nil, err
	}
	witnessData, err := ps.WitnessCalculator.CalculateWTNSBin(zkInput, false)
	if err != nil {
		fmt.Printf("failed to calculator witness, err: %v\n", err)
		return nil, nil, err
	}

	proofString, publicInputsString, err := prover.Groth16ProverRaw(ps.ZKEY, witnessData)
	if err != nil {
		fmt.Printf("failed to calculate proof, err: %v\n", err)
		return nil, nil, err
	}
	var proof Proof
	if err := json.Unmarshal([]byte(proofString), &proof); err != nil {
		return nil, nil, err
	}
	var pubInputs PublicInputs
	if err := json.Unmarshal([]byte(publicInputsString), &pubInputs); err != nil {
		return nil, nil, err
	}
	return &proof, &pubInputs, nil
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
	// j.sdb.Reset(11)
	// return
	c, existed := configs[j.globalCfg.ChainID]
	if !existed {
		return
	}
	j.localCfg = &c
	j.processor = txprocessor.NewTxProcessor(j.sdb, j.localCfg.Cfg)
	for {
		j.process(ctx)
		time.Sleep(time.Duration(j.localCfg.JobIntervalSec) * time.Second)
	}
}

func (j *job) process(ctx context.Context) {
	currentBatch := j.sdb.CurrentBatch()
	batchSynced, _, err := j.getSyncBatch(ctx)

	if err != nil {
		return
	}

	if currentBatch != model.BatchNum(batchSynced) {
		fmt.Printf("forging genesis block, currentBatch: %d, batchSync: %d\n", currentBatch, batchSynced)
		return
	}
	currentBlockL2, err := j.getCurrentBlockL2(ctx)
	if err != nil {
		fmt.Printf("failed to get current Block l2, err: %v\n", err)
		return
	}

	if currentBatch >= model.BatchNum(currentBlockL2) {
		fmt.Println(currentBlockL2)
		return
	}

	blockInfo, err := j.getBlockInfor(ctx, currentBatch+1)
	if err != nil {
		return
	}

	// batchTxs, err := j.getBatch(ctx, currentBatch+1)
	// if err != nil {
	// 	fmt.Printf("failed to get batch txs, err: %v\n", err)
	// 	return
	// }
	//
	// l1txs := make([]model.L1Tx, 0)
	// pooll2txs := make([]model.PoolL2Tx, 0)
	// for _, batchTx := range batchTxs {
	// 	if batchTx.IsL1 {
	// 		l1tx, err := batchTx.L1Tx()
	// 		if err != nil {
	// 			fmt.Printf("failed to convert batchTxs to l1Tx, err: %v\n", err)
	// 		}
	// 		l1txs = append(l1txs, *l1tx)
	// 		continue
	// 	}
	// 	var pooll2tx []model.PoolL2Tx
	// 	query := j.db.Table("tx_pool")
	// 	query = query.Where("tx_id = ?", batchTx.TxID)
	// 	if err := query.Find(&pooll2tx).Error; err != nil {
	// 		fmt.Printf("failed to get l2 data, err %v\n: ", err)
	// 		return
	// 	}
	// 	pooll2txs = append(pooll2txs, pooll2tx...)
	// }
	l1txs, pooll2txs, err := j.getBatchTxs(ctx, currentBatch+1)
	if err != nil {
		fmt.Printf("failed to get txs of batch %d\n", currentBatch+1)
		return
	}

	if err := j.forgeBatchOnChain(ctx, l1txs, pooll2txs, blockInfo.IsL1); err != nil {
		fmt.Printf("failed to forge batch on chain, err: %v\n", err)
		j.sdb.Reset(currentBatch)
		return
	}
}

func (j *job) forgeBatch(ctx context.Context, cfg *model.JobConfig, proof *Proof, zkInput *model.ZKInputs, l1txs []model.L1Tx, l2txs []model.L2Tx, l2txs_pool []model.PoolL2Tx, l1Batch bool) error {
	prvKey, err := crypto.HexToECDSA(os.Getenv("GOVERNANCE_PRIVATE_KEY"))
	if err != nil {
		fmt.Printf("failed to get private key, err: %v\n", prvKey)
		return err
	}
	_, l1L2TxsData, _, err := j.RollupForgeBatch(nil, l1txs, l2txs, []model.Idx{})
	if err != nil {
		fmt.Printf("failed to generate l1l2TxsData, err: %v\n", err)
		return err
	}

	publicKey := prvKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		fmt.Println("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
		return fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}
	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	proofA := [2]*big.Int{proof.PiA[0], proof.PiA[1]}
	proofB := [2][2]*big.Int{{proof.PiB[0][1], proof.PiB[0][0]}, {proof.PiB[1][1], proof.PiB[1][0]}}
	proofC := [2]*big.Int{proof.PiC[0], proof.PiC[1]}

	hash, _ := zkInput.HashGlobalData()
	fmt.Printf("hash: %v", hash)
	fmt.Printf("last idx raw: %+v\n", zkInput.Metadata.NewLastIdxRaw.BigInt())
	fmt.Printf("last state raw: %+v\n", zkInput.Metadata.NewStateRootRaw.BigInt())
	fmt.Printf("last exit raw: %+v\n", zkInput.Metadata.NewExitRootRaw.BigInt())
	fmt.Printf("l1L2TxsData: %x\n", l1L2TxsData)
	fmt.Printf("l1Batch: %v\n", l1Batch)
	fmt.Printf("proofA: %+v\n", proofA)
	fmt.Printf("proofB: %+v\n", proofB)
	fmt.Printf("proofC: %+v\n", proofC)

	data, err := j.abi.Pack(
		"forgeBatch",
		zkInput.Metadata.NewLastIdxRaw.BigInt(),
		zkInput.Metadata.NewStateRootRaw.BigInt(),
		zkInput.Metadata.NewExitRootRaw.BigInt(),
		[]byte{},
		l1L2TxsData,
		[]byte{},
		uint8(0),
		l1Batch,
		proofA,
		proofB,
		proofC,
	)
	fmt.Printf("Data: %v\n", hex.EncodeToString(data))

	if err != nil {
		fmt.Printf("failed to pack data, %v\n", err)
		return err
	}

	chain_id := 421613
	tx, err := utils.SendTx(ctx, prvKey, int64(chain_id), j.globalCfg.RPCs, address, j.globalCfg.Contracts.ZKPayment, data)
	if err != nil {
		fmt.Printf("failed to send tx, err: %v\n", err)
		return err
	}
	fmt.Printf("txs: %+v", tx)
	return nil
}

func (j *job) getL1L2TxSelection() ([]model.Idx, [][]byte, []model.L1Tx,
	[]model.L1Tx, []model.PoolL2Tx, []model.PoolL2Tx, error) {
	var l1txs []model.L1Tx
	query := j.db.Table("txes")
	query = query.Where("is_l1 = ?", true)
	if err := query.Find(&l1txs).Error; err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	return []model.Idx{}, [][]byte{}, l1txs, []model.L1Tx{}, []model.PoolL2Tx{}, []model.PoolL2Tx{}, nil
}

func (j *job) RollupForgeBatch(l1CoordinatorTxs []model.L1Tx, l1UserTxs []model.L1Tx, l2TxsData []model.L2Tx, feeId []model.Idx) ([]byte, []byte, []byte, error) {
	nLevels := 8
	lenBytes := nLevels / 8 //nolint:gomnd
	// newLastIdx := big.NewInt(int64(args.NewLastIdx))
	// L1CoordinatorBytes
	// var l1CoordinatorBytes []byte
	// for i := 0; i < len(L1CoordinatorTxs); i++ {
	// 	l1 := L1CoordinatorTxs[i]
	// 	bytesl1, err := l1.BytesCoordinatorTx(args.L1CoordinatorTxsAuths[i])
	// 	if err != nil {
	// 		return err
	// 	}
	// 	l1CoordinatorBytes = append(l1CoordinatorBytes, bytesl1[:]...)
	// }
	// L1L2TxData
	var l1l2TxData []byte
	for i := 0; i < len(l1UserTxs); i++ {
		l1User := l1UserTxs[i]
		bytesl1User, err := l1User.BytesDataAvailability(uint32(nLevels))
		if err != nil {
			return nil, nil, nil, err
		}
		l1l2TxData = append(l1l2TxData, bytesl1User[:]...)
	}
	for i := 0; i < len(l1CoordinatorTxs); i++ {
		l1Coord := l1CoordinatorTxs[i]
		bytesl1Coord, err := l1Coord.BytesDataAvailability(uint32(nLevels))
		if err != nil {
			return nil, nil, nil, err
		}
		l1l2TxData = append(l1l2TxData, bytesl1Coord[:]...)
	}
	for i := 0; i < len(l2TxsData); i++ {
		l2 := l2TxsData[i]
		bytesl2, err := l2.BytesDataAvailability(uint32(nLevels))
		if err != nil {
			return nil, nil, nil, err
		}
		l1l2TxData = append(l1l2TxData, bytesl2[:]...)
	}
	// FeeIdxCoordinator
	var feeIdxCoordinatorByte []byte
	if len(feeIdxCoordinatorByte) > model.RollupConstMaxFeeIdxCoordinator {
		return nil, nil, nil, nil
	}
	for i := 0; i < model.RollupConstMaxFeeIdxCoordinator; i++ {
		feeIdx := model.Idx(0)
		if i < len(feeId) {
			feeIdx = feeId[i]
		}
		bytesFeeIdx, err := feeIdx.Bytes()
		if err != nil {
			return nil, nil, nil, err
		}
		feeIdxCoordinatorByte = append(feeIdxCoordinatorByte, bytesFeeIdx[len(bytesFeeIdx)-int(lenBytes):]...)
	}

	return nil, l1l2TxData, feeIdxCoordinatorByte, nil
}

func (j *job) getFutureForgeL1txs(ctx context.Context) ([]model.L1Tx, error) {
	var txs []model.L1Tx
	query := j.db.Select("tx_id, min(to_forge_l1_txs_num), position, user_origin, from_idx, from_eth_addr, from_bjj, to_idx, token_id, amount, NULL AS effective__amount, deposit_amount, NULL AS effective_deposit_amount, eth_block_num, type, batch_num")
	query = query.Table("txes")
	query = query.Where("batch_num IS NULL")
	query = query.Group("tx_id, to_forge_l1_txs_num, position, user_origin, from_idx, from_eth_addr, from_bjj, to_idx, token_id, amount, deposit_amount, eth_block_num, type, batch_num")
	query = query.Order("position")
	if err := query.Find(&txs).Error; err != nil {
		return []model.L1Tx{}, err
	}
	return txs, nil
}

func (j *job) getBatchData(ctx context.Context) ([]model.L1Tx, []model.L2Tx, []model.PoolL2Tx, error) {
	var txs []model.Tx
	currentBatch := j.sdb.CurrentBatch()
	query := j.db.Select("tx_id,is_l1 ,to_forge_l1_txs_num, position, user_origin, from_idx, from_eth_addr, from_bjj, to_idx, token_id, amount, NULL AS effective__amount, deposit_amount, NULL AS effective_deposit_amount, eth_block_num, type, batch_num")
	query = query.Table("batch_l2")
	query = query.Where("batch_num = ?", currentBatch+1)
	if err := query.Find(&txs).Error; err != nil {
		return nil, nil, nil, err
	}
	l1txs := make([]model.L1Tx, 0)
	l2txs := make([]model.L2Tx, 0)

	for _, tx := range txs {
		if tx.IsL1 {
			l1tx, err := tx.L1Tx()
			if err != nil {
				fmt.Printf("faild to get l1 tx Batch data, err: %v\n", err)
				return nil, nil, nil, err
			}
			l1txs = append(l1txs, *l1tx)
			continue
		}
		l2txs = append(l2txs, model.L2Tx{
			TxID:    tx.TxID,
			FromIdx: tx.FromIdx,
			ToIdx:   tx.ToIdx,
			// Nonce:   *tx.Nonce,
		})
	}

	return l1txs, l2txs, model.L2TxsToPoolL2Txs(l2txs), nil
}

func (j *job) getCurrentBatch(ctx context.Context) (uint64, bool) {
	var nextBatchPtr sql.NullInt64
	query1 := j.db.Select("MAX(to_forge_l1_txs_num)")
	query1 = query1.Table("batch_l2")
	if err := query1.Scan(&nextBatchPtr).Error; err != nil {
		fmt.Printf("failed to get last l1 forge, %v\n", err)
	}

	currentBatch := uint64(0)
	if !nextBatchPtr.Valid {
		return 0, true
	}
	return currentBatch, false
}

func (j *job) getSyncBatch(ctx context.Context) (uint64, bool, error) {
	var currentBatchPtr sql.NullInt64
	query1 := j.db.Select("MAX(batch_num)")
	query1 = query1.Table("batches")
	if err := query1.Scan(&currentBatchPtr).Error; err != nil {
		fmt.Printf("failed to get last batch sync, %v\n", err)
		return 0, false, err
	}

	if !currentBatchPtr.Valid {
		return 0, true, nil
	}
	return uint64(currentBatchPtr.Int64), false, nil
}

func (j *job) getBatch(ctx context.Context, batchNum model.BatchNum) ([]model.Tx, error) {
	var txs []*model.TxGorm
	query1 := j.db.Table("txes_l2")
	query1 = query1.Where("batch_num = ?", batchNum)
	if err := query1.Find(&txs).Error; err != nil {
		fmt.Printf("faild to get batch l2, %v\n", err)
		return nil, err
	}
	txdb := make([]model.Tx, 0, len(txs))
	for _, tx := range txs {
		txdb = append(txdb, *tx.ToTx())
	}
	return txdb, nil
}

func (j *job) getBatchTxs(ctx context.Context, batchNum model.BatchNum) ([]model.L1Tx, []model.PoolL2Tx, error) {

	var txs []*model.L1TxGorm
	query := j.db.Select("txes.id, txes.is_l1, txes.to_forge_l1_txs_num, txes.position, txes.user_origin, txes.from_idx, txes.from_eth_addr, txes.from_bjj, txes.to_idx, txes.token_id, txes.amount, NULL AS effective_amount, txes.deposit_amount, NULL AS effective_deposit_amount, txes.eth_block_num, txes.type, txes.batch_num")
	query = query.Table("txes_l2, txes")
	query = query.Where("txes_l2.batch_num = ? and txes.id = txes_l2.id", batchNum)
	query = query.Order("txes.position")
	if err := query.Find(&txs).Error; err != nil {
		return nil, nil, err
	}
	l1txs := make([]model.L1Tx, 0, len(txs))
	for _, l1g := range txs {
		l1txs = append(l1txs, *l1g.ToL1Tx())
	}
	var txsGorm []*model.PoolL2TxGorm
	// 	queryPoolL2Tx := j.db.Select(`tx_pool.tx_id, from_idx, to_idx, tx_pool.to_eth_addr,
	// tx_pool.to_bjj, tx_pool.token_id, tx_pool.amount, tx_pool.fee, tx_pool.nonce,
	// tx_pool.state, tx_pool.info, tx_pool.signature, tx_pool.timestamp, rq_from_idx,
	// rq_to_idx, tx_pool.rq_to_eth_addr, tx_pool.rq_to_bjj, tx_pool.rq_token_id, tx_pool.rq_amount,
	// tx_pool.rq_fee, tx_pool.rq_nonce, tx_pool.tx_type, tx_pool.rq_offset, tx_pool.atomic_group_id, tx_pool.max_num_batch`)
	queryPoolL2Tx := j.db.Table(`(SELECT tx_pool.item_id, tx_pool.tx_id, from_idx, to_idx, tx_pool.to_eth_addr,
tx_pool.to_bjj, tx_pool.token_id, tx_pool.amount, tx_pool.fee, tx_pool.nonce,
tx_pool.state, tx_pool.info, tx_pool.signature, tx_pool.timestamp, rq_from_idx,
rq_to_idx, tx_pool.rq_to_eth_addr, tx_pool.rq_to_bjj, tx_pool.rq_token_id, tx_pool.rq_amount,
tx_pool.rq_fee, tx_pool.rq_nonce, tx_pool.tx_type, tx_pool.rq_offset, tx_pool.atomic_group_id, tx_pool.max_num_batch FROM tx_pool INNER JOIN tokens ON tx_pool.token_id = tokens.token_id WHERE state = 'fing') as tx_pool
inner join txes_l2 on txes_l2.id = tx_pool.tx_id`)
	queryPoolL2Tx = queryPoolL2Tx.Where("batch_num = ?", batchNum)
	queryPoolL2Tx = queryPoolL2Tx.Order("tx_pool.item_id")

	if err := queryPoolL2Tx.Find(&txsGorm).Error; err != nil {
		fmt.Printf("failed to get l2 pending, %v\n", err)
		return nil, nil, err
	}

	pooll2txs := make([]model.PoolL2Tx, 0, len(txsGorm))
	for _, txg := range txsGorm {
		pooll2txs = append(pooll2txs, *txg.ToPoolL2Tx())
	}

	return l1txs, pooll2txs, nil
}

func (j *job) emptyForge(ctx context.Context) error {
	output1, err := j.processor.ProcessTxs(nil, nil, nil, nil)
	if err != nil {
		fmt.Printf("failed to process txs, %v\n", err)
		return err
	}

	proofSV := NewProofService()
	proofData0, _, err := proofSV.CalculateProof(output1.ZKInputs)
	if err != nil {
		fmt.Printf("failed to calculate proof, %v\n", err)
		return err
	}
	l1Batch := true
	if err := j.forgeBatch(ctx, j.globalCfg, proofData0, output1.ZKInputs, nil, nil, nil, l1Batch); err != nil {
		fmt.Printf("failed to forgeBatch, err: %v\n", err)
		return err
	}
	fmt.Println("Done")
	return nil
}

func (j *job) forgeBatchOnChain(ctx context.Context, l1txs []model.L1Tx, pooll2txs []model.PoolL2Tx, l1Batch bool) error {
	output1, err := j.processor.ProcessTxs(nil, l1txs, nil, pooll2txs)
	if err != nil {
		fmt.Printf("failed to process txs, %v\n", err)
		return err
	}
	proofSV := NewProofService()
	proofData0, _, err := proofSV.CalculateProof(output1.ZKInputs)
	if err != nil {
		fmt.Printf("failed to calculate proof, %v\n", err)
		return err
	}
	l2txs := make([]model.L2Tx, 0, len(pooll2txs))
	for _, pooll2tx := range pooll2txs {
		l2txs = append(l2txs, pooll2tx.L2Tx())
	}

	if err := j.forgeBatch(ctx, j.globalCfg, proofData0, output1.ZKInputs, l1txs, l2txs, nil, l1Batch); err != nil {
		fmt.Printf("failed to forgeBatch, err: %v\n", err)
		return err
	}
	fmt.Println("Done")
	return nil
}

func (j *job) getLastL1L2Timeout(ctx context.Context) (uint64, error) {
	var lastL1L2TimeOutPtr sql.NullInt64
	query1 := j.db.Select("MAX(batch_num)")
	query1 = query1.Table("(SELECT * FROM batch_l2 WHERE is_l1=true) as l2Batch")
	if err := query1.Scan(&lastL1L2TimeOutPtr).Error; err != nil {
		fmt.Printf("failed to get last l1 forge, %v\n", err)
		return 0, err
	}

	if !lastL1L2TimeOutPtr.Valid {
		return 0, nil
	}
	return uint64(lastL1L2TimeOutPtr.Int64), nil

}

func (j *job) getCurrentBlockL2(ctx context.Context) (uint64, error) {
	var lastBlockL2Ptr sql.NullInt64
	query := j.db.Select("MAX(batch_num)")
	query = query.Table("block_l2")
	if err := query.Scan(&lastBlockL2Ptr).Error; err != nil {
		fmt.Printf("faild to get last block l2")
		return 0, err
	}
	if lastBlockL2Ptr.Valid {
		return uint64(lastBlockL2Ptr.Int64), nil
	}
	return 0, nil
}

func (j *job) getBlockInfor(ctx context.Context, batch model.BatchNum) (*model.BlockL2, error) {
	var blockL2 model.BlockL2
	query := j.db.Table("block_l2")
	query = query.Where("batch_num = ?", batch)

	if err := query.Find(&blockL2).Error; err != nil {
		return nil, err
	}
	return &blockL2, nil
}
