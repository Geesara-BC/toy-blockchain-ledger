package ledger

import (
	"errors"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/wallet"
)

type Ledger struct {
	Blockchain *chain.Blockchain
}

func NewLedger(bc *chain.Blockchain) *Ledger {
	return &Ledger{
		Blockchain: bc,
	}
}

func (l *Ledger) GetBalances() map[string]float64 {
	balances := make(map[string]float64)

	for _, b := range l.Blockchain.Blocks {
		for _, tx := range b.Transactions {
			if tx.Sender != "COINBASE" && tx.Sender != "FAUCET" {
				balances[tx.Sender] -= tx.Amount
			}
			balances[tx.Recipient] += tx.Amount
		}
	}

	return balances
}

func (l *Ledger) GetBalance(account string) float64 {
	balances := l.GetBalances()
	return balances[account]
}

func (l *Ledger) VerifyTransaction(tx block.Transaction) error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be positive")
	}

	if tx.Sender == "COINBASE" || tx.Sender == "FAUCET" {
		return nil
	}

	if tx.Sender != tx.PublicKey {
		return errors.New("UNAUTHORIZED: Sender address does not match the provided Public Key")
	}

	if !wallet.Verify(tx.PublicKey, tx.Payload(), tx.Signature) {
		return errors.New("INVALID SIGNATURE: Transaction tampered or unauthorized")
	}

	currentBalance := l.GetBalance(tx.Sender)
	if currentBalance < tx.Amount {
		return errors.New("insufficient balance: sender does not have enough funds")
	}

	return nil
}
