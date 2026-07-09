package storage

import (
	"encoding/json"
	"os"
	"toy-blockchain/internal/chain"
)

func SaveToFile(filename string, bc *chain.Blockchain) error {
	data, err := json.MarshalIndent(bc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0644)
}

func LoadFromFile(filename string) (*chain.Blockchain, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var bc chain.Blockchain
	err = json.Unmarshal(data, &bc)
	if err != nil {
		return nil, err
	}

	return &bc, nil
}
