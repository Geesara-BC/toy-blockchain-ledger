package node

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/wallet"
)

func (n *Node) SyncFrom(peer string) error {
	resp, err := n.client.Get(strings.TrimRight(peer, "/") + "/sync/height")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	var info struct {
		Height int    `json:"height"`
		Head   string `json:"head"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return err
	}

	candidate := chain.NewBlockchainWithDifficultyConfig(n.BC.Difficulty, n.BC.DifficultyConfig)
	candidate.Blocks = make([]*block.Block, 0, info.Height+1)

	tempBalances := make(map[string]int64)
	tempNonces := make(map[string]int64)

	for i := 0; i <= info.Height; i++ {
		url := fmt.Sprintf("%s/sync/block/%d", strings.TrimRight(peer, "/"), i)
		r2, err := n.client.Get(url)
		if err != nil {
			return err
		}

		var b block.Block
		if err := json.NewDecoder(r2.Body).Decode(&b); err != nil {
			r2.Body.Close()
			return err
		}
		r2.Body.Close()

		if b.Hash != b.CalculateHash() {
			return fmt.Errorf("invalid block hash at index %d", i)
		}

		if err := validateBlockTransactionsAgainstState(b.Transactions, tempBalances, tempNonces); err != nil {
			return fmt.Errorf("invalid transaction in synced block %d: %w", i, err)
		}

		copyb := b
		candidate.Blocks = append(candidate.Blocks, &copyb)
	}

	candidate.RebuildState()
	if valid, _ := candidate.IsValid(); !valid {
		return fmt.Errorf("invalid candidate chain")
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	if err := n.BC.ReplaceWithLongestChain(candidate); err != nil {
		return err
	}
	return nil
}

func validateBlockTransactionsAgainstState(transactions []block.Transaction, balances, nonces map[string]int64) error {
	for _, tx := range transactions {
		if tx.Amount <= 0 {
			return errors.New("transaction amount must be positive")
		}
		if tx.Sender == "" || tx.Recipient == "" {
			return errors.New("sender and recipient must not be empty")
		}
		if tx.Sender == "COINBASE" || tx.Sender == "FAUCET" {
			balances[tx.Recipient] += tx.Amount
			continue
		}
		if tx.PublicKey == "" || tx.Signature == "" {
			return errors.New("UNAUTHORIZED: Transaction must be signed with a valid public key and signature")
		}
		derivedAddress, err := wallet.AddressFromPublicKey(tx.PublicKey)
		if err != nil {
			return errors.New("UNAUTHORIZED: invalid public key")
		}
		if tx.Sender != derivedAddress {
			return errors.New("UNAUTHORIZED: Sender address does not match the provided Public Key")
		}
		if !tx.VerifySignature() {
			return errors.New("INVALID SIGNATURE: Transaction tampered or unauthorized")
		}
		if tx.Nonce <= 0 {
			return errors.New("INVALID_NONCE: transaction nonce must be positive")
		}

		currentNonce := nonces[tx.Sender]
		if tx.Nonce <= currentNonce {
			return errors.New("INVALID_NONCE: transaction nonce already processed or outdated")
		}
		expectedNonce := currentNonce + 1
		if tx.Nonce != expectedNonce {
			return fmt.Errorf("INVALID_NONCE: transaction nonce out of order, expected %d got %d", expectedNonce, tx.Nonce)
		}

		if balances[tx.Sender] < tx.Amount {
			return errors.New("insufficient balance: sender does not have enough funds")
		}

		balances[tx.Sender] -= tx.Amount
		balances[tx.Recipient] += tx.Amount
		nonces[tx.Sender] = tx.Nonce
	}
	return nil
}
