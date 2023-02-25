package main

import "fmt"

func (cli *CLI) createWallet(nodeID string, alias string) {
	wallets := cli.wallets
	address := wallets.AddWallet(alias)
	wallets.SaveToFile(nodeID)

	fmt.Printf("Your new address: %s\n", address)
}
