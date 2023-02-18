package main

import (
	"time"
)

// Block keeps block headers
type Block struct {
	PrevBlockHash []byte
	Timestamp     int64
	Hash          []byte
	Data          []byte
	Nonce         int
}

// NewBlock creates and returns Block
func NewBlock(data string, prevBlockHash []byte) *Block {
	block := &Block{prevBlockHash, time.Now().Unix(), []byte{}, []byte(data), 0}

	pow := NewProofOfWork(block)
	nonce, hash := pow.Run()
	block.Hash = hash[:]
	block.Nonce = nonce
	return block
}

// NewGenesisBlock creates and returns genesis Block
func NewGenesisBlock() *Block {
	return NewBlock("Genesis Block", []byte{})
}
