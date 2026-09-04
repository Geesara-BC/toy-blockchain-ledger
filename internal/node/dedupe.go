package node

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"time"
)

func idFor(v interface{}) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func (n *Node) seenTxRecently(id string) bool {
	n.seenMu.Lock()
	defer n.seenMu.Unlock()
	n.cleanupSeenMapsLocked()
	if _, ok := n.seenTx[id]; ok {
		return true
	}
	n.seenTx[id] = time.Now()
	return false
}

// hasSeenTx returns true if tx id exists in seen map without mutating it.
func (n *Node) hasSeenTx(id string) bool {
	n.seenMu.Lock()
	defer n.seenMu.Unlock()
	n.cleanupSeenMapsLocked()
	_, ok := n.seenTx[id]
	return ok
}

// markTxSeen marks a tx id as seen with the current timestamp.
func (n *Node) markTxSeen(id string) {
	n.seenMu.Lock()
	defer n.seenMu.Unlock()
	n.cleanupSeenMapsLocked()
	n.seenTx[id] = time.Now()
}

func (n *Node) seenBlockRecently(id string) bool {
	n.seenMu.Lock()
	defer n.seenMu.Unlock()
	n.cleanupSeenMapsLocked()
	if _, ok := n.seenBlock[id]; ok {
		return true
	}
	n.seenBlock[id] = time.Now()
	return false
}

func (n *Node) cleanupSeenMapsLocked() {
	now := time.Now()
	for id, seenAt := range n.seenTx {
		if now.Sub(seenAt) > dedupeTTL {
			delete(n.seenTx, id)
		}
	}
	for id, seenAt := range n.seenBlock {
		if now.Sub(seenAt) > dedupeTTL {
			delete(n.seenBlock, id)
		}
	}
}
