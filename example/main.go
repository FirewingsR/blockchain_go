package main

import (
	"fmt"
	"math/big"
)

func main() {
	cli := CLI{}
	cli.Run()
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
