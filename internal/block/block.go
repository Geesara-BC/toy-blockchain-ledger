package block

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"toy-blockchain/internal/merkle"
)

type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	Fee       int64  `json:"fee"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

type Block struct {
	Index        int           `json:"index"`
	Timestamp    int64         `json:"timestamp"`
	Transactions []Transaction `json:"transactions"`
	PreviousHash string        `json:"previous_hash"`
	Nonce        int           `json:"nonce"`
	Difficulty   int           `json:"difficulty"`
	Hash         string        `json:"hash,omitempty"`
	IsImmutable  bool          `json:"is_immutable"`
}

type blockForHash struct {
	Index     int   `json:"index"`
	Timestamp int64 `json:"timestamp"`
	// Transactions []Transaction `json:"transactions"`
	MerkleRoot   string `json:"merkle_root"`
	PreviousHash string `json:"previous_hash"`
	Nonce        int    `json:"nonce"`
	Difficulty   int    `json:"difficulty"`
}

func (tx *Transaction) Payload() string {
	return fmt.Sprintf("%s:%s:%d", tx.Sender, tx.Recipient, tx.Amount)
}

func (b *Block) MerkleRoot() string {
	leaves := make([][]byte, len(b.Transactions))
	for i, tx := range b.Transactions {
		leaves[i] = []byte(tx.Payload())
	}
	return merkle.CalculateMerkleRoot(leaves)
}

func (b *Block) CalculateHash() string {

	blockHash := blockForHash{
		Index:        b.Index,
		Timestamp:    b.Timestamp,
		MerkleRoot:   b.MerkleRoot(),
		PreviousHash: b.PreviousHash,
		Nonce:        b.Nonce,
		Difficulty:   b.Difficulty,
	}

	blockData, err := json.Marshal(blockHash)
	if err != nil {
		return ""
	}

	hashBytes := sha256.Sum256(blockData)
	return fmt.Sprintf("%x", hashBytes)
}

func NewGenesisBlock() *Block {
	genesisBlock := &Block{
		Index:        0,
		Timestamp:    0,
		Transactions: []Transaction{},
		PreviousHash: "0000000000000000000000000000000000000000000000000000000000000000",
		Difficulty:   0,
		Nonce:        0,
		IsImmutable:  true,
	}

	genesisBlock.Hash = genesisBlock.CalculateHash()
	return genesisBlock
}
