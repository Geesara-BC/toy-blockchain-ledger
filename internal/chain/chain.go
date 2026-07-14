package chain

import (
	"errors"

	"strings"
	"time"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/wallet"
)

type Blockchain struct {
	Blocks              []*block.Block      `json:"blocks"`
	Difficulty          int                 `json:"difficulty"`
	PendingTransactions []block.Transaction `json:"pending_transactions"`
}

func NewBlockchain(difficulty int) *Blockchain {
	return &Blockchain{
		Blocks:              []*block.Block{block.NewGenesisBlock()},
		Difficulty:          difficulty,
		PendingTransactions: []block.Transaction{},
	}
}

func (bc *Blockchain) GetLatestBlock() *block.Block {
	return bc.Blocks[len(bc.Blocks)-1]
}

func (bc *Blockchain) GetBalances() map[string]int64 {
	balances := make(map[string]int64)

	for _, b := range bc.Blocks {
		for _, tx := range b.Transactions {
			if tx.Sender != "COINBASE" && tx.Sender != "FAUCET" {
				balances[tx.Sender] -= tx.Amount
			}
			balances[tx.Recipient] += tx.Amount
		}
	}

	return balances
}

func (bc *Blockchain) GetBalance(account string) int64 {
	balances := bc.GetBalances()
	return balances[account]
}

func (bc *Blockchain) PendingOutbound(account string) int64 {
	total := int64(0)
	for _, pendingTx := range bc.PendingTransactions {
		if pendingTx.Sender == account {
			total += pendingTx.Amount
		}
	}
	return total
}

// func (bc *Blockchain) AddTransaction(tx block.Transaction) {
// 	bc.PendingTransactions = append(bc.PendingTransactions, tx)
// }

//new

func (bc *Blockchain) ValidateTransaction(tx block.Transaction) error {
	if tx.Amount <= 0 {
		return errors.New("transaction amount must be positive")
	}

	if tx.Sender == "" || tx.Recipient == "" {
		return errors.New("sender and recipient must not be empty")
	}

	if tx.Sender == "COINBASE" || tx.Sender == "FAUCET" {
		return nil
	}

	if tx.PublicKey == "" || tx.Signature == "" {
		return errors.New("UNAUTHORIZED: Transaction must be signed with a valid public key and signature")
	}

	if tx.Sender != tx.PublicKey {
		return errors.New("UNAUTHORIZED: Sender address does not match the provided Public Key")
	}

	if !wallet.Verify(tx.PublicKey, tx.Payload(), tx.Signature) {
		return errors.New("INVALID SIGNATURE: Transaction tampered or unauthorized")
	}

	currentBalance := bc.GetBalance(tx.Sender)
	pendingOutbound := bc.PendingOutbound(tx.Sender)

	if currentBalance-pendingOutbound < tx.Amount {
		return errors.New("insufficient balance: sender does not have enough funds")
	}

	return nil
}

func (bc *Blockchain) AddTransaction(tx block.Transaction) error {
	if err := bc.ValidateTransaction(tx); err != nil {
		return err
	}
	bc.PendingTransactions = append(bc.PendingTransactions, tx)
	return nil
}

func (bc *Blockchain) MinePendingTransactions(maxBlockSize int) (*block.Block, error) {
	if len(bc.PendingTransactions) == 0 {
		return nil, errors.New("no pending transactions to mine")
	}
	txCount := len(bc.PendingTransactions)
	if txCount > maxBlockSize {
		txCount = maxBlockSize
	}

	transactionsToMine := bc.PendingTransactions[:txCount]

	latestBlock := bc.GetLatestBlock()

	newBlock := &block.Block{
		Index:        latestBlock.Index + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: transactionsToMine,
		PreviousHash: latestBlock.Hash,
		Nonce:        0,
		Difficulty:   bc.Difficulty,
	}

	targetPrefix := strings.Repeat("0", bc.Difficulty)

	for {
		newBlock.Hash = newBlock.CalculateHash()
		if strings.HasPrefix(newBlock.Hash, targetPrefix) {
			break
		}
		newBlock.Nonce++
	}

	bc.Blocks = append(bc.Blocks, newBlock)
	bc.PendingTransactions = bc.PendingTransactions[txCount:]
	return newBlock, nil
}

func (bc *Blockchain) IsValid() (bool, int) {

	if len(bc.Blocks) > 0 {
		genesisBlock := bc.Blocks[0]

		expectedGenesisHash := "24f430256868bc93806e9ebd86a15ea50940b55fb99c8871e5119af4d9f72f36"
		if genesisBlock.Hash != expectedGenesisHash {
			return false, 0
		}

		if genesisBlock.Hash != genesisBlock.CalculateHash() {
			return false, 0
		}

		expectedGenesisPrevHash := "0000000000000000000000000000000000000000000000000000000000000000"
		if genesisBlock.PreviousHash != expectedGenesisPrevHash {
			return false, 0
		}
	}

	for i := 1; i < len(bc.Blocks); i++ {
		current := bc.Blocks[i]
		previous := bc.Blocks[i-1]

		if current.Hash != current.CalculateHash() {
			return false, current.Index
		}
		if current.PreviousHash != previous.Hash {
			return false, current.Index
		}
		targetPrefix := strings.Repeat("0", bc.Difficulty)
		if !strings.HasPrefix(current.Hash, targetPrefix) {
			return false, current.Index
		}
		if current.Index != previous.Index+1 {
			return false, current.Index
		}
		if current.Timestamp < previous.Timestamp {
			return false, current.Index
		}
	}
	return true, -1
}
