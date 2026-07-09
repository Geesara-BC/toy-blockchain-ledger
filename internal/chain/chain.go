package chain

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
	"toy-blockchain/internal/block"
)

type Blockchain struct {
	Blocks              []*block.Block      `json:"blocks"`
	Difficulty          int                 `json:"difficulty"`
	PendingTransactions []block.Transaction `json:"pending_transactions"` // 🌟 Added Pending Pool
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

func (bc *Blockchain) AddTransaction(tx block.Transaction) {
	bc.PendingTransactions = append(bc.PendingTransactions, tx)
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

func (bc *Blockchain) SaveToFile(filename string) error {
	data, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func LoadFromFile(filename string) (*Blockchain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var bc Blockchain
	err = json.Unmarshal(data, &bc)
	if err != nil {
		return nil, err
	}

	return &bc, nil
}
