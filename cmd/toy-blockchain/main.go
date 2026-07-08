package main

import (
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

	var dbPath string

	if filepath.Base(cwd) == "toy-blockchain" && filepath.Base(filepath.Dir(cwd)) == "cmd" {
		rootPath := filepath.Dir(filepath.Dir(cwd))
		dbPath = filepath.Join(rootPath, "data", "blockchain.json")
	} else {
		dbPath = filepath.Join(cwd, "data", "blockchain.json")
	}

	c := cli.NewCLI(dbPath)
	c.Run()
}
