package main

import "fmt"

func (cli *CLI) createWallet() {
	wallets, _ := NewWallets()
	address := wallets.CreateWallet()
	fmt.Printf("address is %s\n", address)
	wallets.SaveToFile()

	fmt.Printf("Your new address: %s\n", address)
}
