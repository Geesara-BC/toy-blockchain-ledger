package node

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/storage"
	"toy-blockchain/internal/wallet"
)

func TestVerifyReadsPersistedBlockchain(t *testing.T) {
	dbPath := t.TempDir() + string(os.PathSeparator) + "blockchain.json"
	bc := chain.NewBlockchain(1)
	if err := storage.SaveToFile(dbPath, bc); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), bc.Blocks[0].Hash, "tampered", 1))
	if err := os.WriteFile(dbPath, data, 0644); err != nil {
		t.Fatal(err)
	}

	n := NewNode(bc, nil)
	n.SetDBPath(dbPath)
	server := httptest.NewServer(n.Handler())
	defer server.Close()

	response, err := http.Get(server.URL + "/verify")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	var result struct {
		Valid       bool `json:"valid"`
		FaultyIndex int  `json:"faulty_index"`
	}
	if err := json.NewDecoder(response.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if result.Valid {
		t.Fatal("expected tampered persisted blockchain to be invalid")
	}
}

func postJSON(t *testing.T, url string, v interface{}) *http.Response {
	data, _ := json.Marshal(v)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post error: %v", err)
	}
	return resp
}

func TestInvalidSignatureRejected(t *testing.T) {
	bc := chain.NewBlockchain(1)
	n := NewNode(bc, nil)
	srv := httptest.NewServer(n.Handler())
	defer srv.Close()

	// create a tx with invalid signature
	tx := block.Transaction{
		Sender:    "deadbeef",
		Recipient: "cafefe",
		Amount:    1,
		Nonce:     1,
		PublicKey: "deadbeef",
		Signature: "bad",
	}

	resp := postJSON(t, srv.URL+"/tx", tx)
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("expected rejection, got OK")
	}
	if len(bc.PendingTransactions) != 0 {
		t.Fatalf("tx should not be in mempool")
	}
}

func TestTransactionPropagatesToPeer(t *testing.T) {
	// node B
	bcB := chain.NewBlockchain(1)
	nB := NewNode(bcB, nil)
	srvB := httptest.NewServer(nB.Handler())
	defer srvB.Close()

	// node A with peer B
	bcA := chain.NewBlockchain(1)
	nA := NewNode(bcA, nil)
	nA.SetPeers([]string{srvB.URL})
	srvA := httptest.NewServer(nA.Handler())
	defer srvA.Close()

	// create a signed tx; first fund the account via FAUCET and mine so balance exists
	priv, pub, err := wallet.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	addr, err := wallet.AddressFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	// fund with faucet
	faucet := block.Transaction{Sender: "FAUCET", Recipient: addr, Amount: 10, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}
	if err := bcA.AddTransaction(faucet); err != nil {
		t.Fatal(err)
	}
	if _, err := bcA.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}

	// also fund node B so it can validate the same transaction
	faucetB := block.Transaction{Sender: "FAUCET", Recipient: addr, Amount: 10, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}
	if err := bcB.AddTransaction(faucetB); err != nil {
		t.Fatal(err)
	}
	if _, err := bcB.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}

	tx := block.Transaction{Sender: addr, Recipient: addr, Amount: 1, Nonce: 1, PublicKey: pub}
	sig, err := wallet.Sign(priv, tx.Payload())
	if err != nil {
		t.Fatal(err)
	}
	tx.Signature = sig

	resp := postJSON(t, srvA.URL+"/tx", tx)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from A, got %d", resp.StatusCode)
	}

	// wait for propagation
	time.Sleep(200 * time.Millisecond)

	if bcB.GetPendingTransactionCount() == 0 {
		t.Fatalf("peer B should have received tx")
	}
}

func TestMinedBlockAcceptedByPeer(t *testing.T) {
	// node B
	bcB := chain.NewBlockchain(1)
	nB := NewNode(bcB, nil)
	srvB := httptest.NewServer(nB.Handler())
	defer srvB.Close()

	// node A with peer B
	bcA := chain.NewBlockchain(1)
	nA := NewNode(bcA, nil)
	nA.SetPeers([]string{srvB.URL})
	srvA := httptest.NewServer(nA.Handler())
	defer srvA.Close()

	// fund A via faucet and mine so there is a committed block to broadcast
	_, pub, err := wallet.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	addr, err := wallet.AddressFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	faucet := block.Transaction{Sender: "FAUCET", Recipient: addr, Amount: 10, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}
	if err := bcA.AddTransaction(faucet); err != nil {
		t.Fatal(err)
	}
	mined, err := bcA.MinePendingTransactions(10)
	if err != nil {
		t.Fatal(err)
	}

	// broadcast mined block to B
	resp := postJSON(t, srvB.URL+"/block", mined)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from B, got %d", resp.StatusCode)
	}

	time.Sleep(100 * time.Millisecond)
	if len(bcB.Blocks) != 2 {
		t.Fatalf("peer B height should increase; got %d blocks", len(bcB.Blocks))
	}
}

