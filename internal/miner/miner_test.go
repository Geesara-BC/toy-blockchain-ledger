package miner

import (
	"testing"
	"toy-blockchain/internal/block"
)

func TestRewardEngineBuildsCoinbaseTransaction(t *testing.T) {
	engine := NewRewardEngine(50, NewFixedFeePolicy(0))

	txs := []block.Transaction{
		{Sender: "Alice", Recipient: "Bob", Amount: 10, Fee: 3},
		{Sender: "Carol", Recipient: "Dan", Amount: 20, Fee: 2},
	}

	coinbase, reward, err := engine.BuildCoinbaseTransaction("miner-1", txs)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if coinbase.Sender != "COINBASE" {
		t.Fatalf("expected coinbase sender, got %q", coinbase.Sender)
	}
	if coinbase.Recipient != "miner-1" {
		t.Fatalf("expected miner recipient, got %q", coinbase.Recipient)
	}
	if coinbase.Amount != 55 {
		t.Fatalf("expected reward amount 55, got %d", coinbase.Amount)
	}
	if reward.Total != 55 {
		t.Fatalf("expected total reward 55, got %d", reward.Total)
	}
}
