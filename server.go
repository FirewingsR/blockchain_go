package main

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	mrand "math/rand"
	"net"
	"os"
	"runtime"
	"sync"
	"syscall"

	"github.com/dgraph-io/badger"
	"github.com/libp2p/go-libp2p"
	dht "github.com/libp2p/go-libp2p-kad-dht"
	crypto "github.com/libp2p/go-libp2p/core/crypto"
	host "github.com/libp2p/go-libp2p/core/host"
	network "github.com/libp2p/go-libp2p/core/network"
	peer "github.com/libp2p/go-libp2p/core/peer"
	peerstore "github.com/libp2p/go-libp2p/core/peerstore"
	drouting "github.com/libp2p/go-libp2p/p2p/discovery/routing"
	dutil "github.com/libp2p/go-libp2p/p2p/discovery/util"
	ma "github.com/multiformats/go-multiaddr"
	DEATH "github.com/vrecan/death/v3"
)

const protocol = "tcp"
const nodeVersion = 1
const commandLength = 12

var (
	chain  *BlockChain
	peers  *Peers
	ha     host.Host
	nodeId string // 지금 노드의 nodeId
)

// nodePeerId
var nodeAddress string
var miningAddress string
var knownNodes = []string{"localhost:3000"}
var blocksInTransit = [][]byte{}
var mempool = make(map[string]Transaction)

type addr struct {
	AddrList []string
}

type block struct {
	AddrFrom string
	Block    []byte
}

type getblocks struct {
	AddrFrom string
}

type getdata struct {
	AddrFrom string
	Type     string
	ID       []byte
}

type inv struct {
	AddrFrom string
	Type     string
	Items    [][]byte
}

type tx struct {
	AddFrom     string
	Transaction []byte
}

type version struct {
	Version    int
	BestHeight int
	AddrFrom   string
}

func commandToBytes(command string) []byte {
	var bytes [commandLength]byte

	for i, c := range command {
		bytes[i] = byte(c)
	}

	return bytes[:]
}

func bytesToCommand(bytes []byte) string {
	var command []byte

	for _, b := range bytes {
		if b != 0x0 {
			command = append(command, b)
		}
	}

	return fmt.Sprintf("%s", command)
}

func extractCommand(request []byte) []byte {
	return request[:commandLength]
}

func requestBlocks() {
	for _, node := range knownNodes {
		sendGetBlocks(node)
	}
}

func sendAddr(address string) {
	nodes := addr{knownNodes}
	nodes.AddrList = append(nodes.AddrList, nodeAddress)
	payload := gobEncode(nodes)
	request := append(commandToBytes("addr"), payload...)

	sendData(address, request)
}

func sendBlock(addr string, b *Block) {
	data := block{nodeAddress, b.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("block"), payload...)

	sendData(addr, request)
}

func sendDataNet(addr string, data []byte) {
	conn, err := net.Dial(protocol, addr)
	if err != nil {
		fmt.Printf("%s is not available\n", addr)
		var updatedNodes []string

		for _, node := range knownNodes {
			if node != addr {
				updatedNodes = append(updatedNodes, node)
			}
		}

		knownNodes = updatedNodes

		return
	}
	defer conn.Close()

	_, err = io.Copy(conn, bytes.NewReader(data))
	if err != nil {
		log.Panic(err)
	}
}

func sendInv(address, kind string, items [][]byte) {
	inventory := inv{nodeAddress, kind, items}
	payload := gobEncode(inventory)
	request := append(commandToBytes("inv"), payload...)

	sendData(address, request)
}

func sendGetBlocks(address string) {
	payload := gobEncode(getblocks{nodeAddress})
	request := append(commandToBytes("getblocks"), payload...)

	sendData(address, request)
}

func sendGetData(address, kind string, id []byte) {
	payload := gobEncode(getdata{nodeAddress, kind, id})
	request := append(commandToBytes("getdata"), payload...)

	sendData(address, request)
}

