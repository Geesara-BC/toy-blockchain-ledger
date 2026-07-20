package miner

import (
	"fmt"
	"toy-blockchain/internal/block"
)

// Reward represents the total reward paid to a miner for a mined block.
type Reward struct {
	BlockReward int64
	Fees        int64
	Total       int64
}

// FeePolicy defines how transaction fees are collected for a block.
type FeePolicy interface {
	CalculateFees(transactions []block.Transaction) int64
}

// FixedFeePolicy is a simple fee policy for the current toy blockchain.
type FixedFeePolicy struct {
	FeePerTransaction int64
}

func NewFixedFeePolicy(feePerTransaction int64) FeePolicy {
	return &FixedFeePolicy{FeePerTransaction: feePerTransaction}
}

func (p *FixedFeePolicy) CalculateFees(transactions []block.Transaction) int64 {
	total := int64(0)
	for _, tx := range transactions {
		total += tx.Fee
	}
	return total
}

// RewardEngine builds reward transactions for miners.
type RewardEngine struct {
	BlockReward int64
	FeePolicy   FeePolicy
}

func NewRewardEngine(blockReward int64, feePolicy FeePolicy) *RewardEngine {
	if feePolicy == nil {
		feePolicy = NewFixedFeePolicy(0)
	}
	return &RewardEngine{BlockReward: blockReward, FeePolicy: feePolicy}
}

func (r *RewardEngine) CalculateReward(transactions []block.Transaction) Reward {
	fees := r.FeePolicy.CalculateFees(transactions)
	return Reward{
		BlockReward: r.BlockReward,
		Fees:        fees,
		Total:       r.BlockReward + fees,
	}
}

func (r *RewardEngine) BuildCoinbaseTransaction(minerAddress string, transactions []block.Transaction) (block.Transaction, Reward, error) {
	if minerAddress == "" {
		return block.Transaction{}, Reward{}, fmt.Errorf("miner address must not be empty")
	}

	reward := r.CalculateReward(transactions)
	coinbase := block.Transaction{
		Sender:    "COINBASE",
		Recipient: minerAddress,
		Amount:    reward.Total,
		Fee:       0,
	}
	return coinbase, reward, nil
}
