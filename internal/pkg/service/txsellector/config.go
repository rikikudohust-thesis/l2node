package txsellector

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
	BlockRangeLimit             uint64
	BatchSize                   uint64
	JobIntervalSec              uint64
	Cfg                         txprocessor.Config
}

var configs = map[uint64]config{
	model.ChainIDEthereum: {
		StartBlock:     3708390,
		JobIntervalSec: 20,
		Cfg: txprocessor.Config{
			NLevels:  8,
			MaxFeeTx: 4,
			MaxTx:    16,
			MaxL1Tx:  8,
			ChainID:  1,
		},
	},
}