func sendTx(addr string, tnx *Transaction) {
	data := tx{nodeAddress, tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)

	sendData(addr, request)
}

// 发送Transaction（如果传输一次后结束）
func sendTxOnce(addr string, tnx *Transaction) {
	data := tx{nodeAddress, tnx.Serialize()}
	payload := gobEncode(data)
	request := append(commandToBytes("tx"), payload...)

	sendDataOnce(addr, request)
}

func sendVersion(addr string, bc *BlockChain) {
	bestHeight := bc.GetBestHeight()
	payload := gobEncode(version{nodeVersion, bestHeight, nodeAddress})

	request := append(commandToBytes("version"), payload...)

	log.Printf("Send Version {version: %d, height: %d} to %s\n", ver, bestHeight, addr)

	sendData(addr, request)
}

func handleAddr(request []byte) {
	var buff bytes.Buffer
	var payload addr

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	knownNodes = append(knownNodes, payload.AddrList...)
	fmt.Printf("There are %d known nodes now!\n", len(knownNodes))
	requestBlocks()
}

func handleBlock(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload block

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blockData := payload.Block
	block := DeserializeBlock(blockData)

	fmt.Println("Recevied a new block!")
	bc.AddBlock(block)

	fmt.Printf("Added block %x\n", block.Hash)

	if len(blocksInTransit) > 0 {
		blockHash := blocksInTransit[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		blocksInTransit = blocksInTransit[1:]
	} else {
		UTXOSet := UTXOSet{bc}
		UTXOSet.Reindex()
	}
}

func handleInv(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload inv

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	fmt.Printf("Recevied inventory with %d %s\n", len(payload.Items), payload.Type)

	if payload.Type == "block" {
		blocksInTransit = payload.Items

		blockHash := payload.Items[0]
		sendGetData(payload.AddrFrom, "block", blockHash)

		newInTransit := [][]byte{}
		for _, b := range blocksInTransit {
			if bytes.Compare(b, blockHash) != 0 {
				newInTransit = append(newInTransit, b)
			}
		}
		blocksInTransit = newInTransit
	}

	if payload.Type == "tx" {
		txID := payload.Items[0]

		if mempool[hex.EncodeToString(txID)].ID == nil {
			sendGetData(payload.AddrFrom, "tx", txID)
		}
	}
}

func handleGetBlocks(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload getblocks

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	blocks := bc.GetBlockHashes()
	sendInv(payload.AddrFrom, "block", blocks)
}

func handleGetData(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload getdata

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	if payload.Type == "block" {
		block, err := bc.GetBlock([]byte(payload.ID))
		if err != nil {
			return
		}

		sendBlock(payload.AddrFrom, &block)
	}

	if payload.Type == "tx" {
		txID := hex.EncodeToString(payload.ID)
		tx := mempool[txID]

		sendTx(payload.AddrFrom, &tx)
		// delete(mempool, txID)
	}
}

func handleTxNet(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.ID)] = tx

	if nodeAddress == knownNodes[0] {
		for _, node := range knownNodes {
			if node != nodeAddress && node != payload.AddFrom {
				sendInv(node, "tx", [][]byte{tx.ID})
			}
		}
	} else {
		if len(mempool) >= 2 && len(miningAddress) > 0 {
		MineTransactions:
			var txs []*Transaction

			for id := range mempool {
				fmt.Printf("tx: ^s\n", mempool[id].ID)
				tx := mempool[id]
				if bc.VerifyTransaction(&tx) {
					txs = append(txs, &tx)
				}
			}

			if len(txs) == 0 {
				fmt.Println("All transactions are invalid! Waiting for new ones...")
				return
			}

			cbTx := NewCoinbaseTX(miningAddress, "")
			txs = append(txs, cbTx)

			newBlock := bc.MineBlock(txs)
			UTXOSet := UTXOSet{bc}
			UTXOSet.Reindex()

			fmt.Println("New block is mined!")

			for _, tx := range txs {
				txID := hex.EncodeToString(tx.ID)
				delete(mempool, txID)
			}

			for _, node := range knownNodes {
				if node != nodeAddress {
					sendInv(node, "block", [][]byte{newBlock.Hash})
				}
			}

			if len(mempool) > 0 {
				goto MineTransactions
			}
		}
	}
}

