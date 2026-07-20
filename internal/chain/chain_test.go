package chain

import (
	"errors"
	"strings"
	"testing"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/wallet"
)

func TestGenesisBlockInChain(t *testing.T) {
	difficulty := 2
	bc := NewBlockchain(difficulty)

	if len(bc.GetBlocks()) != 1 {
		t.Fatalf("Expected chain to have exactly 1 block, got %d", len(bc.GetBlocks()))
	}

	genesisBlock := bc.GetBlocks()[0]
	if genesisBlock.Index != 0 {
		t.Errorf("Expected genesis block index to be 0, got %d", genesisBlock.Index)
	}

	expectedPrevHash := "0000000000000000000000000000000000000000000000000000000000000000"
	if genesisBlock.PreviousHash != expectedPrevHash {
		t.Errorf("Expected previous hash %s, got %s", expectedPrevHash, genesisBlock.PreviousHash)
	}
}

func TestMinedBlockDifficulty(t *testing.T) {
	difficulty := 2
	bc := NewBlockchain(difficulty)
	bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})

	newBlock, err := bc.MinePendingTransactions(5)
	if err != nil {
		t.Fatalf("Mining failed: %v", err)
	}

	targetPrefix := strings.Repeat("0", difficulty)
	if !strings.HasPrefix(newBlock.Hash, targetPrefix) {
		t.Errorf("Block hash does not meet difficulty. Expected prefix '%s', got hash %s", targetPrefix, newBlock.Hash)
	}

	recalculatedHash := newBlock.CalculateHash()
	if recalculatedHash != newBlock.Hash {
		t.Errorf("Nonce does not reproduce the hash. Expected %s, got %s", newBlock.Hash, recalculatedHash)
	}
}

func TestHonestChainValidation(t *testing.T) {
	bc := NewBlockchain(3)

	err1 := bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 50})
	if err1 != nil {
		t.Fatalf("Faucet transaction failed: %v", err1)
	}
	bc.MinePendingTransactions(5)

	tx := block.Transaction{Sender: "Alice", Recipient: "Bob", Amount: 10}
	err2 := bc.AddTransaction(tx)

	expectedErr := "UNAUTHORIZED: Transaction must be signed with a valid public key and signature"

	if err2 == nil {
		t.Errorf("Test failed: An unsigned transaction was incorrectly accepted!")
	} else if err2.Error() != expectedErr {
		t.Errorf("Test failed: Expected error '%s', but got: %v", expectedErr, err2)
	}

	isValid, _ := bc.IsValid()
	if !isValid {
		t.Errorf("Test failed: Chain should be valid even after a rejected transaction attempt")
	}
}
func TestTamperedChainDetection(t *testing.T) {
	bc := NewBlockchain(3)

	bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100})
	bc.MinePendingTransactions(5)

	priv, pub, _ := wallet.GenerateKeyPair()
	tx := block.Transaction{Sender: pub, Recipient: "Bob", Amount: 20, PublicKey: pub}
	sig, _ := wallet.Sign(priv, tx.Payload())
	tx.Signature = sig

	bc.AddTransaction(tx)
	bc.MinePendingTransactions(5)

	// Use TamperBlockForTesting to modify block for testing immutability detection
	bc.TamperBlockForTesting(1, func(b *block.Block) {
		b.Transactions[0].Amount = 500
	})

	isValid, faultyIndex := bc.IsValid()

	if isValid {
		t.Errorf("Security Test Failed: Tampered chain was incorrectly marked as valid!")
	}

	if faultyIndex != 1 {
		t.Errorf("Security Test Failed: Expected failure at block index 1, but got %d", faultyIndex)
	}
}

func TestMineBlockConcurrentFindsValidHash(t *testing.T) {
	bc := NewBlockchain(2)
	if err := bc.AddTransaction(block.Transaction{Sender: "FAUCET", Recipient: "Alice", Amount: 100}); err != nil {
		t.Fatalf("faucet transaction failed: %v", err)
	}

	latest := bc.GetLatestBlock()
	template := &block.Block{
		Index:        latest.Index + 1,
		Timestamp:    1,
		Transactions: bc.PendingTransactions,
		PreviousHash: latest.Hash,
		Difficulty:   bc.Difficulty,
	}

	targetPrefix := strings.Repeat("0", bc.Difficulty)
	mined, err := bc.mineBlockConcurrent(template, targetPrefix, 4)
	if err != nil {
		t.Fatalf("concurrent mining failed: %v", err)
	}

	if !strings.HasPrefix(mined.Hash, targetPrefix) {
		t.Fatalf("expected hash with prefix %q, got %s", targetPrefix, mined.Hash)
	}
	if mined.CalculateHash() != mined.Hash {
		t.Fatalf("mined hash does not reproduce: expected %s, got %s", mined.Hash, mined.CalculateHash())
	}
}

