package node

import (
	"net/http"
	"sync"
	"time"

	"toy-blockchain/internal/chain"
)

type Node struct {
	BC *chain.Blockchain
	mu sync.RWMutex

	peers   []string
	peersMu sync.RWMutex

	seenTx    map[string]time.Time
	seenBlock map[string]time.Time
	seenMu    sync.Mutex

	client *http.Client
}

func NewNode(bc *chain.Blockchain, client *http.Client) *Node {
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	return &Node{
		BC:        bc,
		seenTx:    make(map[string]time.Time),
		seenBlock: make(map[string]time.Time),
		client:    client,
	}
}

func (n *Node) SetPeers(peers []string) {
	n.peersMu.Lock()
	defer n.peersMu.Unlock()
	n.peers = append([]string{}, peers...)
}

func (n *Node) addPeer(peer string) {
	n.peersMu.Lock()
	defer n.peersMu.Unlock()
	for _, existingPeer := range n.peers {
		if existingPeer == peer {
			return
		}
	}
	n.peers = append(n.peers, peer)
}
