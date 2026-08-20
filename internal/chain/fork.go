package chain

import (
	"errors"
	"fmt"
	"log"
	"math/big"
	"toy-blockchain/internal/block"
)

var ErrInvalidChain = errors.New("candidate chain is invalid")
var ErrShorterChain = errors.New("candidate chain is not longer or heavier than the current chain")

// ReplaceWithLongestChain adopts a valid chain when it is longer or carries more work.
func (bc *Blockchain) ReplaceWithLongestChain(candidate *Blockchain) error {
	if candidate == nil {
		return errors.New("candidate chain is nil")
	}
	bc.mu.RLock()
	localLen := len(bc.Blocks)
	bc.mu.RUnlock()

	bc.mu.RLock()

	// new

	// new

	// new

	localBlocks := make([]*block.Block, len(bc.Blocks))
	copy(localBlocks, bc.Blocks)
	bc.mu.RUnlock()
	if len(candidate.Blocks) < localLen || (len(candidate.Blocks) == localLen && compareChainWork(candidate.Blocks, localBlocks) <= 0) {
		return ErrShorterChain
	}

	// new

	// new

	// new

	if localLen == 0 || len(candidate.Blocks) == 0 {
		return ErrInvalidChain
	}
	if candidate.Blocks[0] == nil {
		return errors.New("candidate chain does not share the same genesis block")
	}
	bc.mu.RLock()
	if bc.Blocks[0] == nil || candidate.Blocks[0].Hash != bc.Blocks[0].Hash {
		bc.mu.RUnlock()
		return errors.New("candidate chain does not share the same genesis block")
	}
	bc.mu.RUnlock()

	if valid, _ := candidate.IsValid(); !valid {
		return ErrInvalidChain
	}

	// Snapshot current committed blocks and pending txs under read lock
	bc.mu.RLock()
	oldBlocks := make([]*block.Block, len(bc.Blocks))
	copy(oldBlocks, bc.Blocks)
	oldPending := make([]block.Transaction, len(bc.PendingTransactions))
	copy(oldPending, bc.PendingTransactions)
	localLen = len(oldBlocks)
	bc.mu.RUnlock()

	// Build a set of transaction keys that the candidate confirms
	confirmed := make(map[string]struct{})
	for _, cb := range candidate.Blocks {
		if cb == nil {
			return ErrInvalidChain
		}
		for _, tx := range cb.Transactions {
			confirmed[transactionKey(tx)] = struct{}{}
		}
	}

	// Collect orphaned transactions from the old chain (committed previously but not in candidate)
	orphaned := make([]block.Transaction, 0)
	for _, ob := range oldBlocks {
		if ob == nil {
			continue
		}
		for _, tx := range ob.Transactions {
			// skip coinbase/faucet system transactions
			if tx.Sender == "COINBASE" || tx.Sender == "FAUCET" {
				continue
			}
			key := transactionKey(tx)
			if _, exists := confirmed[key]; !exists {
				orphaned = append(orphaned, tx)
			}
		}
	}

	// Validate orphaned txs against candidate; only resurrect those that remain valid
	resurrected := make([]block.Transaction, 0)
	for _, tx := range orphaned {
		if err := candidate.ValidateTransaction(tx); err == nil {
			resurrected = append(resurrected, tx)
		}
	}

	blocks := make([]*block.Block, len(candidate.Blocks))
	for i, candidateBlock := range candidate.Blocks {
		copied := *candidateBlock
		blocks[i] = &copied
	}

	// Commit replacement under write lock for atomic swap of core fields
	bc.mu.Lock()
	bc.Blocks = blocks
	bc.Difficulty = candidate.Difficulty
	bc.BaseDifficulty = candidate.BaseDifficulty

	// Recompute pending transactions: keep prior pending txs that are not confirmed by candidate
	pendingMap := make(map[string]struct{})
	pending := make([]block.Transaction, 0)
	for _, tx := range oldPending {
		key := transactionKey(tx)
		if _, ok := confirmed[key]; !ok {
			pending = append(pending, tx)
			pendingMap[key] = struct{}{}
		}
	}
	// append resurrected txs if not already present
	for _, tx := range resurrected {
		key := transactionKey(tx)
		if _, ok := pendingMap[key]; !ok {
			pending = append(pending, tx)
			pendingMap[key] = struct{}{}
		}
	}
	bc.PendingTransactions = pending

	bc.MinerAddress = candidate.MinerAddress
	bc.MinerWorkers = candidate.MinerWorkers
	bc.MinerTimeout = candidate.MinerTimeout
	bc.mu.Unlock()

	// Rebuild state (rebuildState will take its own lock)
	bc.rebuildState()

	log.Printf("[reorg] replaced chain len %d -> %d; orphaned=%d; resurrected=%d", localLen, len(candidate.Blocks), len(orphaned), len(resurrected))

	return nil
}

// new

// new

// new
func compareChainWork(candidate, current []*block.Block) int {
	candidateWork := chainWork(candidate)
	currentWork := chainWork(current)
	return candidateWork.Cmp(currentWork)
}

func chainWork(blocks []*block.Block) *big.Int {
	work := new(big.Int)
	base := big.NewInt(16)
	for _, current := range blocks {
		if current == nil || current.Difficulty <= 0 {
			continue
		}
		blockWork := new(big.Int).Exp(base, big.NewInt(int64(current.Difficulty)), nil)
		work.Add(work, blockWork)
	}
	return work
}

// new

// new

// new

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
