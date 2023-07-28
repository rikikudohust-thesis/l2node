package exit

import (
	"math/big"
	"net/http"
	"strconv"

	"github.com/rikikudohust-thesis/l2node/internal/pkg/database/l2db"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model/nonce"

	"github.com/rikikudohust-thesis/l2node/pkg/context"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"gorm.io/gorm"
)

func SetupRouter(router *gin.RouterGroup, db *gorm.DB, r model.IService) {
	router.GET("/exit", getExitDataByEth(db))
}

func getExitData(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.New(c).WithLogPrefix("get-exit-api")
		idx := ctx.Param("idx")
		idxNum, err := strconv.Atoi(idx)
		if err != nil {
			ctx.Infof("invalid idx, err: %v", err)
			ctx.AbortWith500(err.Error())
			return

		}
		account, err := getAccount(ctx, db, model.Idx(idxNum))
		if err != nil {
			ctx.Infof("failed to get account with idx %d: %v", idxNum, err)
			ctx.AbortWith500(err.Error())
			return
		}

		exittree, err := l2db.GetExitTree(db, account.Idx)
		if err != nil {
			ctx.Infof("failed to get exittre with idx %d: %v", idxNum, err)
			ctx.AbortWith500(err.Error())
			return
		}

		exitResponse := make([]*ExitResponse, 0, len(exittree))
		for _, exit := range exittree {
			ctx.Infof("exit tree root: %v", exit.MerkleProof.Root.String())
			siblings := make([]string, 0, len(exit.MerkleProof.Siblings))
			for i := range exit.MerkleProof.Siblings {
				siblings = append(siblings, exit.MerkleProof.Siblings[i].BigInt().String())
			}
			exitResponse := append(exitResponse, &ExitResponse{
				TokenID:         account.TokenID,
				Amount:          exit.Balance,
				BJJ:             account.BJJ.String(),
				NumExitRoot:     uint64(exit.BatchNum),
				Siblings:        siblings,
				Idx:             account.Idx,
				InstantWithdraw: false,
			})

			ctx.RespondWith(http.StatusOK, "success", exitResponse)
		}
	}
}

func getExitDataByEth(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.New(c).WithLogPrefix("get-exit-api-by-bjj-and-eth")

		var p struct {
			EthAddr string `form:"ethAddr"`
		}
		if err := ctx.ShouldBindQuery(&p); err != nil {
			ctx.Errorf("faild to bind query, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}
		ctx.Infoln(p.EthAddr)
		query := ` select exit_trees.*, accounts.eth_addr, accounts.bjj ,accounts.token_id 
		from exit_trees 
		inner join txes on exit_trees.account_idx = txes.from_idx 
		inner join accounts on accounts.idx = exit_trees.account_idx 
		where txes.type = 'ForceExit' and accounts.eth_addr = ? and exit_trees.instant_withdrawn is null;`
		var resp []model.ExitInfoGormV2
		if err := db.Raw(query, common.HexToAddress(p.EthAddr)).Find(&resp).Error; err != nil {
			ctx.Errorf("faild to get exit data, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		exitResponse := make([]*ExitResponse, 0, len(resp))
		for _, exit := range resp {
			ctx.Infof("exit tree root: %v", exit.MerkleProof.Root.String())
			siblings := make([]string, 0, len(exit.MerkleProof.Siblings))
			for i := range exit.MerkleProof.Siblings {
				siblings = append(siblings, exit.MerkleProof.Siblings[i].BigInt().String())
			}
			exitResponse = append(exitResponse, &ExitResponse{
				TokenID:         exit.TokenID,
				Amount:          exit.Balance,
				BJJ:             exit.BJJ.String(),
				NumExitRoot:     uint64(exit.BatchNum),
				Siblings:        siblings,
				Idx:             exit.AccountIdx,
				InstantWithdraw: false,
			})
		}

		ctx.RespondWith(http.StatusOK, "success", exitResponse)
	}
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
