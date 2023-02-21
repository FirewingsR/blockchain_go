package main

import (
	"log"

	"github.com/boltdb/bolt"
)

// BlockchainIterator is used to iterate over blockchain blocks
type BlockChainIterator struct {
	curHash []byte
	db      *bolt.DB
}

// Next returns next block starting from the tip
func (iter *BlockChainIterator) Next() *Block {
	var block *Block

	err := iter.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(blocksBucket))
		encodedBlock := b.Get(iter.curHash)
		block = DeserializeBlock(encodedBlock)

		return nil
	})

	if err != nil {
		log.Panic(err)
	}

	iter.curHash = block.PrevBlockHash

	return block
}
