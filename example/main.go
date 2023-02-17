package main

import (
	"fmt"
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
		fmt.Println()
	}
}
