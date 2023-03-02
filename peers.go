package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/libp2p/go-libp2p/core/peer"
)

const (
	peerDBPath = "peers_%s"
	peerDBFile = "MANIFEST"
)

type Peers struct {
	Database *badger.DB
}

func peerDBexist(path string) bool {
	if _, err := os.Stat(path + "/" + peerDBFile); os.IsNotExist(err) {
		return false
	}
	return true
}

// 创建Peers
func getPeerDB(nodeId string) (*Peers, error) {
	path := fmt.Sprintf(peerDBPath, nodeId)
	// 通过文件名打开DB
	opts := badger.DefaultOptions(path)
	// 忽略log
	// opts.Logger = nil
	db, err := openDB(path, opts)
	if err != nil {
		log.Panic(err)
	}
	peers := Peers{db}
	return &peers, err
}

// 向Peers添加信息
// []byte(peer.ID) => peer.AddrInfo
func (pa *Peers) addPeer(info peer.AddrInfo) {
	err := pa.Database.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get([]byte(info.ID)); err == nil {
			return nil
		}
		infoData, _ := info.MarshalJSON()
		err := txn.Set([]byte(info.ID), infoData)
		return err
	})
	if err != nil {
		log.Panic(err)
	}
}

// 从Peers中删除{pid}信息
func (pa *Peers) deletePeer(pid peer.ID) {
	err := pa.Database.Update(func(txn *badger.Txn) error {
		if err := txn.Delete([]byte(pid)); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
}

func (pa Peers) findAllAddrInfo() []peer.AddrInfo {
	db := pa.Database
	var addrInfos []peer.AddrInfo
	err := db.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		it := txn.NewIterator(opts)
		defer it.Close()
		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			// k := item.Key()
			err := item.Value(func(v []byte) error {
				tmp := peer.AddrInfo{}
				tmp.UnmarshalJSON(v)
				log.Println(tmp)
				addrInfos = append(addrInfos, tmp)
				return nil
			})
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		log.Panic(err)
	}
	return addrInfos
}

func retry(dir string, originalOpts badger.Options) (*badger.DB, error) {
	lockPath := filepath.Join(dir, "LOCK")
	if err := os.Remove(lockPath); err != nil {
		return nil, fmt.Errorf(`removing "LOCK": %s`, err)
	}
	retryOpts := originalOpts
	retryOpts.Truncate = true
	db, err := badger.Open(retryOpts)
	return db, err
}

func openDB(dir string, opts badger.Options) (*badger.DB, error) {
	if db, err := badger.Open(opts); err != nil {
		if strings.Contains(err.Error(), "LOCK") {
			if db, err := retry(dir, opts); err == nil {
				log.Println("database unlocked, value log truncated")
				return db, nil
			}
			log.Println("could not unlock database:", err)
		}
		return nil, err
	} else {
		return db, nil
	}
}
