package node

import (
    "bytes"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "toy-blockchain/internal/block"
    "toy-blockchain/internal/chain"
    "toy-blockchain/internal/wallet"
)

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

    if len(bcB.PendingTransactions) == 0 {
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
