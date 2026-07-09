package chain

import (
	"strings"
	"testing"
	"toy-blockchain/internal/block"
)

func TestGenesisBlockInChain(t *testing.T) {
	difficulty := 2
	bc := NewBlockchain(difficulty)

	if len(bc.Blocks) != 1 {
		t.Fatalf("Expected chain to have exactly 1 block, got %d", len(bc.Blocks))
	}

	genesisBlock := bc.Blocks[0]
	if genesisBlock.Index != 0 {
		t.Errorf("Expected genesis block index to be 0, got %d", genesisBlock.Index)
	}

	expectedPrevHash := "0000000000000000000000000000000000000000000000000000000000000000"
	if genesisBlock.PreviousHash != expectedPrevHash {
		t.Errorf("Expected previous hash %s, got %s", expectedPrevHash, genesisBlock.PreviousHash)
	}
}

func TestMinedBlockDifficulty(t *testing.T) {
	difficulty := 2
	bc := NewBlockchain(difficulty)
	bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})

	newBlock, err := bc.MinePendingTransactions(5)
	if err != nil {
		t.Fatalf("Mining failed: %v", err)
	}

	targetPrefix := strings.Repeat("0", difficulty)
	if !strings.HasPrefix(newBlock.Hash, targetPrefix) {
		t.Errorf("Block hash does not meet difficulty. Expected prefix '%s', got hash %s", targetPrefix, newBlock.Hash)
	}

	recalculatedHash := newBlock.CalculateHash()
	if recalculatedHash != newBlock.Hash {
		t.Errorf("Nonce does not reproduce the hash. Expected %s, got %s", newBlock.Hash, recalculatedHash)
	}
}

func TestHonestChainValidation(t *testing.T) {
	bc := NewBlockchain(1)
	bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 50})
	bc.MinePendingTransactions(5)
	bc.AddTransaction(block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 10})
	bc.MinePendingTransactions(5)

	isValid, _ := bc.IsValid()

	if !isValid {
		t.Errorf("Expected an honest chain to validate successfully, but it failed")
	}
}

func TestTamperedChainDetection(t *testing.T) {
	bc := NewBlockchain(1)
	bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	bc.MinePendingTransactions(5)
	bc.AddTransaction(block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 20})
	bc.MinePendingTransactions(5)

	bc.Blocks[1].Transactions[0].Amount = 500

	isValid, faultyIndex := bc.IsValid()

	if isValid {
		t.Errorf("Expected tampered chain to fail validation, but it passed")
	}

	if faultyIndex != 1 {
		t.Errorf("Expected validation to identify block index 1 as faulty, got %d", faultyIndex)
	}
}
