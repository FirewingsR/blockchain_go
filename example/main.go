package main

import (
	"fmt"
	"math/big"
)

func main() {

	fmt.Println("main")

	bc := NewBlockChain()
	bc.AddBlock("Send 1 BTC To Densey")
	bc.AddBlock("Send 2 more BTC To Densey")

	printChain(bc.Iterator())
}

func main2() {

	printTarget(8)
	printTarget(256 - targetBits)

}

func printTarget(targetBits uint) {
	target := big.NewInt(1)
	target.Lsh(target, targetBits)
	fmt.Printf("%b\n", target)
}