func handleTx(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload tx

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	txData := payload.Transaction
	tx := DeserializeTransaction(txData)
	mempool[hex.EncodeToString(tx.ID)] = tx

	log.Printf("%s received Tx, now %d txs in memoryPool\n", nodeAddress, len(mempool))

	for _, node := range knownNodes {
		if node != nodeAddress && node != payload.AddFrom {
			sendInv(node, "tx", [][]byte{tx.ID})
		}
	}

	if len(mempool) >= 2 && len(miningAddress) > 0 {
	MineTransactions:
		var txs []*Transaction

		for id := range mempool {
			fmt.Printf("tx: ^s\n", mempool[id].ID)
			tx := mempool[id]
			if bc.VerifyTransaction(&tx) {
				txs = append(txs, &tx)
			}
		}

		if len(txs) == 0 {
			fmt.Println("All transactions are invalid! Waiting for new ones...")
			return
		}

		cbTx := NewCoinbaseTX(miningAddress, "")
		txs = append(txs, cbTx)

		newBlock := bc.MineBlock(txs)
		UTXOSet := UTXOSet{bc}
		UTXOSet.Reindex()

		fmt.Println("New block is mined!")

		for _, tx := range txs {
			txID := hex.EncodeToString(tx.ID)
			delete(mempool, txID)
		}

		for _, node := range knownNodes {
			if node != nodeAddress {
				sendInv(node, "block", [][]byte{newBlock.Hash})
			}
		}

		if len(mempool) > 0 {
			goto MineTransactions
		}
	}
}

func handleVersion(request []byte, bc *BlockChain) {
	var buff bytes.Buffer
	var payload version

	buff.Write(request[commandLength:])
	dec := gob.NewDecoder(&buff)
	err := dec.Decode(&payload)
	if err != nil {
		log.Panic(err)
	}

	myBestHeight := bc.GetBestHeight()
	foreignerBestHeight := payload.BestHeight

	if myBestHeight < foreignerBestHeight {
		sendGetBlocks(payload.AddrFrom)
	} else if myBestHeight > foreignerBestHeight {
		sendVersion(payload.AddrFrom, bc)
	}

	// sendAddr(payload.AddrFrom)
	if !nodeIsKnown(payload.AddrFrom) {
		knownNodes = append(knownNodes, payload.AddrFrom)
	}
}

func handleConnection(conn net.Conn, bc *BlockChain) {
	request, err := ioutil.ReadAll(conn)
	if err != nil {
		log.Panic(err)
	}
	command := bytesToCommand(request[:commandLength])
	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		fmt.Println("Unknown command!")
	}

	defer conn.Close()
}

// StartServer starts a node
func StartServer(nodeID, minerAddress string) {
	nodeAddress = fmt.Sprintf("localhost:%s", nodeID)
	miningAddress = minerAddress
	ln, err := net.Listen(protocol, nodeAddress)
	if err != nil {
		log.Panic(err)
	}
	defer ln.Close()

	bc := NewBlockChain(nodeID)
	defer bc.db.Close()
	go CloseDB(bc)

	if nodeAddress != knownNodes[0] {
		sendVersion(knownNodes[0], bc)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Panic(err)
		}
		go handleConnection(conn, bc)
	}
}

func gobEncode(data interface{}) []byte {
	var buff bytes.Buffer

	enc := gob.NewEncoder(&buff)
	err := enc.Encode(data)
	if err != nil {
		log.Panic(err)
	}

	return buff.Bytes()
}

func nodeIsKnown(addr string) bool {
	for _, node := range knownNodes {
		if node == addr {
			return true
		}
	}

	return false
}

