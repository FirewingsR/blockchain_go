package main

import (
	"fmt"
)

func (cli *CLI) listAddresses(nodeID string) {
	wallets := cli.wallets
	// if err != nil {
	// 	log.Panic(err)
	// }
	addresses := wallets.GetAllAliases()

	for _, alias := range addresses {
		fmt.Println(wallets.GetAddress(alias), " alias: ", alias)
	}
}
