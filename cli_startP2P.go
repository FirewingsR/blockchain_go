package main

import (
	"fmt"
	"log"
	"strconv"

	dht "github.com/libp2p/go-libp2p-kad-dht"
	maddr "github.com/multiformats/go-multiaddr"
)

// 使用listen端口启动服务器{nodeId}
// 如果有{minterAddress}，则此服务器将作为minter运行
// 收集transaction后，创建块并获得{minterAddress}的奖励
// 如果有{dest}，则通过{dest}节点连接p2p网络
func (cli CLI) startP2P(nodeId, minterAlias string, secio bool, randseed int64, rendezvous string, bootstrapPeersString string) {
	fmt.Printf("Starting Host localhost:%s\n", nodeId)

	minterAddress := cli.wallets.GetAddress(minterAlias)

	if len(minterAddress) > 0 {
		if ValidateAddress(minterAddress) {
			fmt.Println("Mining is on. Address to receive rewards: ", minterAddress)
		} else {
			log.Panic("Wrong minter address!")
		}
	}

	port, err := strconv.Atoi(nodeId)

	if err != nil {
		log.Panic(err)
	}

	var bootstrapPeers []maddr.Multiaddr

	if bootstrapPeersString == "" {
		bootstrapPeers = dht.DefaultBootstrapPeers
	}

	startHost(port, minterAddress, secio, 0, rendezvous, bootstrapPeers)
}
