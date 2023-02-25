package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/gob"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"sync"
)

const walletFile = "wallet_%s.dat"

// Wallets stores a collection of wallets
type Wallets struct {
	Wallets map[string]*Wallet
	Alias   map[string]string
}

var instance *Wallets
var once sync.Once

func WalletsInstance(nodeID string) *Wallets {
	once.Do(func() {
		wallets, err := newWallets(nodeID)

		if err != nil {
			log.Panic(err)
		}

		instance = wallets
	})
	return instance
}

// NewWallets creates Wallets and fills it from a file if it exists
func newWallets(nodeID string) (*Wallets, error) {
	wallets := Wallets{}
	wallets.Wallets = make(map[string]*Wallet)
	wallets.Alias = make(map[string]string)

	err := wallets.LoadFromFile(nodeID)

	return &wallets, err
}

// CreateWallet adds a Wallet to Wallets
// func (ws *Wallets) CreateWallet() string {
// 	wallet := NewWallet()
// 	address := fmt.Sprintf("%s", wallet.GetAddress())

// 	ws.Wallets[address] = wallet

// 	return address
// }

// GetAddresses returns an array of addresses stored in the wallet file
// func (ws *Wallets) GetAddresses() []string {
// 	var addresses []string

// 	for address := range ws.Wallets {
// 		addresses = append(addresses, address)
// 	}

// 	return addresses
// }

// GetWallet returns a Wallet by its address
func (ws Wallets) GetWallet(address string) Wallet {
	return *ws.Wallets[address]
}

// LoadFromFile loads wallets from the file
func (ws *Wallets) LoadFromFile(nodeID string) error {
	walletFile := fmt.Sprintf(walletFile, nodeID)
	if _, err := os.Stat(walletFile); os.IsNotExist(err) {
		ws.SaveToFile(nodeID)
		return err
	}

	fileContent, err := ioutil.ReadFile(walletFile)
	if err != nil {
		log.Panic(err)
	}

	var wallets Wallets
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(fileContent))
	err = decoder.Decode(&wallets)
	if err != nil {
		log.Panic(err)
	}

	ws.Wallets = wallets.Wallets
	ws.Alias = wallets.Alias

	return nil
}

// SaveToFile saves wallets to a file
func (ws Wallets) SaveToFile(nodeID string) {
	var content bytes.Buffer
	walletFile := fmt.Sprintf(walletFile, nodeID)

	gob.Register(elliptic.P256())

	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(ws)
	if err != nil {
		log.Panic(err)
	}

	err = ioutil.WriteFile(walletFile, content.Bytes(), 0644)
	if err != nil {
		log.Panic(err)
	}
}

func (ws *Wallets) AddWallet(alias string) string {

	wallet := NewWallet()

	address := fmt.Sprintf("%s", wallet.GetAddress())

	ws.Wallets[address] = wallet

	if alias != "" {
		_, exists := ws.Alias[alias]
		if exists {
			log.Panic("alias already exists")
		}
		ws.Alias[alias] = address
	} else {
		ws.Alias[address] = address
	}

	return address

}

func (ws *Wallets) GetAddress(alias string) string {
	address, exists := ws.Alias[alias]

	if exists {
		return address
	}

	return alias
}

func (ws *Wallets) GetAllAliases() []string {
	var aliases []string

	for alias := range ws.Alias {
		aliases = append(aliases, alias)
	}

	return aliases
}
