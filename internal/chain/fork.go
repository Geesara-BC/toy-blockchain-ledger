package chain

import (
	"errors"
	"fmt"
	"toy-blockchain/internal/block"
)

var ErrInvalidChain = errors.New("candidate chain is invalid")
var ErrShorterChain = errors.New("candidate chain is not longer than the current chain")

// ReplaceWithLongestChain adopts a competing chain when it is valid and longer.
func (bc *Blockchain) ReplaceWithLongestChain(candidate *Blockchain) error {
	if candidate == nil {
		return errors.New("candidate chain is nil")
	}
	if len(candidate.Blocks) <= len(bc.Blocks) {
		return ErrShorterChain
	}

	if len(bc.Blocks) == 0 || len(candidate.Blocks) == 0 {
		return ErrInvalidChain
	}
	if candidate.Blocks[0] == nil || bc.Blocks[0] == nil || candidate.Blocks[0].Hash != bc.Blocks[0].Hash {
		return errors.New("candidate chain does not share the same genesis block")
	}

	if valid, _ := candidate.IsValid(); !valid {
		return ErrInvalidChain
	}

	blocks := make([]*block.Block, len(candidate.Blocks))
	for i, candidateBlock := range candidate.Blocks {
		if candidateBlock == nil {
			return ErrInvalidChain
		}
		copied := *candidateBlock
		blocks[i] = &copied
	}

	bc.Blocks = blocks
	bc.Difficulty = candidate.Difficulty
	bc.BaseDifficulty = candidate.BaseDifficulty
	bc.PendingTransactions = bc.filterPendingTransactions(candidate)
	bc.MinerAddress = candidate.MinerAddress
	bc.MinerWorkers = candidate.MinerWorkers
	bc.MinerTimeout = candidate.MinerTimeout
	bc.rebuildState()

	return nil
}

func (bc *Blockchain) filterPendingTransactions(candidate *Blockchain) []block.Transaction {
	if candidate == nil {
		return nil
	}

	confirmed := make(map[string]struct{}, len(candidate.Blocks))
	for _, candidateBlock := range candidate.Blocks {
		if candidateBlock == nil {
			continue
		}
		for _, tx := range candidateBlock.Transactions {
			confirmed[transactionKey(tx)] = struct{}{}
		}
	}

	pending := make([]block.Transaction, 0, len(bc.PendingTransactions))
	for _, tx := range bc.PendingTransactions {
		if _, exists := confirmed[transactionKey(tx)]; !exists {
			pending = append(pending, tx)
		}
	}
	return pending
}

func transactionKey(tx block.Transaction) string {
	return fmt.Sprintf("%s|%s|%d|%d|%d|%s|%s", tx.Sender, tx.Recipient, tx.Amount, tx.Fee, tx.Nonce, tx.PublicKey, tx.Signature)
}
