package account

import (
	"github.com/rikikudohust-thesis/l2node/internal/pkg/model"
	"github.com/rikikudohust-thesis/l2node/pkg/context"

	"net/http"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/iden3/go-iden3-crypto/babyjub"
	"gorm.io/gorm"
)

const (
	accountRawQuery = `
SELECT accounts_l2.item_id, accounts_l2.idx as idx, accounts_l2.batch_num, 
	accounts_l2.bjj, accounts_l2.eth_addr, tokens.token_id, tokens.item_id AS token_item_id, tokens.eth_block_num AS token_block,
	tokens.eth_addr as token_eth_addr, tokens.name, tokens.symbol, tokens.decimals, 
	account_updates_l2.nonce, account_updates_l2.balance, COUNT(*) OVER() AS total_items
	FROM accounts_l2 INNER JOIN (
		SELECT DISTINCT idx,
		first_value(nonce) OVER w AS nonce,
		first_value(balance) OVER w AS balance
		FROM account_updates_l2
		WINDOW w as (PARTITION BY idx ORDER BY item_id DESC)
	) AS account_updates_l2 ON account_updates_l2.idx = accounts_l2.idx INNER JOIN tokens ON accounts_l2.token_id = tokens.token_id `
)

func SetupRouter(router *gin.RouterGroup, db *gorm.DB, r model.IService) {
	router.GET("/accounts", getAccounts(db))
  router.GET("/accounts/:idx", getAccountByIndex(db))
}

func getAccounts(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.New(c).WithLogPrefix("get-accounts-api")

		var p struct {
			EthAddr  *string  `form:"ethAddr"`
			BJJ      *string  `form:"bjj"`
			TokenIDs []string `form:"tokenIDs"`
		}

		if err := ctx.ShouldBindQuery(&p); err != nil {
			ctx.Errorf("failed to bind params, err: %v", err)
			ctx.AbortWith400(err.Error())
			return
		}

		nextIsAnd := false
		var args []interface{}
		queryRaw := accountRawQuery
		if p.EthAddr != nil {
			queryRaw += `where accounts_l2.eth_addr = ?`
			args = append(args, common.HexToAddress(*p.EthAddr))
			nextIsAnd = true
		} else if p.BJJ != nil {
			queryRaw = accountRawQuery + `where accounts_l2.bjj = ?`
			byteBjj, _ := model.HezStringToBJJ(*p.BJJ, "hez")
			args = append(args, babyjub.PublicKeyComp(*byteBjj))
			nextIsAnd = true
		}

		if len(p.TokenIDs) > 0 {
			if nextIsAnd {
				queryRaw += "AND "
			} else {
				queryRaw += "where "
			}
			queryRaw += "accounts_l2.token_id IN (?) "
			args = append(args, p.TokenIDs)
		}

		var accounts []*accountAPI
		if err := db.Raw(queryRaw, args...).Find(&accounts).Error; err != nil {
			ctx.Errorf("failed to find accounts, err: %v", err)
			ctx.AbortWith500(err.Error())
			return
		}

		accountResponses := make([]*accountResponse, 0, len(accounts))
		for _, account := range accounts {
			accountResponses = append(accountResponses, &accountResponse{
				ItemID:       account.ItemID,
				Idx:          account.Idx,
				BatchNum:     account.BatchNum,
				BJJ:          model.BjjToString(account.BJJ),
				EthAddr:      account.EthAddr,
				TokenID:      account.TokenID,
				TokenBlock:   account.TokenBlock,
				TokenEthAddr: account.TokenEthAddr,
				Name:         account.Name,
				Symbol:       account.Symbol,
				Decimals:     account.Decimals,
				Nonce:        account.Nonce,
				Balance:      account.Balance,
				TotalItems:   account.TotalItems,
			})
		}

		ctx.RespondWith(http.StatusOK, "success", accountResponses)
	}
}

func getAccountByIndex(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := context.New(c).WithLogPrefix("get-accounts-api")

		idx := ctx.Param("idx")
		queryRaw := accountRawQuery + "WHERE accounts.idx = $1"
		var account *accountAPI
		if err := db.Raw(queryRaw, idx).Find(&account).Error; err != nil {
			ctx.Errorf("failed to find accounts, err: %v", err)
			ctx.AbortWith500(err.Error())
			return
		}

		accountResponses := &accountResponse{
			ItemID:       account.ItemID,
			Idx:          account.Idx,
			BatchNum:     account.BatchNum,
			BJJ:          model.BjjToString(account.BJJ),
			EthAddr:      account.EthAddr,
			TokenID:      account.TokenID,
			TokenBlock:   account.TokenBlock,
			TokenEthAddr: account.TokenEthAddr,
			Name:         account.Name,
			Symbol:       account.Symbol,
			Decimals:     account.Decimals,
			Nonce:        account.Nonce,
			Balance:      account.Balance,
			TotalItems:   account.TotalItems,
		}

		ctx.RespondWith(http.StatusOK, "success", accountResponses)
	}
}
