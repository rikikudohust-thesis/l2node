package utils

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

func GetEvmClient(ctx context.Context, rpcs []string) (*ethclient.Client, error) {
	rpcsLen := len(rpcs)
	indexRand, err := rand.Int(rand.Reader, big.NewInt(int64(rpcsLen)))
	if err != nil {
		return nil, err
	}
	index := int(indexRand.Int64())

	for i := 0; i < rpcsLen; i++ {
		rpc := rpcs[(index+i)%rpcsLen]
		client, err := ethclient.Dial(rpc)
		if err == nil {
			return client, nil
		}
		fmt.Printf("failed to connect %s, err: %v", rpc, err)
	}

	return nil, fmt.Errorf("failed to connect any rpcs")
}

func SendTx(
	ctx context.Context,
	prvKey *ecdsa.PrivateKey, chainID int64, rpcs []string,
	from, to common.Address, data []byte,
) (*types.Transaction, error) {
	client, err := GetEvmClient(ctx, rpcs)
	if err != nil {
		fmt.Printf("failed to get evm client, err: %v", err)
		return nil, err
	}

	nonce, err := client.PendingNonceAt(ctx, from)
	if err != nil {
		fmt.Printf("failed to get nonce, err: %v", err)
		return nil, err
	}

	gasPrice, err := client.SuggestGasPrice(ctx)
	if err != nil {
		fmt.Printf("failed to get gas price, err: %v", err)
		return nil, err
	}

	gasLimit, err := client.EstimateGas(ctx, ethereum.CallMsg{
		To:       &to,
		GasPrice: gasPrice,
		Data:     data,
	})
	if err != nil {
		fmt.Printf("failed to estimate gas, err: %v", err)
		return nil, err
	}

	tx := types.NewTransaction(nonce, to, big.NewInt(0), gasLimit, gasPrice, data)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(big.NewInt(chainID)), prvKey)
	if err != nil {
		fmt.Printf("failed to sign tx, err: %v", err)
		return nil, err
	}
  fmt.Println("tx: ", tx)
  fmt.Println("Hash: ", signedTx.Hash().Hex())

	// signer := types.LatestSignerForChainID(big.NewInt(chainID))
	// signedTxn, err := types.SignNewTx(prvKey, signer, &types.LegacyTx{
	// 	Nonce:    nonce,
	// 	To:       &to,
	// 	GasPrice: gasPrice,
	// 	Gas:      gasLimit,
	// 	Data:     data,
	// })

	// if err != nil {
	// 	fmt.Printf("failed to sign tx, err: %v", err)
	// 	return nil, err
	// }

	if err = client.SendTransaction(ctx, signedTx); err != nil {
		fmt.Printf("failed to send tx, err: %v", err)
		return nil, err
	}

	return signedTx, err
}
