package node

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/storage"
	"toy-blockchain/internal/wallet"
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
	dbPath string
}

const dedupeTTL = 5 * time.Minute

func NewNode(bc *chain.Blockchain, client *http.Client) *Node {
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   20,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   5 * time.Second,
				ExpectContinueTimeout: 1 * time.Second,
			},
		}
	}
	return &Node{
		BC:        bc,
		seenTx:    make(map[string]time.Time),
		seenBlock: make(map[string]time.Time),
		client:    client,
	}
}

func (n *Node) SetDBPath(p string) {
	if p == "" {
		return
	}
	n.dbPath = p
}

func (n *Node) Save() error {
	if n.dbPath == "" || n.BC == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(n.dbPath), 0o755); err != nil {
		return err
	}
	return storage.SaveToFile(n.dbPath, n.BC)
}

func (n *Node) WalletStorePath() string {
	// Wallets are shared across nodes and stored at data/wallets.json
	return filepath.Join("data", "wallets.json")
}

func (n *Node) LoadWalletStore() (map[string]string, map[string]string, error) {
	path := n.WalletStorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		// attempt migration from node-local wallets.json (legacy)
		if n.dbPath != "" {
			local := filepath.Join(filepath.Dir(n.dbPath), "wallets.json")
			if b, err2 := os.ReadFile(local); err2 == nil {
				// persist to shared path
				_ = os.MkdirAll(filepath.Dir(path), 0o755)
				_ = os.WriteFile(path, b, 0644)
				data = b
			} else {
				return make(map[string]string), make(map[string]string), nil
			}
		} else {
			return make(map[string]string), make(map[string]string), nil
		}
	}
	var store struct {
		AddressBook map[string]string `json:"address_book"`
		PrivateKeys map[string]string `json:"private_keys"`
	}
	if err := json.Unmarshal(data, &store); err != nil {
		return make(map[string]string), make(map[string]string), err
	}
	return store.AddressBook, store.PrivateKeys, nil
}

func (n *Node) persistWalletStore(addresses, privates map[string]string) error {
	store := struct {
		AddressBook map[string]string `json:"address_book"`
		PrivateKeys map[string]string `json:"private_keys"`
	}{addresses, privates}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	p := n.WalletStorePath()
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	return os.WriteFile(p, data, 0644)
}

// PersistWalletStore is an exported wrapper to persist the shared wallet store.
func (n *Node) PersistWalletStore(addresses, privates map[string]string) error {
	return n.persistWalletStore(addresses, privates)
}

// Keystore handling per-node
type KeyStore struct {
	MinerAddress    string `json:"miner_address"`
	MinerPrivateKey string `json:"miner_private_key"`
}

func (n *Node) KeystorePath() string {
	if n.dbPath == "" {
		return filepath.Join("data", "keystore.json")
	}
	dir := filepath.Dir(n.dbPath)
	return filepath.Join(dir, "keystore.json")
}

func (n *Node) LoadKeystore() (*KeyStore, error) {
	path := n.KeystorePath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var ks KeyStore
	if err := json.Unmarshal(data, &ks); err != nil {
		return nil, err
	}
	return &ks, nil
}

func (n *Node) SaveKeystore(ks *KeyStore) error {
	if ks == nil {
		return nil
	}
	data, err := json.MarshalIndent(ks, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(n.KeystorePath()), 0o755); err != nil {
		return err
	}
	return os.WriteFile(n.KeystorePath(), data, 0644)
}



func (n *Node) CreateWalletForNode(name string) (string, string, error) {
	addresses, privates, err := n.LoadWalletStore()
	if err != nil {
		return "", "", err
	}

	// 1. එකම නමින් Wallet එකක් දැනටමත් තිබේදැයි පරීක්ෂා කිරීම (Duplicate Check)
	if _, exists := addresses[name]; exists {
		return "", "", fmt.Errorf("wallet with name '%s' already exists", name)
	}

	// 2. නම භාවිතයේ නැතිනම් පමණක් අලුත් Keypair එක සෑදීම
	priv, pub, err := wallet.GenerateKeyPair()
	if err != nil {
		return "", "", err
	}
	addr, _ := wallet.AddressFromPublicKey(pub)

	// Store the derived on-chain address in the address book (not raw public key)
	addresses[name] = addr
	privates[name] = priv
	_ = n.persistWalletStore(addresses, privates)
	return addr, pub, nil
}

func (n *Node) markBlockSeen(b interface{}) {
	id := idFor(b)
	n.seenMu.Lock()
	n.seenBlock[id] = time.Now()
	n.seenMu.Unlock()
}

func (n *Node) SetPeers(peers []string) {
	n.peersMu.Lock()
	defer n.peersMu.Unlock()
	n.peers = append([]string{}, peers...)
}

func (n *Node) Peers() []string {
	n.peersMu.RLock()
	defer n.peersMu.RUnlock()
	return append([]string{}, n.peers...)
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

func (n *Node) distinctPeers(peers []string) []string {
	seen := make(map[string]struct{}, len(peers))
	out := make([]string, 0, len(peers))
	for _, peer := range peers {
		trimmed := strings.TrimRight(peer, "/")
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}
