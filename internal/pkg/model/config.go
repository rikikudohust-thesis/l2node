package model

import "github.com/ethereum/go-ethereum/common"

type JobConfig struct {
	Network         string
	ChainID         uint64
	ConfirmedBlocks uint64
	BlockPerDay     uint64
	NativeToken     string
	WrapToken       string
	Multicall       string
	Jobs            []string
	RPCs            []string
	Contracts       Contract
	StartBlock      uint64
	ZKPayment       string
}

type Contract struct {
	ZKPayment common.Address
}
