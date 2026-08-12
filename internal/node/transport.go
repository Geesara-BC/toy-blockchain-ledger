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
	// 1. Peer list එක thread-safe ලෙස snapshot එකක් ගනී
	n.peersMu.RLock()
	if len(n.peers) == 0 {
		n.peersMu.RUnlock()
		return
	}
	peers := append([]string{}, n.peers...)
	n.peersMu.RUnlock()

	// 2. Loop එකට කලින් JSON එක සාදා ගෙන Error එක Handle කරයි
	data, err := json.Marshal(payload)
	if err != nil {
		log.Printf("[Gossip Error] Failed to marshal payload: %v", err)
		return
	}

	// 3. Goroutines ගණන සීමා කිරීමට Concurrency Semaphore එකක් (Max 20 concurrent requests)
	const maxConcurrency = 20
	sem := make(chan struct{}, maxConcurrency)

	for _, p := range peers {
		url := strings.TrimRight(p, "/") + path

		sem <- struct{}{} // token එක ලබා ගනී
		go func(targetURL string) {
			defer func() { <-sem }() // වැඩේ ඉවර වූ පසු token එක නිදහස් කරයි

			// 4. Request එක සඳහා Context Timeout එකක් (තත්පර 3) යොදයි
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(data))
			if err != nil {
				return
			}
			req.Header.Set("Content-Type", "application/json")

			// 5. HTTP call එක සහ Response Body එක properly Close කිරීම (Socket Leak වැළැක්වීමට)
			resp, err := n.client.Do(req)
			if err != nil {
				return // Peer එක Offline නම් හෝ Timeout වුණොත් මෙතැනින් හැරේ
			}
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
		}(url)
	}
}
