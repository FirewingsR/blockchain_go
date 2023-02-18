package main

import (
	"fmt"
	"time"
)

// Blockchain keeps a sequence of Blocks
type BlockChain struct {
	blocks []*Block
}

// AddBlock saves provided data as a block in the blockchain
func (bc *BlockChain) AddBlock(data string) {
	a := time.Now().UnixMilli()
	prevBlock := bc.blocks[len(bc.blocks)-1]
	newBlock := NewBlock(data, prevBlock.Hash)
	bc.blocks = append(bc.blocks, newBlock)
	b := time.Now().UnixMilli()
	fmt.Printf("add a block using time is %d ms\n", b-a)
}

// NewBlockchain creates a new Blockchain with genesis Block
func NewBlockChain() *BlockChain {
	return &BlockChain{[]*Block{NewGenesisBlock()}}
}
