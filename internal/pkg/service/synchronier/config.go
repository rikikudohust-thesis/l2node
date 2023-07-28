package synchronier

import (
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/service/txprocessor"

	ethCommon "github.com/ethereum/go-ethereum/common"
)

type config struct {
	StartBlock                  uint64
	EndBlock                    uint64
	L1TxEvent                   ethCommon.Hash
	ForgeBatch                  ethCommon.Hash
	AddToken                    ethCommon.Hash
	UpdateForgeL1L2BatchTimeout ethCommon.Hash
	UpdateFeeAddToken           ethCommon.Hash
	Withdrawal                  ethCommon.Hash
	BlockRangeLimit             uint64
	BatchSize                   uint64
	JobIntervalSec              uint64
	Cfg                         txprocessor.Config
}

var configs = map[uint64]config{
	model.ChainIDEthereum: {
		StartBlock:                  3834530,
		L1TxEvent:                   ethCommon.HexToHash("0xdd5c7c5ea02d3c5d1621513faa6de53d474ee6f111eda6352a63e3dfe8c40119"),
		ForgeBatch:                  ethCommon.HexToHash("0xe00040c8a3b0bf905636c26924e90520eafc5003324138236fddee2d34588618"),
		AddToken:                    ethCommon.HexToHash("0xcb73d161edb7cd4fb1d92fedfd2555384fd997fd44ab507656f8c81e15747dde"),
		UpdateForgeL1L2BatchTimeout: ethCommon.HexToHash("0xff6221781ac525b04585dbb55cd2ebd2a92c828ca3e42b23813a1137ac974431"),
		UpdateFeeAddToken:           ethCommon.HexToHash("0xd1c873cd16013f0dc5f37992c0d12794389698512895ec036a568e393b46e3c1"),
		Withdrawal:                  ethCommon.HexToHash("0x69177d798b38e27bcc4e0338307e4f1490e12d1006729d0e6e9cc82a8732f415"),
		BlockRangeLimit:             1000,
		BatchSize:                   50,
		JobIntervalSec:              1,
		Cfg: txprocessor.Config{
			NLevels:  8,
			MaxFeeTx: 4,
			MaxTx:    16,
			MaxL1Tx:  8,
			ChainID:  1,
		},
	},
}
