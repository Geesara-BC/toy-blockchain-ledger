package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path"

	"toy-blockchain/internal/block"
)

func (n *Node) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tx", n.handleTx)
	mux.HandleFunc("/block", n.handleBlock)
	mux.HandleFunc("/peers", n.handlePeers)
	mux.HandleFunc("/sync/height", n.handleSyncHeight)
	mux.HandleFunc("/sync/block/", n.handleSyncBlock)
	mux.HandleFunc("/status", n.handleStatus)
	return mux
}

func (n *Node) handleTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var tx block.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if n.seenTxRecently(idFor(tx)) {
		w.WriteHeader(http.StatusOK)
		return
	}

	n.mu.Lock()
	if err := n.BC.ValidateTransaction(tx); err != nil {
		n.mu.Unlock()
		http.Error(w, "invalid transaction: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := n.BC.AddTransaction(tx); err != nil {
		n.mu.Unlock()
		http.Error(w, "add failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	n.mu.Unlock()

	go n.gossipToPeers("/tx", tx)
	w.WriteHeader(http.StatusOK)
}

func (n *Node) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var b block.Block
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	if n.seenBlockRecently(idFor(b)) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if b.Hash != b.CalculateHash() {
		http.Error(w, "invalid block hash", http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	if err := n.BC.ValidateBlockTransactions(b.Transactions); err != nil {
		n.mu.Unlock()
		http.Error(w, "invalid transaction in block: "+err.Error(), http.StatusBadRequest)
		return
	}
	latest := n.BC.GetLatestBlock()
	if b.PreviousHash == latest.Hash {
		b.IsImmutable = true
		n.BC.Blocks = append(n.BC.Blocks, &b)
		remain := make([]block.Transaction, 0, len(n.BC.PendingTransactions))
		for _, ptx := range n.BC.PendingTransactions {
			found := false
			for _, itx := range b.Transactions {
				if idFor(ptx) == idFor(itx) {
					found = true
					break
				}
			}
			if !found {
				remain = append(remain, ptx)
			}
		}
		n.BC.PendingTransactions = remain
		n.BC.RebuildState()
		n.mu.Unlock()
		go n.gossipToPeers("/block", b)
		w.WriteHeader(http.StatusOK)
		return
	}
	n.mu.Unlock()

	http.Error(w, "competing block or wrong prev", http.StatusConflict)
}

func (n *Node) handlePeers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		n.peersMu.RLock()
		defer n.peersMu.RUnlock()
		_ = json.NewEncoder(w).Encode(n.peers)
	case http.MethodPost:
		var p struct{ Addr string `json:"addr"` }
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}
		if p.Addr == "" {
			http.Error(w, "empty", http.StatusBadRequest)
			return
		}
		n.addPeer(p.Addr)
		go func(addr string) { _ = n.SyncFrom(addr) }(p.Addr)
		w.WriteHeader(http.StatusOK)
	default:
		http.Error(w, "method", http.StatusMethodNotAllowed)
	}
}

func (n *Node) handleSyncHeight(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	latest := n.BC.GetLatestBlock()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"height": latest.Index, "head": latest.Hash})
}

func (n *Node) handleSyncBlock(w http.ResponseWriter, r *http.Request) {
	idxStr := path.Base(r.URL.Path)
	var idx int
	if _, err := fmt.Sscanf(idxStr, "%d", &idx); err != nil {
		http.Error(w, "bad index", http.StatusBadRequest)
		return
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	if idx < 0 || idx >= len(n.BC.Blocks) {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	_ = json.NewEncoder(w).Encode(n.BC.Blocks[idx])
}

func (n *Node) handleStatus(w http.ResponseWriter, r *http.Request) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	latest := n.BC.GetLatestBlock()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"height":  len(n.BC.Blocks) - 1,
		"head":    latest.Hash,
		"mempool": len(n.BC.PendingTransactions),
	})
}
