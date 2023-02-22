package main

import (
	"fmt"
	"log"
)

func (cli *CLI) send(from, to string, amount int) {
	if !ValidateAddress(from) {
		log.Panic("ERROR: Sender address is not valid")
	}
	if !ValidateAddress(to) {
		log.Panic("ERROR: Recipient address is not valid")
	}

	bc := NewBlockChain() // ContinueBlockChain // 从数据库接收区块链
	UTXOSet := UTXOSet{bc}
	defer bc.db.Close()

	tx := NewUTXOTransaction(from, to, amount, &UTXOSet) // 同时创建一个发送交易
	cbTx := NewCoinbaseTX(from, "")                      // 创建一个 coinbase 交易
	txs := []*Transaction{cbTx, tx}

	newBlock := bc.MineBlock(txs) // 添加到新块。
	UTXOSet.Update(newBlock)
	fmt.Println("Success!")
}