// 安全的 DB close
func CloseDB(bc *BlockChain) {
	// SIGINT, SIGTERM : unix, linux / Interrupt : window
	d := DEATH.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		bc.db.Close()
	})
}

func CloseDB2(db *badger.DB) {
	// SIGINT, SIGTERM : unix, linux / Interrupt : window
	d := DEATH.NewDeath(syscall.SIGINT, syscall.SIGTERM, os.Interrupt)

	d.WaitForDeathWithFunc(func() {
		defer os.Exit(1)
		defer runtime.Goexit()
		db.Close()
	})
}

func makeBasicHost(lintenPost int, secio bool, randseed int64) (host.Host, error) {

	// 如果randseed为0，就不是完美的随机值。将使用可预测值生成相同的priv。

	var r io.Reader

	if randseed == 0 {
		r = rand.Reader
	} else {
		r = mrand.New(mrand.NewSource(randseed))
	}

	//创建此主机的key pair
	priv, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		return nil, err
	}

	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", lintenPost)),
		libp2p.Identity(priv),
		libp2p.DisableRelay(),
	}

	return libp2p.New(opts...)

}

// !发送{data}（cmd+payload）
// !p2p使用peer ID进行通信
func sendData(destPeerID string, data []byte) {
	peerID, err := peer.Decode(destPeerID)

	if err != nil {
		log.Panic(err)
	}

	// 创建{ha}=>{peerID}的Stream
	// 此Stream将由{peerID}主机的steamHandler处理

	s, err := ha.NewStream(context.Background(), peerID, "/p2p/1.0.0")

	if err != nil {

		log.Printf("%s is \033[1;33mnot reachable\033[0m\n", peerID)

		peers.deletePeer(peerID)

		log.Printf("%s deleted\n", peerID)

		// TODO：从KnownNodes中删除无法通信的{peer}

		var updatedPeers []string
		for _, node := range knownNodes {
			if node != destPeerID {
				updatedPeers = append(updatedPeers, node)
			}
		}

		knownNodes = updatedPeers
	}

	defer s.Close()

	_, err = s.Write(data)
	if err != nil {
		log.Panic(err)
	}
}

// 向{targetPeer}发送{data}
// 创建并发送一次性host
func sendDataOnce(targetPeer string, data []byte) {
	host, err := libp2p.New()
	if err != nil {
		log.Panic(err)
	}
	defer host.Close()

	destPeerID := addAddrToPeerstore(host, targetPeer)
	sendData(peer.Encode(destPeerID), data)
}

func getHostAddress(_ha host.Host) string {
	hostAddr, _ := ma.NewMultiaddr(fmt.Sprintf("/ipfs/%s", _ha.ID().Pretty()))

	// 现在我们可以建立一个完整的多地址来访问这个主机
	// 通过封装两个地址：
	addr := _ha.Addrs()[0]
	return addr.Encapsulate(hostAddr).String()
}

func handleStream(s network.Stream) {
	defer s.Close()

	// 为Non blocking read/write创建缓冲流
	rw := bufio.NewReadWriter(bufio.NewReader(s), bufio.NewWriter(s))

	// connection处理异步处理为go例程
	go HandleP2PConnection(rw, chain)
}

func HandleP2PConnection(rw *bufio.ReadWriter, bc *BlockChain) {
	request, err := ioutil.ReadAll(rw)
	if err != nil {
		log.Panic(err)
	}

	command := bytesToCommand(request[:commandLength])

	fmt.Printf("Received %s command\n", command)

	switch command {
	case "addr":
		handleAddr(request)
	case "block":
		handleBlock(request, bc)
	case "inv":
		handleInv(request, bc)
	case "getblocks":
		handleGetBlocks(request, bc)
	case "getdata":
		handleGetData(request, bc)
	case "tx":
		handleTx(request, bc)
	case "version":
		handleVersion(request, bc)
	default:
		log.Println("\033[1;31mUnknown command!\033[0m", string(request))
	}
}

