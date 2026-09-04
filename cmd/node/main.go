package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/storage"
	"toy-blockchain/internal/node"
	"toy-blockchain/internal/wallet"
)

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		fmt.Printf("error getting current working directory: %v\n", err)
		os.Exit(1)
	}

	defaultDBPath := filepath.Join(cwd, "data", "blockchain.json")
	if filepath.Base(filepath.Dir(cwd)) == "cmd" && filepath.Base(cwd) == "node" {
		rootPath := filepath.Dir(filepath.Dir(cwd))
		defaultDBPath = filepath.Join(rootPath, "data", "blockchain.json")
	}

	addr := flag.String("addr", ":8081", "listen address for the node HTTP server")
	peers := flag.String("peers", "", "comma-separated peer URLs for initial connection")
	dbPath := flag.String("db-path", defaultDBPath, "file path for blockchain persistence")
	difficulty := flag.Int("difficulty", 3, "starting mining difficulty")
	miningWorkers := flag.Int("mining-workers", 4, "number of goroutines used for concurrent mining")
	difficultyRetargetInterval := flag.Int("difficulty-retarget-interval", chain.DefaultDifficultyRetargetInterval, "how often to retarget difficulty in blocks")
	expectedBlockTimeSeconds := flag.Int("expected-block-time", chain.DefaultExpectedBlockTimeSeconds, "target block time in seconds used by difficulty retargeting")
	flag.Parse()

	if err := os.MkdirAll(filepath.Dir(*dbPath), 0o755); err != nil {
		fmt.Printf("error creating data directory: %v\n", err)
		os.Exit(1)
	}

	bc := chain.NewBlockchainWithDifficultyConfig(*difficulty, chain.DifficultyConfig{
		RetargetInterval:         *difficultyRetargetInterval,
		ExpectedBlockTimeSeconds: *expectedBlockTimeSeconds,
	})
	bc.MinerWorkers = *miningWorkers
	bc.MinerTimeout = 10 * time.Second

	// try to load existing blockchain from dbPath if present
	if _, err := os.Stat(*dbPath); err == nil {
		if loaded, err := storage.LoadFromFile(*dbPath); err == nil {
			bc = loaded
		}
	}

	n := node.NewNode(bc, nil)
	n.SetDBPath(*dbPath)
	// ensure keystore exists for this node; if not, create one automatically
	if ks, err := n.LoadKeystore(); err == nil && ks != nil && ks.MinerAddress != "" {
		bc.MinerAddress = ks.MinerAddress
	} else {
		// create or load a per-node keystore; ensure each node gets a distinct miner address
		addresses, privs, _ := n.LoadWalletStore()
		// derive a node-unique name from the db path (e.g., data/node1 -> node1)
		nodeDir := filepath.Base(filepath.Dir(*dbPath))
		chosenName := nodeDir
		chosenAddr := ""
		chosenPriv := ""
		if v, ok := addresses[chosenName]; ok {
			chosenAddr = v
			if p, ok := privs[chosenName]; ok {
				chosenPriv = p
			}
		} else {
			// generate a new wallet entry dedicated to this node
			priv, pub, err := wallet.GenerateKeyPair()
			if err == nil {
				addr, _ := wallet.AddressFromPublicKey(pub)
				if chosenName == "" {
					chosenName = "node-miner"
				}
				addresses[chosenName] = addr
				privs[chosenName] = priv
				_ = n.PersistWalletStore(addresses, privs)
				chosenAddr = addr
				chosenPriv = priv
			}
		}
		if chosenAddr != "" {
			ks := &node.KeyStore{MinerAddress: chosenAddr, MinerPrivateKey: chosenPriv}
			_ = n.SaveKeystore(ks)
			bc.MinerAddress = chosenAddr
		}
	}
	if parsedPeers := parsePeers(*peers); len(parsedPeers) > 0 {
		n.SetPeers(parsedPeers)
		for _, peerURL := range parsedPeers {
			go func(peer string) {
				_ = n.SyncFrom(peer)
			}(peerURL)
		}
	}

	fmt.Printf("starting node http server on %s\n", *addr)
	fmt.Printf("known peers: %v\n", n.Peers())
	if err := http.ListenAndServe(*addr, n.Handler()); err != nil {
		fmt.Printf("node server exited: %v\n", err)
		os.Exit(1)
	}
}

func parsePeers(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	peers := make([]string, 0, len(parts))
	for _, part := range parts {
		peer := strings.TrimSpace(part)
		if peer == "" {
			continue
		}
		if !strings.HasPrefix(peer, "http://") && !strings.HasPrefix(peer, "https://") {
			peer = "http://" + peer
		}
		peers = append(peers, peer)
	}
	return peers
}
