package node

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"path"
	"strings"

	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
)

func writeJSONError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (n *Node) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tx", n.handleTx)
	mux.HandleFunc("/block", n.handleBlock)
	mux.HandleFunc("/mine", n.handleMine)
	mux.HandleFunc("/wallet", n.handleWallet)
	mux.HandleFunc("/wallets", n.handleWallet)
	mux.HandleFunc("/peers", n.handlePeers)
	mux.HandleFunc("/sync/height", n.handleSyncHeight)
	mux.HandleFunc("/sync/block/", n.handleSyncBlock)
	mux.HandleFunc("/balances", n.handleBalances)
	mux.HandleFunc("/nonce/", n.handleNonce)
	mux.HandleFunc("/status", n.handleStatus)
	mux.HandleFunc("/verify", n.handleVerify)
	return mux
}

func (n *Node) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "method", http.StatusMethodNotAllowed)
		return
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	valid, faulty := n.BC.IsValid()
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"valid": valid, "faulty_index": faulty})
}

func (n *Node) handleNonce(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "method", http.StatusMethodNotAllowed)
		return
	}
	addr := path.Base(r.URL.Path)
	if addr == "" || addr == "/" {
		writeJSONError(w, "bad address", http.StatusBadRequest)
		return
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	// return next pending nonce (GetPendingAccountNonce returns last+pending)
	nonce := n.BC.GetPendingAccountNonce(addr) + 1
	_ = json.NewEncoder(w).Encode(map[string]int64{"next_nonce": nonce})
}

func (n *Node) handleBalances(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, "method", http.StatusMethodNotAllowed)
		return
	}
	n.mu.RLock()
	defer n.mu.RUnlock()
	balances := n.BC.GetBalances()
	_ = json.NewEncoder(w).Encode(balances)
}

func (n *Node) handleWallet(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		addressBook, privateKeys, err := n.LoadWalletStore()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "load wallet store failed: " + err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"address_book": addressBook,
			"private_keys": privateKeys,
			"wallet_path":  n.WalletStorePath(),
		})
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "bad request: invalid json",
			})
			return
		}

		name := strings.TrimSpace(req.Name)
		if name == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": "wallet name cannot be empty",
			})
			return
		}

		addr, pubKey, err := n.CreateWalletForNode(name)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusBadRequest)
			_ = json.NewEncoder(w).Encode(map[string]string{
				"error": err.Error(),
			})
			return
		}

		// ✅ නිවැරදි Response Payload එක:
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        name,
			"address":     addr,
			"public_key":  pubKey,
			"wallet_path": n.WalletStorePath(),
		})
		return
	}

	writeJSONError(w, "method", http.StatusMethodNotAllowed)
}
func (n *Node) handleMine(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "method", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		MaxBlockSize int `json:"max_block_size"`
	}
	if r.Body != nil && r.ContentLength != 0 {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONError(w, "bad request", http.StatusBadRequest)
			return
		}
	}
	if req.MaxBlockSize <= 0 {
		req.MaxBlockSize = 10
	}

	n.mu.RLock()
	if n.BC == nil {
		n.mu.RUnlock()
		writeJSONError(w, "blockchain is not initialized", http.StatusInternalServerError)
		return
	}
	chainCopy := n.BC
	n.mu.RUnlock()

	mined, err := chainCopy.MinePendingTransactions(req.MaxBlockSize)
	if err != nil {
		writeJSONError(w, "mine failed: "+err.Error(), http.StatusBadRequest)
		return
	}

	if n.dbPath != "" {
		if err := n.Save(); err != nil {
			log.Printf("[Node persistence] failed to save mined block: %v", err)
		}
	}

	n.markBlockSeen(mined)
	go n.gossipToPeers("/block", *mined)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"block":  mined,
	})
}

