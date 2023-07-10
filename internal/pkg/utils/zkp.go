package utils

import (
	"os"

	"github.com/iden3/go-rapidsnark/witness"
)

func GetZkey(file string) []byte {
	byteData, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return byteData
}

func GetWasm(file string) []byte {
	byteData, err := os.ReadFile(file)
	if err != nil {
		panic(err)
	}
	return byteData
}

func NewWasmCalculate(file string) *witness.Circom2WitnessCalculator {
	wasmBytes := GetWasm(file)
	witnessCalculator, err := witness.NewCircom2WitnessCalculator(wasmBytes, false)
	if err != nil {
		panic(err)
	}
	return witnessCalculator
}