// 启动Host。
func startHost(listenPort int, minter string, secio bool, randseed int64, rendezvous string, bootstrapPeers []ma.Multiaddr) {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 将minter的地址存储在全局变量中。
	miningAddress = minter

	// {listenPort}将被用作nodeId
	nodeId = fmt.Sprintf("%d", listenPort)
	// 将chain存储在全局变量中
	chain = NewBlockChain(nodeId)
	go CloseDB(chain) // 等待硬件中断并安全关闭DB的函数
	defer chain.db.Close()

	// 创建p2p host
	host, err := makeBasicHost(listenPort, secio, randseed)

	if err != nil {
		log.Panic(err)
	}

	// 将{host}存储在全局变量{ha}中
	ha = host

	// {nodePeerId}：此节点的peer ID。
	// 通信使用Peer ID
	nodeAddress = peer.Encode(host.ID())

	if len(knownNodes) == 0 {
		//KnownNodes[0]是自己
		knownNodes = append(knownNodes, nodeAddress)
	}

	fullAddr := getHostAddress(ha)
	log.Printf("I am %s\n", fullAddr)

	ha.SetStreamHandler("/p2p/1.0.0", handleStream)

	// 加载保存的对等方
	peers, err = getPeerDB(nodeId)
	if err != nil {
		log.Println(err)
	}
	go CloseDB2(peers.Database)
	defer peers.Database.Close()

	// 首先尝试连接到存储的对等
	connected := connectToKnownPeer(host, peers)
	// 如果未连接到存储的对等
	if !connected {
		peerDiscovery(ctx, host, peers, rendezvous, bootstrapPeers)
	}

	// Wait forever
	select {}

}

// 联系存储在DB中的Peer
func connectToKnownPeer(host host.Host, peers *Peers) bool {
	// 输出保存的peer
	peerAddrInfos := peers.findAllAddrInfo()
	log.Println("\033[1;36mIn peers DB:\033[0m")
	for _, peerAddrInfo := range peerAddrInfos {
		fmt.Printf("%s\n", peerAddrInfo)
	}
	// 首先连接到存储的对等方
	for _, peerinfo := range peerAddrInfos {
		// 创建{host}=>{peer}的Stream。
		// 此Stream将由{peer}主机的steamHandler处理。
		s, err := host.NewStream(context.Background(), peerinfo.ID, "/p2p/1.0.0")
		if err != nil {
			log.Printf("%s is \033[1;33mnot reachable\033[0m\n", peerinfo.ID)
			// 如果无法连接，请从peer DB中删除它。
			peers.deletePeer(peerinfo.ID)
			log.Printf("%s => %s deleted\n", peerinfo.ID, peerinfo.Addrs)
			// TODO：从KnownPeers中删除无法通信的{peer}。
			var updatedPeers []string
			// 从KnownPeers中删除无法通信的{addr}。
			for _, node := range knownNodes {
				if node != peer.Encode(peerinfo.ID) {
					updatedPeers = append(updatedPeers, node)
				}
			}
			knownNodes = updatedPeers
		} else {
			// 连接后发送Version消息
			sendVersion(peer.Encode(peerinfo.ID), chain)
			s.Close()
			return true
		}
	}
	return false
}

