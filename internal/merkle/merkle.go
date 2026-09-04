package merkle

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
)

func hashBytes(value []byte) []byte {
	sum := sha256.Sum256(value)
	return append([]byte(nil), sum[:]...)
}

func hashPair(left, right []byte) []byte {
	combined := append(append([]byte(nil), left...), right...)
	sum := sha256.Sum256(combined)
	return append([]byte(nil), sum[:]...)
}

func CalculateMerkleRoot(leaves [][]byte) string {
	if len(leaves) == 0 {
		emptyHash := sha256.Sum256([]byte{})
		return fmt.Sprintf("%x", emptyHash)
	}

	hashes := make([][]byte, len(leaves))
	for i, leaf := range leaves {
		hashes[i] = hashBytes(leaf)
	}

	for len(hashes) > 1 {
		nextLevel := make([][]byte, 0, (len(hashes)+1)/2)
		for i := 0; i < len(hashes); i += 2 {
			left := hashes[i]
			right := left

			if i+1 < len(hashes) {
				right = hashes[i+1]
			}

			nextLevel = append(nextLevel, hashPair(left, right))
		}
		hashes = nextLevel
	}

	return fmt.Sprintf("%x", hashes[0])
}

func GenerateProof(leaves [][]byte, index int) ([]string, error) {
	if len(leaves) == 0 {
		return nil, errors.New("cannot generate proof for empty leaf set")
	}
	if index < 0 || index >= len(leaves) {
		return nil, errors.New("transaction index out of range")
	}
	if len(leaves) == 1 {
		return []string{}, nil
	}

	level := make([][]byte, len(leaves))
	for i, leaf := range leaves {
		level[i] = hashBytes(leaf)
	}

	proof := make([]string, 0)
	idx := index
	for len(level) > 1 {
		if idx%2 == 0 {
			if idx+1 < len(level) {
				proof = append(proof, fmt.Sprintf("%x", level[idx+1]))
			} else {
				proof = append(proof, fmt.Sprintf("%x", level[idx]))
			}
		} else {
			proof = append(proof, fmt.Sprintf("%x", level[idx-1]))
		}

		nextLevel := make([][]byte, 0, (len(level)+1)/2)
		for i := 0; i < len(level); i += 2 {
			left := level[i]
			right := left
			if i+1 < len(level) {
				right = level[i+1]
			}
			nextLevel = append(nextLevel, hashPair(left, right))
		}
		level = nextLevel
		idx = idx / 2
	}

	return proof, nil
}

func VerifyProof(leaf []byte, proof []string, root string, index int) bool {
	if root == "" {
		return false
	}
	current := hashBytes(leaf)
	idx := index
	for _, siblingHex := range proof {
		sibling, err := hex.DecodeString(siblingHex)
		if err != nil {
			return false
		}
		if idx%2 == 0 {
			current = hashPair(current, sibling)
		} else {
			current = hashPair(sibling, current)
		}
		idx /= 2
	}
	return hex.EncodeToString(current) == root
}
