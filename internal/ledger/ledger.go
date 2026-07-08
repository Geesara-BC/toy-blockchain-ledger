package ledger

import (
	"errors"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
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

	currentBalance := l.GetBalance(tx.Sender)
	if currentBalance < tx.Amount {
		return errors.New("insufficient balance: sender does not have enough funds")
	}

	return nil
}
