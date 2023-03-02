package main

import (
	"fmt"
	"log"
)

func (cli *CLI) mint(to string, nodeId string) {
	to = cli.wallets.GetAddress(to)

	if !ValidateAddress(to) {
		log.Panic("Address is not Valid")
	}

	chain := NewBlockChain(nodeId)
	UTXOSet := UTXOSet{BlockChain: chain}
	defer chain.db.Close()

	cbTx := NewCoinbaseTX(to, "")
	txs := []*Transaction{cbTx}
	block := chain.MineBlock(txs)
	UTXOSet.Update(block)

	fmt.Println("Success!")

}
