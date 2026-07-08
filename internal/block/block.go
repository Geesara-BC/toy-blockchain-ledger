package block

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

type Transaction struct {
	Sender    string  `json:"sender"`
	Recipient string  `json:"recipient"`
	Amount    float64 `json:"amount"`
}

type Block struct {
	Index        int           `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	PreviousHash string        `json:"previous_hash"`
	Nonce        int           `json:"nonce"`
	Hash         string        `json:"hash"`
}

func (b *Block) CalculateHash() string {
	oldHash := b.Hash
	b.Hash = ""

	blockData, err := json.Marshal(b)
	if err != nil {
		return ""
	}

	b.Hash = oldHash

	hashBytes := sha256.Sum256(blockData)
	return fmt.Sprintf("%x", hashBytes)
}

func NewGenesisBlock() *Block {
	genesisBlock := &Block{
		Index:        0,
		Timestamp:    time.Now().Unix(),
		Transactions: []Transaction{},
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
		Nonce:        0,
	}

	genesisBlock.Hash = genesisBlock.CalculateHash()
	return genesisBlock
}
