package utils

import (
	"bytes"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi"
)

func GetAbis(filename string) abi.ABI {
	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var abiInterface abi.ABI
	abiInterface, err = abi.JSON(bytes.NewReader(data))
	if err != nil {
		panic(err)
	}

	return abiInterface
}
