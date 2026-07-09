package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"toy-blockchain/internal/cli"
)

func main() {

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Error getting current working directory: %v\n", err)
		os.Exit(1)
	}

	var defaultDbPath string
	if filepath.Base(cwd) == "toy-blockchain" && filepath.Base(filepath.Dir(cwd)) == "cmd" {
		rootPath := filepath.Dir(filepath.Dir(cwd))
		defaultDbPath = filepath.Join(rootPath, "data", "blockchain.json")
	} else {
		defaultDbPath = filepath.Join(cwd, "data", "blockchain.json")
	}

	dbPath := flag.String("db-path", defaultDbPath, "File path for the blockchain database")
	difficulty := flag.Int("difficulty", 3, "Mining difficulty (number of leading zeroes)")
	maxBlockSize := flag.Int("max-block-size", 5, "Maximum transactions per block")

	flag.Parse()

	c := cli.NewCLI(*dbPath, *difficulty, *maxBlockSize)
	c.Run()
}
