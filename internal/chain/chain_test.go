package chain

import (
	"strings"
	"testing"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/wallet"
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
	bc := NewBlockchain(3)

	err1 := bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 50})
	if err1 != nil {
		t.Fatalf("Faucet transaction failed: %v", err1)
	}
	bc.MinePendingTransactions(5)

	tx := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 10}
	err2 := bc.AddTransaction(tx)

	expectedErr := "UNAUTHORIZED: Transaction must be signed with a valid public key and signature"

	if err2 == nil {
		t.Errorf("Test failed: An unsigned transaction was incorrectly accepted!")
	} else if err2.Error() != expectedErr {
		t.Errorf("Test failed: Expected error '%s', but got: %v", expectedErr, err2)
	}

	isValid, _ := bc.IsValid()
	if !isValid {
		t.Errorf("Test failed: Chain should be valid even after a rejected transaction attempt")
	}
}
func TestTamperedChainDetection(t *testing.T) {
	bc := NewBlockchain(3)

	bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	bc.MinePendingTransactions(5)

	priv, pub, _ := wallet.GenerateKeyPair()
	tx := block.Transaction{Sender: pub, Recipient: "Bob", Amount: 20, PublicKey: pub}
	sig, _ := wallet.Sign(priv, tx.Payload())
	tx.Signature = sig

	bc.AddTransaction(tx)
	bc.MinePendingTransactions(5)

	bc.Blocks[1].Transactions[0].Amount = 500

	isValid, faultyIndex := bc.IsValid()

	if isValid {
		t.Errorf("Security Test Failed: Tampered chain was incorrectly marked as valid!")
	}

	if faultyIndex != 1 {
		t.Errorf("Security Test Failed: Expected failure at block index 1, but got %d", faultyIndex)
	}
}
