package main

import "bytes"

// TXInput represents a transaction input
// 指向要用作Input的UTXO。
type TxInput struct {
	ID        []byte // 生成UTXO的事务的ID
	Out       int    // 事务中是第几次UTXO
	Signature []byte // 要使用UTXO的人的签名
	PubKey    []byte // UTXO中的PublicKeyHash值
}

// UsesKey checks whether the address initiated the transaction
func (in *TXInput) UsesKey(pubKeyHash []byte) bool {
	lockingHash := HashPubKey(in.PubKey)

	return bytes.Compare(lockingHash, pubKeyHash) == 0
}