func (n *Node) handleTx(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var tx block.Transaction
	if err := json.NewDecoder(r.Body).Decode(&tx); err != nil {
		writeJSONError(w, "bad request", http.StatusBadRequest)
		return
	}

	// If we've already seen this tx recently, acknowledge without re-adding.
	if n.hasSeenTx(idFor(tx)) {
		w.WriteHeader(http.StatusOK)
		return
	}

	n.mu.Lock()
	if err := n.BC.ValidateTransaction(tx); err != nil {
		n.mu.Unlock()
		writeJSONError(w, "invalid transaction: "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := n.BC.AddTransaction(tx); err != nil {
		n.mu.Unlock()
		writeJSONError(w, "add failed: "+err.Error(), http.StatusBadRequest)
		return
	}
	n.mu.Unlock()

	// mark tx as seen only after successful validation and add
	n.markTxSeen(idFor(tx))

	if n.dbPath != "" {
		if err := n.Save(); err != nil {
			log.Printf("[Node persistence] failed to save tx update: %v", err)
		}
	}

	go n.gossipToPeers("/tx", tx)
	w.WriteHeader(http.StatusOK)
}

func (n *Node) handleBlock(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, "method", http.StatusMethodNotAllowed)
		return
	}
	var b block.Block
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		writeJSONError(w, "bad request", http.StatusBadRequest)
		return
	}

	if n.seenBlockRecently(idFor(b)) {
		w.WriteHeader(http.StatusOK)
		return
	}

	if b.Hash != b.CalculateHash() {
		writeJSONError(w, "invalid block hash", http.StatusBadRequest)
		return
	}

	n.mu.Lock()
	if err := n.BC.ValidateBlockTransactions(b.Transactions); err != nil {
		n.mu.Unlock()
		writeJSONError(w, "invalid transaction in block: "+err.Error(), http.StatusBadRequest)
		return
	}
	latest := n.BC.GetLatestBlock()
	if b.PreviousHash == latest.Hash {
		if b.Index != latest.Index+1 {
			n.mu.Unlock()
			writeJSONError(w, "invalid block index", http.StatusBadRequest)
			return
		}
		if b.MerkleRootValue != "" && b.MerkleRootValue != b.MerkleRoot() {
			n.mu.Unlock()
			writeJSONError(w, "invalid merkle root", http.StatusBadRequest)
			return
		}

		candidate := chain.NewBlockchainWithDifficultyConfig(n.BC.Difficulty, n.BC.DifficultyConfig)
		candidate.Blocks = make([]*block.Block, len(n.BC.Blocks)+1)
		for i, existing := range n.BC.Blocks {
			copyBlock := *existing
			candidate.Blocks[i] = &copyBlock
		}
		copyBlock := b
		copyBlock.IsImmutable = true
		candidate.Blocks[len(n.BC.Blocks)] = &copyBlock
		candidate.RebuildState()
		if valid, faultyIndex := candidate.IsValid(); !valid {
			n.mu.Unlock()
			writeJSONError(w, fmt.Sprintf("invalid block at index %d", faultyIndex), http.StatusBadRequest)
			return
		}

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
		if n.dbPath != "" {
			if err := n.Save(); err != nil {
				log.Printf("[Node persistence] failed to save block update: %v", err)
			}
		}
		go n.gossipToPeers("/block", b)
		w.WriteHeader(http.StatusOK)
		return
	}
	n.mu.Unlock()

	// A competing block may be the tip of a longer chain. Ask peers for their
	// chains so ReplaceWithLongestChain can perform the shared reorganisation.
	for _, peer := range n.Peers() {
		go func(peerURL string) {
			if err := n.SyncFrom(peerURL); err != nil {
				log.Printf("[Node sync] failed after competing block from %s: %v", peerURL, err)
			}
		}(peer)
	}

	writeJSONError(w, "competing block or wrong prev", http.StatusConflict)
}

func (n *Node) handlePeers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		n.peersMu.RLock()
		defer n.peersMu.RUnlock()
		_ = json.NewEncoder(w).Encode(n.peers)
	case http.MethodPost:
		var p struct {
			Addr string `json:"addr"`
		}
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			writeJSONError(w, "bad request", http.StatusBadRequest)
			return
		}
		if p.Addr == "" {
			writeJSONError(w, "empty", http.StatusBadRequest)
			return
		}
		n.addPeer(p.Addr)
		go func(addr string) { _ = n.SyncFrom(addr) }(p.Addr)
		w.WriteHeader(http.StatusOK)
	default:
		writeJSONError(w, "method", http.StatusMethodNotAllowed)
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
