package miner

import (
	"errors"
	"sync"
)

// Registry stores miner addresses and is intentionally simple for this toy blockchain.
// It is designed to be extended later with stats, metadata, and reward policy hooks.
type Registry struct {
	mu      sync.RWMutex
	miners  map[string]struct{}
	weights map[string]int64
}

func NewRegistry() *Registry {
	return &Registry{
		miners:  make(map[string]struct{}),
		weights: make(map[string]int64),
	}
}

func (r *Registry) Register(address string) error {
	if address == "" {
		return errors.New("invalid miner address")
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.miners[address] = struct{}{}
	r.weights[address] = 1
	return nil
}

func (r *Registry) IsRegistered(address string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.miners[address]
	return ok
}

func (r *Registry) Unregister(address string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.miners, address)
	delete(r.weights, address)
}

func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	addresses := make([]string, 0, len(r.miners))
	for addr := range r.miners {
		addresses = append(addresses, addr)
	}
	return addresses
}