// 从rendezvous point获取并连接其他对等方的信息。
func peerDiscovery(ctx context.Context, host host.Host, peers *Peers, rendezvous string, bootstrapPeers []ma.Multiaddr) {
	kademliaDHT, err := dht.New(ctx, host)
	if err != nil {
		panic(err)
	}
	log.Println("Bootstrapping the DHT")
	if err = kademliaDHT.Bootstrap(ctx); err != nil {
		panic(err)
	}
	// Bootstrap节点告知网络中其他节点的信息。
	// 当然，我们的信息也会传递给连接的其他节点。
	var wg sync.WaitGroup
	for _, peerAddr := range bootstrapPeers {
		peerinfo, _ := peer.AddrInfoFromP2pAddr(peerAddr)
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := host.Connect(ctx, *peerinfo); err != nil {
				log.Fatalln(err)
			} else {
				log.Println("Connection established with bootstrap node:", *peerinfo)
			}
		}()
	}
	wg.Wait()
	// 在rendezvous point上写下我们的信息
	log.Println("Announcing ourselves...")
	routingDiscovery := drouting.NewRoutingDiscovery(kademliaDHT)
	dutil.Advertise(ctx, routingDiscovery, rendezvous)
	log.Println("Successfully announced!")
	log.Println("Searching for other peers...")
	log.Printf("Now run \"go run main.go startp2p -rendezvous %s\" on a different terminal\n", rendezvous)
	// 查找peer[]返回peer.AddrInfo。
	peerChan, err := routingDiscovery.FindPeers(ctx, rendezvous)
	if err != nil {
		panic(err)
	}
	for p := range peerChan {
		if p.ID == host.ID() {
			continue
		}
		// 如果您有有效的Addrs
		if len(p.Addrs) > 0 {
			log.Println("\033[1;36mConnecting to:\033[0m", p)
			// 将此信息存储在Peer DB中
			peers.addPeer(p)
			// 打开Stream
			s, err := ha.NewStream(context.Background(), p.ID, "/p2p/1.0.0")
			if err != nil {
				log.Printf("%s is \033[1;33mnot reachable\033[0m\n", p.ID)
				// 如果Stream创建出现错误，请从PeerDB中删除Peer。
				peers.deletePeer(p.ID)
				log.Printf("%s => %s \033[1;33mdeleted\033[0m\n", p.ID, p.Addrs)
			} else {
				s.Close()
				// 向{p}发送{Chain}的版本
				sendVersion(peer.Encode(p.ID), chain)
			}
		} else {
			// Peer无效可能存储在DB中，请删除它。
			peers.deletePeer(p.ID)
			log.Println("\033[1;31mINVAILD ADDR\033[0m", p)
		}
	}
}

// 启动listening server（中央服务器）
func runListener(ctx context.Context, ha host.Host, listenPort int, secio bool) {

	fullAddr := getHostAddress(ha)
	log.Printf("I am %s\n", fullAddr)

	// 设置StreamHandler
	// {handleStream}是收到stream时调用的处理程序函数
	// p2p/1.0.0是user-defined protocal

	ha.SetStreamHandler("/p2p/1.0.0", handleStream)

	log.Printf("Now run \"go run main.go startp2p -dest %s\" on a different terminal\n", fullAddr)

}

// 设置StreamHandler，并向{targetPeer}发送版本信息。
func runSender(ctx context.Context, ha host.Host, targetPeer string) {

	fullAddr := getHostAddress(ha)
	log.Printf("I am %s\n", fullAddr)

	ha.SetStreamHandler("/p2p/1.0.0", handleStream)

	// 将targetPeer保存在ha的Peerstore中，并接收destination的peerId。

	destPeerID := addAddrToPeerstore(ha, targetPeer)

	// 将{chain}的版本发送给{destPeerID}
	sendVersion(peer.Encode(destPeerID), chain)

}

// 接收peer的{addr}，解析为multiaddress，然后保存到host的peerstore中
// 如果您通过该信息知道peer ID，就可以知道如何进行通信
// 返回peer的ID
func addAddrToPeerstore(ha host.Host, addr string) peer.ID {
	// 解析到multiaddress后
	ipfsaddr, err := ma.NewMultiaddr(addr)
	if err != nil {
		log.Fatalln(err)
	}

	// 从multiaddress获取Address和PeerID信息
	info, err := peer.AddrInfoFromP2pAddr(ipfsaddr)
	if err != nil {
		log.Fatalln(err)
	}

	// 让LibP2P参考
	// 将Peer ID和address存储在peerstore中
	ha.Peerstore().AddAddrs(info.ID, info.Addrs, peerstore.PermanentAddrTTL)
	return info.ID

}
