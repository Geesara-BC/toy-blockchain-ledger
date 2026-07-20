package merkle

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

func TestCalculateMerkleRootEmptyLeaves(t *testing.T) {
	expected := fmt.Sprintf("%x", sha256.Sum256([]byte{}))
	got := CalculateMerkleRoot([][]byte{})

	if got != expected {
		t.Fatalf("expected empty merkle root %s, got %s", expected, got)
	}
}

func TestCalculateMerkleRootSingleLeaf(t *testing.T) {
	leaf := []byte("hello")
	expected := fmt.Sprintf("%x", sha256.Sum256(leaf))
	got := CalculateMerkleRoot([][]byte{leaf})

	if got != expected {
		t.Fatalf("expected single-leaf merkle root %s, got %s", expected, got)
	}
}

func TestCalculateMerkleRootEvenLeaves(t *testing.T) {
	leaf1 := []byte("a")
	leaf2 := []byte("b")

	hash1 := sha256.Sum256(leaf1)
	hash2 := sha256.Sum256(leaf2)
	parent := sha256.Sum256(append(append([]byte(nil), hash1[:]...), hash2[:]...))
	expected := fmt.Sprintf("%x", parent)

	got := CalculateMerkleRoot([][]byte{leaf1, leaf2})

	if got != expected {
		t.Fatalf("expected merkle root %s, got %s", expected, got)
	}
}

func TestCalculateMerkleRootOddLeaves(t *testing.T) {
	leaf1 := []byte("a")
	leaf2 := []byte("b")
	leaf3 := []byte("c")

	hash1 := sha256.Sum256(leaf1)
	hash2 := sha256.Sum256(leaf2)
	hash3 := sha256.Sum256(leaf3)

	parent12 := sha256.Sum256(append(append([]byte(nil), hash1[:]...), hash2[:]...))
	parent33 := sha256.Sum256(append(append([]byte(nil), hash3[:]...), hash3[:]...))
	root := sha256.Sum256(append(append([]byte(nil), parent12[:]...), parent33[:]...))
	expected := fmt.Sprintf("%x", root)

	got := CalculateMerkleRoot([][]byte{leaf1, leaf2, leaf3})

	if got != expected {
		t.Fatalf("expected merkle root %s, got %s", expected, got)
	}
}
