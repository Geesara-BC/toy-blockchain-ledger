package node

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
	"time"

	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/wallet"
)

// Test that a node will adopt a longer chain from a peer and resurrect orphaned transactions
func TestSyncAdoptsLongerChainAndResurrectsOrphans(t *testing.T) {
	// node B (will become longer)
	bcB := chain.NewBlockchain(1)
	nB := NewNode(bcB, nil)
	srvB := httptest.NewServer(nB.Handler())
	defer srvB.Close()

	// node A (will start with a different block)
	bcA := chain.NewBlockchain(1)
	nA := NewNode(bcA, nil)
	srvA := httptest.NewServer(nA.Handler())
	defer srvA.Close()

	// make sure both nodes have a funded account
	priv, pub, err := wallet.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	addr, err := wallet.AddressFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}

	// fund both A and B so txs validate
	faucet := block.Transaction{Sender: "FAUCET", Recipient: addr, Amount: 10, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}
	if err := bcA.AddTransaction(faucet); err != nil {
		t.Fatal(err)
	}
	if _, err := bcA.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}

	if err := bcB.AddTransaction(faucet); err != nil {
		t.Fatal(err)
	}
	if _, err := bcB.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}

	// On A: create a transaction and mine it into a block (this tx will become orphaned)
	tx := block.Transaction{Sender: addr, Recipient: addr, Amount: 1, Nonce: 1, PublicKey: pub}
	sig, _ := wallet.Sign(priv, tx.Payload())
	tx.Signature = sig
	if err := bcA.AddTransaction(tx); err != nil {
		t.Fatal(err)
	}
	if _, err := bcA.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}

	// At this point A has one extra block containing tx, B does not.
	// Now make B longer by mining two empty (coinbase) blocks
	if err := bcB.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: addr, Amount: 1, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}); err != nil {
		t.Fatal(err)
	}
	if _, err := bcB.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}
	if err := bcB.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: addr, Amount: 1, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}); err != nil {
		t.Fatal(err)
	}
	if _, err := bcB.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}

	// Ensure B is longer
	if len(bcB.Blocks) <= len(bcA.Blocks) {
		t.Fatalf("expected B to be longer than A")
	}

	// Ask A to sync from B
	if err := nA.SyncFrom(srvB.URL); err != nil {
		t.Fatalf("sync failed: %v", err)
	}

	// After sync, A should have same length as B
	if len(bcA.Blocks) != len(bcB.Blocks) {
		t.Fatalf("expected A to adopt B's chain length %d, got %d", len(bcB.Blocks), len(bcA.Blocks))
	}

	// The transaction originally committed on A's old chain should be resurrected into pending pool on A (if still valid)
	// Wait small moment for any background processing
	time.Sleep(50 * time.Millisecond)

	found := false
	for _, p := range bcA.PendingTransactions {
		if p.Signature == tx.Signature && p.Nonce == tx.Nonce && p.Sender == tx.Sender {
			found = true
			break
		}
	}

	if !found {
		// Fail, but print diagnostics
		bB, _ := json.MarshalIndent(bcB, "", "  ")
		bA, _ := json.MarshalIndent(bcA, "", "  ")
		t.Fatalf("orphaned tx not resurrected into A pending pool. A height=%d B height=%d\nA: %s\nB: %s", len(bcA.Blocks), len(bcB.Blocks), string(bA), string(bB))
	}
}
