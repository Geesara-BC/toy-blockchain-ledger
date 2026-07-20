package storage

import (
	"path/filepath"
	"testing"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
)

func TestSaveAndLoadPersistsRegisteredMinersAndMiningState(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "blockchain.json")

	bc := chain.NewBlockchain(2)
	if err := bc.RegisterMiner("miner-1"); err != nil {
		t.Fatalf("register miner-1 failed: %v", err)
	}
	if err := bc.RegisterMiner("miner-2"); err != nil {
		t.Fatalf("register miner-2 failed: %v", err)
	}

	if err := bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100}); err != nil {
		t.Fatalf("add faucet transaction failed: %v", err)
	}

	if err := SaveToFile(path, bc); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	if loaded.RewardEngine == nil {
		t.Fatal("expected reward engine to be initialized after load")
	}
	if loaded.MinerRegistry == nil {
		t.Fatal("expected miner registry to be initialized after load")
	}
	if !loaded.IsMinerRegistered("miner-1") {
		t.Fatal("expected miner-1 to be restored")
	}
	if !loaded.IsMinerRegistered("miner-2") {
		t.Fatal("expected miner-2 to be restored")
	}

	if _, err := loaded.MinePendingTransactions(5); err != nil {
		t.Fatalf("mining after load failed: %v", err)
	}
}
