package main

import (
	"fmt"
	"math/big"
	"strconv"
)

func main() {

	fmt.Println("main")

	bc := NewBlockChain()
	bc.AddBlock("Send 1 BTC To Densey")
	bc.AddBlock("Send 2 more BTC To Densey")

	for _, block := range bc.blocks {
		fmt.Printf("Prev. hash: %x\n", block.PrevBlockHash)
		fmt.Printf("Data: %s\n", block.Data)
		fmt.Printf("Hash: %x\n", block.Hash)
		fmt.Printf("Nonce: %x\n", block.Nonce)
		pow := NewProofOfWork(block)
		fmt.Printf("PoW: %s\n", strconv.FormatBool(pow.Validate()))
		fmt.Println()
	}
}

func main2() {

	printTarget(8)
	printTarget(256 - targetBits)

}

func printTarget(targetBits uint) {
	target := big.NewInt(1)
	target.Lsh(target, targetBits)
	fmt.Printf("%b\n", target)
}
