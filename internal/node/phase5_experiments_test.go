package node

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/wallet"
)

const phase5WaitTimeout = 2 * time.Second

// TestPhase5ForkConvergence records the winning chain and the orphaned
// transaction that returns to the pending pool after reconnection.
func TestPhase5ForkConvergence(t *testing.T) {
	account, privateKey, publicKey := phase5Account(t)
	chainA := chain.NewBlockchain(1)
	chainB := chain.NewBlockchain(1)
	nodeA := NewNode(chainA, nil)
	nodeB := NewNode(chainB, nil)
	serverA := httptest.NewServer(nodeA.Handler())
	defer serverA.Close()
	serverB := httptest.NewServer(nodeB.Handler())
	defer serverB.Close()

	phase5FundAndMine(t, chainA, account, 10)
	phase5FundAndMine(t, chainB, account, 10)

	orphan := block.Transaction{Sender: account, Recipient: account, Amount: 1, Nonce: 1, PublicKey: publicKey}
	orphan.Signature, _ = wallet.Sign(privateKey, orphan.Payload())
	if err := chainA.AddTransaction(orphan); err != nil {
		t.Fatal(err)
	}
	if _, err := chainA.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}

	// B gets a competing block and then one additional block, making it the
	// objectively heavier candidate when the nodes reconnect.
	phase5FundAndMine(t, chainB, account, 1)
	phase5FundAndMine(t, chainB, account, 1)
	beforeA := len(chainA.Blocks)
	beforeB := len(chainB.Blocks)
	if beforeB <= beforeA {
		t.Fatalf("experiment setup failed: expected B to be longer, A=%d B=%d", beforeA, beforeB)
	}

	if err := nodeA.SyncFrom(serverB.URL); err != nil {
		t.Fatalf("A failed to reconnect to B: %v", err)
	}
	if len(chainA.Blocks) != len(chainB.Blocks) {
		t.Fatalf("chains did not converge: A=%d B=%d", len(chainA.Blocks), len(chainB.Blocks))
	}
	for index := range chainA.Blocks {
		if chainA.Blocks[index].Hash != chainB.Blocks[index].Hash {
			t.Fatalf("chains differ at block %d: A=%s B=%s", index, chainA.Blocks[index].Hash, chainB.Blocks[index].Hash)
		}
	}

	orphanFound := false
	for _, pending := range chainA.PendingTransactions {
		if pending.Signature == orphan.Signature {
			orphanFound = true
			break
		}
	}
	if !orphanFound {
		t.Fatalf("orphan transaction was not returned to A's pending pool")
	}

	t.Logf("fork convergence: winner=B, orphaned_block=A[%d], resurrected_tx=%s", beforeA-1, orphan.Signature)
}

// TestPhase5GossipCost measures the outbound HTTP messages caused by one
// transaction on a three-node full mesh and verifies deduplication bounds it.
func TestPhase5GossipCost(t *testing.T) {
	account, privateKey, publicKey := phase5Account(t)
	chains := []*chain.Blockchain{chain.NewBlockchain(1), chain.NewBlockchain(1), chain.NewBlockchain(1)}
	nodes := make([]*Node, len(chains))
	servers := make([]*httptest.Server, len(chains))
	transports := make([]*phase5CountingTransport, len(chains))

	for index := range chains {
		transports[index] = &phase5CountingTransport{base: http.DefaultTransport}
		nodes[index] = NewNode(chains[index], &http.Client{Transport: transports[index], Timeout: phase5WaitTimeout})
		servers[index] = httptest.NewServer(nodes[index].Handler())
		defer servers[index].Close()
		phase5FundAndMine(t, chains[index], account, 10)
	}

	for index, node := range nodes {
		peers := make([]string, 0, len(nodes)-1)
		for peerIndex, server := range servers {
			if peerIndex != index {
				peers = append(peers, server.URL)
			}
		}
		node.SetPeers(peers)
	}

	tx := block.Transaction{Sender: account, Recipient: account, Amount: 1, Nonce: 1, PublicKey: publicKey}
	var err error
	tx.Signature, err = wallet.Sign(privateKey, tx.Payload())
	if err != nil {
		t.Fatal(err)
	}
	if response := phase5PostJSON(t, servers[0].URL+"/tx", tx); response.StatusCode != http.StatusOK {
		t.Fatalf("transaction submission failed with status %d", response.StatusCode)
	}

	phase5Eventually(t, func() bool {
		for _, currentChain := range chains {
			if currentChain.GetPendingTransactionCount() != 1 {
				return false
			}
		}
		return true
	})

	messageCount := int64(0)
	for _, transport := range transports {
		messageCount += atomic.LoadInt64(&transport.requests)
	}
	if messageCount > 6 {
		t.Fatalf("gossip deduplication did not bound traffic: %d messages", messageCount)
	}

	t.Logf("gossip cost: nodes=3, topology=full-mesh, outbound_messages=%d, pending_copies=3", messageCount)
}

type phase5CountingTransport struct {
	requests int64
	base     http.RoundTripper
}

func (transport *phase5CountingTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	atomic.AddInt64(&transport.requests, 1)
	return transport.base.RoundTrip(request)
}

func phase5Account(t *testing.T) (string, string, string) {
	t.Helper()
	privateKey, publicKey, err := wallet.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	address, err := wallet.AddressFromPublicKey(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	return address, privateKey, publicKey
}

func phase5FundAndMine(t *testing.T, blockchain *chain.Blockchain, account string, amount int64) {
	t.Helper()
	transaction := block.Transaction{Sender: "FAUCET", Recipient: account, Amount: amount, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}
	if err := blockchain.AddTransaction(transaction); err != nil {
		t.Fatal(err)
	}
	if _, err := blockchain.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}
}

func phase5PostJSON(t *testing.T, url string, value interface{}) *http.Response {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	response, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	return response
}

func phase5Eventually(t *testing.T, condition func() bool) {
	t.Helper()
	deadline := time.Now().Add(phase5WaitTimeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal(fmt.Sprintf("condition was not met within %s", phase5WaitTimeout))
}
