package main

import (
	"fmt"
	"log"
)

func (cli *CLI) createBlockChain(alias, nodeID string) {
	address := cli.wallets.GetAddress(alias)
	fmt.Printf("address: %s\n", address)
	if !ValidateAddress(address) {
		log.Panic("ERROR: Address is not valid")
	}
	bc := CreateBlockChain(address, nodeID)
	defer bc.db.Close()
	UTXOSet := UTXOSet{bc}
	UTXOSet.Reindex()

	fmt.Println("Done!")
}
