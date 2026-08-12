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
	if _, ok := n.seenTx[id]; ok {
		return true
	}
	n.seenTx[id] = time.Now()
	return false
}

func (n *Node) seenBlockRecently(id string) bool {
	n.seenMu.Lock()
	defer n.seenMu.Unlock()
	if _, ok := n.seenBlock[id]; ok {
		return true
	}
	n.seenBlock[id] = time.Now()
	return false
}
