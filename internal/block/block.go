package block

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"toy-blockchain/internal/wallet"
	"toy-blockchain/internal/merkle"
)

type Transaction struct {
	Sender    string `json:"sender"`
	Recipient string `json:"recipient"`
	Amount    int64  `json:"amount"`
	Fee       int64  `json:"fee"`
	Nonce     int64  `json:"nonce"`
	PublicKey string `json:"public_key"`
	Signature string `json:"signature"`
}

type Block struct {
	Index           int           `json:"index"`
	Timestamp       int64         `json:"timestamp"`
	Transactions    []Transaction `json:"transactions"`
	PreviousHash    string        `json:"previous_hash"`
	Nonce           int           `json:"nonce"`
	MerkleRootValue string        `json:"merkle_root,omitempty"`
	Difficulty      int           `json:"difficulty"`
	Hash            string        `json:"hash,omitempty"`
	IsImmutable     bool          `json:"is_immutable"`
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
	return fmt.Sprintf("%s:%s:%d:%d:%d", tx.Sender, tx.Recipient, tx.Amount, tx.Fee, tx.Nonce)
}

func (tx *Transaction) VerifySignature() bool {
	if tx.PublicKey == "" || tx.Signature == "" {
		return false
	}
	return wallet.Verify(tx.PublicKey, tx.Payload(), tx.Signature)
}

func (b *Block) MerkleRoot() string {
	leaves := make([][]byte, len(b.Transactions))
	for i, tx := range b.Transactions {
		leaves[i] = []byte(tx.Payload())
	}
	return merkle.CalculateMerkleRoot(leaves)
}

func (b *Block) MerkleProofForTransaction(txIndex int) ([]string, error) {
	if txIndex < 0 || txIndex >= len(b.Transactions) {
		return nil, fmt.Errorf("transaction index %d out of range for block with %d transactions", txIndex, len(b.Transactions))
	}
	leaves := make([][]byte, len(b.Transactions))
	for i, tx := range b.Transactions {
		leaves[i] = []byte(tx.Payload())
	}
	return merkle.GenerateProof(leaves, txIndex)
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

	// Ensure merkle root is populated for persistence
	genesisBlock.MerkleRootValue = genesisBlock.MerkleRoot()
	genesisBlock.Hash = genesisBlock.CalculateHash()
	return genesisBlock
}