func TestRetargetsDifficultyWhenBlocksAreMinedTooQuickly(t *testing.T) {
	bc := NewBlockchain(2)
	base := block.NewGenesisBlock()
	bc.Blocks = []*block.Block{base}

	for i := 1; i <= DefaultDifficultyRetargetInterval; i++ {
		bc.Blocks = append(bc.Blocks, &block.Block{
			Index:        i,
			Timestamp:    int64(100 + i),
			Difficulty:   2,
			PreviousHash: bc.GetLatestBlock().Hash,
			Hash:         "placeholder",
		})
	}

	nextDifficulty := bc.calculateNextDifficulty()
	if nextDifficulty <= 2 {
		t.Fatalf("expected difficulty to increase when blocks are mined too quickly, got %d", nextDifficulty)
	}
}

func makeValidBlock(parent *block.Block, index int, timestamp int64, difficulty int) *block.Block {
	targetPrefix := strings.Repeat("0", difficulty)
	template := &block.Block{
		Index:        index,
		Timestamp:    timestamp,
		Transactions: []block.Transaction{},
		PreviousHash: parent.Hash,
		Difficulty:   difficulty,
	}

	for nonce := 0; ; nonce++ {
		template.Nonce = nonce
		template.Hash = template.CalculateHash()
		if strings.HasPrefix(template.Hash, targetPrefix) {
			return template
		}
	}
}

func TestReplaceWithLongestChainAdoptsLongerValidFork(t *testing.T) {
	mainChain := NewBlockchain(2)
	mainChain.Blocks = []*block.Block{block.NewGenesisBlock()}

	for i := 1; i <= 2; i++ {
		mainChain.Blocks = append(mainChain.Blocks, makeValidBlock(mainChain.GetLatestBlock(), i, int64(100+i), 2))
	}

	forkChain := NewBlockchain(2)
	forkChain.Blocks = []*block.Block{block.NewGenesisBlock()}
	for i := 1; i <= 3; i++ {
		forkChain.Blocks = append(forkChain.Blocks, makeValidBlock(forkChain.GetLatestBlock(), i, int64(200+i), 2))
	}

	if err := mainChain.ReplaceWithLongestChain(forkChain); err != nil {
		t.Fatalf("expected longer valid fork to be accepted: %v", err)
	}
	if len(mainChain.Blocks) != 4 {
		t.Fatalf("expected chain length 4 after replacement, got %d", len(mainChain.Blocks))
	}
}

func TestReplaceWithLongestChainRejectsShorterOrInvalidFork(t *testing.T) {
	mainChain := NewBlockchain(2)
	mainChain.Blocks = []*block.Block{block.NewGenesisBlock()}
	mainChain.Blocks = append(mainChain.Blocks, makeValidBlock(mainChain.GetLatestBlock(), 1, 100, 2))

	shorterFork := NewBlockchain(2)
	shorterFork.Blocks = []*block.Block{block.NewGenesisBlock()}
	if err := mainChain.ReplaceWithLongestChain(shorterFork); !errors.Is(err, ErrShorterChain) {
		t.Fatalf("expected shorter chain error, got %v", err)
	}

	invalidFork := NewBlockchain(2)
	invalidFork.Blocks = []*block.Block{block.NewGenesisBlock()}
	invalidFork.Blocks = append(invalidFork.Blocks, makeValidBlock(block.NewGenesisBlock(), 1, 100, 2))

	invalidFork.Blocks[1].PreviousHash = "bad-hash"
	invalidFork.Blocks[1].Hash = invalidFork.Blocks[1].CalculateHash()
	if err := mainChain.ReplaceWithLongestChain(invalidFork); !errors.Is(err, ErrInvalidChain) && err == nil {
		t.Fatalf("expected invalid chain error, got %v", err)
	}
}