func TestMerkleProofEndpointServesInclusionProof(t *testing.T) {
	bc := chain.NewBlockchain(1)
	n := NewNode(bc, nil)
	srv := httptest.NewServer(n.Handler())
	defer srv.Close()

	_, pub, err := wallet.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	addr, err := wallet.AddressFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	faucetTx := block.Transaction{Sender: "FAUCET", Recipient: addr, Amount: 10, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}
	if err := bc.AddTransaction(faucetTx); err != nil {
		t.Fatal(err)
	}
	mined, err := bc.MinePendingTransactions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(mined.Transactions) == 0 {
		t.Fatal("expected mined block to include at least one transaction")
	}

	resp, err := http.Get(srv.URL + "/tx/proof?block_index=1&tx_index=0")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("proof endpoint should succeed, got %d", resp.StatusCode)
	}
	var payload map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload["verified"] != true {
		t.Fatalf("proof should verify for mined transaction, got payload: %#v", payload)
	}
	if payload["merkle_root"] == nil {
		t.Fatal("proof response missing merkle root")
	}
}

func TestNodeIntrospectionAndPeerExchange(t *testing.T) {
	bc := chain.NewBlockchain(1)
	n := NewNode(bc, nil)
	srv := httptest.NewServer(n.Handler())
	defer srv.Close()

	_, pub, err := wallet.GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}
	addr, err := wallet.AddressFromPublicKey(pub)
	if err != nil {
		t.Fatal(err)
	}
	faucet := block.Transaction{Sender: "FAUCET", Recipient: addr, Amount: 10, PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY", Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE"}
	if err := bc.AddTransaction(faucet); err != nil {
		t.Fatal(err)
	}
	if _, err := bc.MinePendingTransactions(10); err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(srv.URL + "/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status endpoint should succeed, got %d", resp.StatusCode)
	}
	var status map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["height"] == nil || status["head"] == nil || status["mempool"] == nil {
		t.Fatalf("status response missing fields: %#v", status)
	}

	resp, err = http.Get(srv.URL + "/chain")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("chain endpoint should succeed, got %d", resp.StatusCode)
	}
	var chainDump []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&chainDump); err != nil {
		t.Fatal(err)
	}
	if len(chainDump) < 2 {
		t.Fatalf("expected at least genesis plus mined block, got %d blocks", len(chainDump))
	}

	peerNodeA := NewNode(chain.NewBlockchain(1), nil)
	peerSrvA := httptest.NewServer(peerNodeA.Handler())
	defer peerSrvA.Close()
	peerNodeB := NewNode(chain.NewBlockchain(1), nil)
	peerSrvB := httptest.NewServer(peerNodeB.Handler())
	defer peerSrvB.Close()
	peerNodeB.SetPeers([]string{peerSrvA.URL})

	peerResp, err := http.Post(peerSrvB.URL+"/peers", "application/json", bytes.NewReader([]byte(`{"addr":"`+peerSrvA.URL+`"}`)))
	if err != nil {
		t.Fatal(err)
	}
	defer peerResp.Body.Close()
	if peerResp.StatusCode != http.StatusOK {
		t.Fatalf("peer registration should accept, got %d", peerResp.StatusCode)
	}

	peerListResp, err := http.Get(peerSrvB.URL + "/peers")
	if err != nil {
		t.Fatal(err)
	}
	defer peerListResp.Body.Close()
	if peerListResp.StatusCode != http.StatusOK {
		t.Fatalf("peers endpoint should succeed, got %d", peerListResp.StatusCode)
	}
	var peerList []string
	if err := json.NewDecoder(peerListResp.Body).Decode(&peerList); err != nil {
		t.Fatal(err)
	}
	if len(peerList) == 0 {
		t.Fatal("expected peer list to include at least the connected peer")
	}
}

func TestSyncDiscoversTransitivePeers(t *testing.T) {
	seed := NewNode(chain.NewBlockchain(1), nil)
	seedServer := httptest.NewServer(seed.Handler())
	defer seedServer.Close()

	middle := NewNode(chain.NewBlockchain(1), nil)
	middleServer := httptest.NewServer(middle.Handler())
	defer middleServer.Close()

	leaf := NewNode(chain.NewBlockchain(1), nil)
	leafServer := httptest.NewServer(leaf.Handler())
	defer leafServer.Close()

	seed.SetPeers([]string{middleServer.URL})
	middle.SetPeers([]string{leafServer.URL})

	joining := NewNode(chain.NewBlockchain(1), nil)
	if err := joining.SyncFrom(seedServer.URL); err != nil {
		t.Fatalf("sync from seed failed: %v", err)
	}

	peers := joining.Peers()
	if len(peers) != 2 {
		t.Fatalf("expected two discovered peers, got %v", peers)
	}
	seen := map[string]bool{}
	for _, peer := range peers {
		seen[peer] = true
	}
	if !seen[middleServer.URL] || !seen[leafServer.URL] {
		t.Fatalf("expected seed topology to discover middle and leaf, got %v", peers)
	}
}
