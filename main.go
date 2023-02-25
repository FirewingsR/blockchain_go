package main

import (
	"fmt"
	"os"
)

func main() {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		fmt.Printf("NODE_ID env. var is not set!")
		os.Exit(1)
	}
	cli := CLI{WalletsInstance(nodeID)}
	cli.Run(nodeID)
}
