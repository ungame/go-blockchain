package main

import (
	"go-blockchain/cli"
	"os"
)

func main() {
	defer os.Exit(0)
	c := cli.CommandLine{}
	c.Run()
}

