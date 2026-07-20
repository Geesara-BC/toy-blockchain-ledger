package block

import (
	"crypto/sha256"
	"fmt"
	"testing"
	"time"
)

func TestDeterministicHashing(t *testing.T) {

	b := &Block{
		Index:        1,
		Timestamp:    time.Now().Unix(),
		Transactions: []Transaction{{Sender: "Alice", Recipient: "Bob", Amount: 50}},
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        100,
	}

	hash1 := b.CalculateHash()
	hash2 := b.CalculateHash()

	if hash1 != hash2 {
		t.Errorf("Hashes are not identical!\nHash 1: %s\nHash 2: %s", hash1, hash2)
	}
}

func TestMerkleRootSingleTransaction(t *testing.T) {
	tx := Transaction{Sender: "Alice", Recipient: "Bob", Amount: 50}
	b := &Block{Transactions: []Transaction{tx}}

	expected := fmt.Sprintf("%x", sha256.Sum256([]byte(tx.Payload())))
	if got := b.MerkleRoot(); got != expected {
		t.Errorf("expected merkle root %s for a single transaction, got %s", expected, got)
	}
}

func TestBlockMerkleRootIntegration(t *testing.T) {
	b := &Block{
		Transactions: []Transaction{
			{Sender: "Alice", Recipient: "Bob", Amount: 10},
			{Sender: "Carol", Recipient: "Dan", Amount: 20},
		},
	}

	leaf1 := sha256.Sum256([]byte(b.Transactions[0].Payload()))
	leaf2 := sha256.Sum256([]byte(b.Transactions[1].Payload()))
	parent := sha256.Sum256(append(append([]byte(nil), leaf1[:]...), leaf2[:]...))
	expected := fmt.Sprintf("%x", parent)

	if got := b.MerkleRoot(); got != expected {
		t.Errorf("expected block merkle root %s, got %s", expected, got)
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
