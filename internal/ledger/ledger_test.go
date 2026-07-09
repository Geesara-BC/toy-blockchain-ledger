package ledger

import (
	"testing"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
)

func TestOverspendingTransactionRejected(t *testing.T) {
	bc := chain.NewBlockchain(1)
	l := NewLedger(bc)

	bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	bc.MinePendingTransactions(5)

	initialBalance := l.GetBalance("Alice")
	if initialBalance != 100 {
		t.Fatalf("Expected Alice to have balance 100, got %f", initialBalance)
	}

	tx := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 150}
	err := l.VerifyTransaction(tx)

	if err == nil {
		t.Errorf("Expected transaction to be rejected due to overspending, but got no error")
	}

	currentBalance := l.GetBalance("Alice")
	if currentBalance != 100 {
		t.Errorf("Expected account balance to remain 100, got %f", currentBalance)
	}
}

func TestMalformedTransactionRejected(t *testing.T) {
	bc := chain.NewBlockchain(1)
	l := NewLedger(bc)

	txZero := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 0}
	txNegative := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: -50}

	if err := l.VerifyTransaction(txZero); err == nil {
		t.Errorf("Expected transaction with amount 0 to be rejected")
	}

	if err := l.VerifyTransaction(txNegative); err == nil {
		t.Errorf("Expected transaction with negative amount to be rejected")
	}
}
