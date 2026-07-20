package main

import (
	"fmt"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
)

func main() {
	bc := chain.NewBlockchain(1)
	_ = bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	b, err := bc.MinePendingTransactions(5)
	fmt.Println("err", err)
	fmt.Printf("blocks=%d pending=%d\n", len(bc.Blocks), len(bc.PendingTransactions))
	fmt.Printf("block txs=%#v\n", b.Transactions)
	fmt.Printf("balances=%v\n", bc.GetBalances())
	fmt.Printf("alice=%d\n", bc.GetBalance("Alice"))
}
