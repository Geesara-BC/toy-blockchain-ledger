package merkle

import (
	"crypto/sha256"
	"fmt"
)

func CalculateMerkleRoot(leaves [][]byte) string {
	if len(leaves) == 0 {
		emptyHash := sha256.Sum256([]byte{})
		return fmt.Sprintf("%x", emptyHash)
	}

	hashes := make([][]byte, len(leaves))
	for i, leaf := range leaves {
		h := sha256.Sum256(leaf)
		hashes[i] = append([]byte(nil), h[:]...)
	}

	for len(hashes) > 1 {
		nextLevel := make([][]byte, 0, (len(hashes)+1)/2)
		for i := 0; i < len(hashes); i += 2 {
			left := hashes[i]
			right := left

			if i+1 < len(hashes) {
				right = hashes[i+1]
			}

			combined := append(append([]byte(nil), left...), right...)
			parentHash := sha256.Sum256(combined)
			nextLevel = append(nextLevel, append([]byte(nil), parentHash[:]...))
		}
		hashes = nextLevel
	}

	return fmt.Sprintf("%x", hashes[0])
}
