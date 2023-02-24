package main

import (
	"fmt"
	"math/big"
	"os"
)

func main() {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env. var is not set!")
		os.Exit(1)
	}
	cli := CLI{GetInstance(nodeID)}
	cli.Run(nodeID)
}

func main2() {

	a := ValidateAddress("1NSgvKsJSVEHJZECndxKfVKC$WG7hGEuo$")

	fmt.Print("a is ", a)

}

func printTarget(targetBits uint) {
	target := big.NewInt(1)
	target.Lsh(target, targetBits)
	fmt.Printf("%b\n", target)
}
