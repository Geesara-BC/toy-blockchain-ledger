package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
	"toy-blockchain/internal/chain"
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
	miningWorkers := flag.Int("mining-workers", 4, "Number of goroutines used for concurrent mining")
	difficultyRetargetInterval := flag.Int("difficulty-retarget-interval", chain.DefaultDifficultyRetargetInterval, "How often to retarget difficulty (in blocks)")
	expectedBlockTimeSeconds := flag.Int("expected-block-time", chain.DefaultExpectedBlockTimeSeconds, "Target block time in seconds for difficulty retargeting")

	flag.Parse()

	c := cli.NewCLI(*dbPath, *difficulty, *maxBlockSize, *miningWorkers, 10*time.Second, *difficultyRetargetInterval, *expectedBlockTimeSeconds)
	c.Run()
}
