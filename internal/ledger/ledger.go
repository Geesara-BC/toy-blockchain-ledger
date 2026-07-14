package ledger

import (
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

func (l *Ledger) GetBalances() map[string]int64 {
	return l.Blockchain.GetBalances()
}

func (l *Ledger) GetBalance(account string) int64 {
	return l.Blockchain.GetBalance(account)
}

func (l *Ledger) VerifyTransaction(tx block.Transaction) error {
	return l.Blockchain.ValidateTransaction(tx)
}
