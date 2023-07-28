package exit

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
)

type ExitResponse struct {
	TokenID         model.TokenID `json:"tokenID"`
	Amount          string        `json:"amount"`
	BJJ             string        `json:"bjj"`
	NumExitRoot     uint64        `json:"numExitRoot"`
	Siblings        []string      `json:"siblings"`
	Idx             model.Idx     `json:"idx"`
	InstantWithdraw bool          `json:"instantWithdraw"`
}

type ExitResp struct {
	TokenID         model.TokenID  `json:"tokenID"`
	Amount          string         `json:"amount"`
	BJJ             string         `json:"bjj"`
	NumExitRoot     uint64         `json:"numExitRoot"`
	Siblings        []string       `json:"siblings"`
	Idx             model.Idx      `json:"idx"`
	InstantWithdraw bool           `json:"instantWithdraw"`
	FromEthAddr     common.Address `json:"ethAddr"`
}
