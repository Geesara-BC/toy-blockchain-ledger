package ledger

import (
	"testing"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/wallet"
)

func TestOverspendingTransactionRejected(t *testing.T) {
	bc := chain.NewBlockchain(1)
	l := NewLedger(bc)

	bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	bc.MinePendingTransactions(5)

	initialBalance := l.GetBalance("Alice")
	if initialBalance != 100 {
		t.Fatalf("Expected Alice to have balance 100, got %d", initialBalance)
	}

	privKey, pubKey, _ := wallet.GenerateKeyPair()
	addr, err := wallet.AddressFromPublicKey(pubKey)
	if err != nil {
		t.Fatalf("failed to derive address: %v", err)
	}

	tx := block.Transaction{
		Sender:    addr,
		Recipient: "Bob",
		Amount:    150,
		Nonce:     1,
		PublicKey: pubKey,
	}

	sig, _ := wallet.Sign(privKey, tx.Payload())
	tx.Signature = sig

	err = l.VerifyTransaction(tx)

	if err == nil {
		t.Errorf("Expected transaction to be rejected due to insufficient balance, but got no error")
	} else if err.Error() != "insufficient balance: sender does not have enough funds" {
		t.Errorf("Expected 'insufficient balance' error, but got: %v", err)
	}

	currentBalance := l.GetBalance("Alice")
	if currentBalance != 100 {
		t.Errorf("Expected account balance to remain 100, got %d", currentBalance)
	}
}

func TestMalformedTransactionRejected(t *testing.T) {
	bc := chain.NewBlockchain(1)
	l := NewLedger(bc)

	privKey, pubKey, _ := wallet.GenerateKeyPair()
	addr, err := wallet.AddressFromPublicKey(pubKey)
	if err != nil {
		t.Fatalf("failed to derive address: %v", err)
	}

	txZero := block.Transaction{
		Sender:    addr,
		Recipient: "Bob",
		Amount:    0,
		Nonce:     1,
		PublicKey: pubKey,
	}
	sigZero, _ := wallet.Sign(privKey, txZero.Payload())
	txZero.Signature = sigZero

	txNegative := block.Transaction{
		Sender:    addr,
		Recipient: "Bob",
		Amount:    -50,
		Nonce:     1,
		PublicKey: pubKey,
	}
	sigNeg, _ := wallet.Sign(privKey, txNegative.Payload())
	txNegative.Signature = sigNeg

	expectedErr := "transaction amount must be positive"

	errZero := l.VerifyTransaction(txZero)
	if errZero == nil || errZero.Error() != expectedErr {
		t.Errorf("Expected error '%s' for amount 0, but got: %v", expectedErr, errZero)
	}

	errNeg := l.VerifyTransaction(txNegative)
	if errNeg == nil || errNeg.Error() != expectedErr {
		t.Errorf("Expected error '%s' for negative amount, but got: %v", expectedErr, errNeg)
	}
}
