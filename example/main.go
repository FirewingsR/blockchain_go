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

	a := ValidateAddress("1NSgvKsJSVEHJZECndxKfVKC$WG7hGEuo$")

	fmt.Print("a is ", a)

}

func printTarget(targetBits uint) {
	target := big.NewInt(1)
	target.Lsh(target, targetBits)
	fmt.Printf("%b\n", target)
}
