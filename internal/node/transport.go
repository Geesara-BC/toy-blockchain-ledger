package node

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

func (n *Node) gossipToPeers(path string, payload interface{}) {
	n.peersMu.RLock()
	if len(n.peers) == 0 {
		n.peersMu.RUnlock()
		return
	}
	peers := n.distinctPeers(append([]string{}, n.peers...))
	n.peersMu.RUnlock()

	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Gossip Error] Failed to marshal payload: %v", err)
		return
	}

	const (
		maxConcurrency = 20
		requestTimeout = 5 * time.Second
	)
	sem := make(chan struct{}, maxConcurrency)

	for _, p := range peers {
		url := strings.TrimRight(p, "/") + path
		sem <- struct{}{}
		go func(targetURL string) {
			defer func() { <-sem }()

			ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(data))
			if err != nil {
				log.Printf("[Gossip Error] invalid request for %s: %v", targetURL, err)
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json")

			resp, err := n.client.Do(req)
			if err != nil {
				n.removePeer(targetURL)
				log.Printf("[Gossip Error] peer %s unreachable: %v", targetURL, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
				_, _ = io.Copy(io.Discard, resp.Body)
				n.removePeer(targetURL)
				log.Printf("[Gossip Error] peer %s returned status %s", targetURL, resp.Status)
				return
			}

			_, _ = io.Copy(io.Discard, resp.Body)
		}(url)
	}
}
