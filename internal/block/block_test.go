package block

import (
	"testing"
	"time"
)

func TestDeterministicHashing(t *testing.T) {

	b := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 50.0}},
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        100,
	}

	hash1 := b.CalculateHash()
	hash2 := b.CalculateHash()

	if hash1 != hash2 {
		t.Errorf("Hashes are not identical!\nHash 1: %s\nHash 2: %s", hash1, hash2)
	}
}

func TestNewGenesisBlock(t *testing.T) {
	genesisBlock := NewGenesisBlock()

	if genesisBlock.Index != 0 {
		t.Errorf("Genesis block index must be 0, got %d", genesisBlock.Index)
	}

	expectedPrevHash := "0000000000000000000000000000000000000000000000000000000000000000"
	if genesisBlock.PreviousHash != expectedPrevHash {
		t.Errorf("Expected previous hash %s, got %s", expectedPrevHash, genesisBlock.PreviousHash)
	}

	if genesisBlock.Hash == "" || genesisBlock.Hash != genesisBlock.CalculateHash() {
		t.Errorf("Genesis block hash is invalid or not correctly calculated")
	}
}
