package main

import (
	"fmt"
	"log"
)

func (cli *CLI) createBlockChain(address, nodeID string) {
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := CreateBlockChain(address, nodeID)
	defer bc.db.Close()
	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	fmt.Println("Done!")
}
