package chain

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/miner"
	"toy-blockchain/internal/wallet"
)

type Blockchain struct {
	Blocks                 []*block.Block      `json:"blocks"`
	Difficulty             int                 `json:"difficulty"`
	BaseDifficulty         int                 `json:"base_difficulty"`
	DifficultyConfig       DifficultyConfig    `json:"difficulty_config"`
	PendingTransactions    []block.Transaction `json:"pending_transactions"`
	RewardEngine           *miner.RewardEngine `json:"-"`
	MinerRegistry          *miner.Registry     `json:"-"`
	RewardAmount           int64               `json:"reward_amount"`
	FeePerTransaction      int64               `json:"fee_per_transaction"`
	RegisteredMinerAddress []string            `json:"registered_miners,omitempty"`
	MinerAddress           string              `json:"miner_address"`
	MinerWorkers           int                 `json:"miner_workers"`
	MinerTimeout           time.Duration       `json:"miner_timeout"`
}

func NewBlockchain(difficulty int) *Blockchain {
	return NewBlockchainWithDifficultyConfig(difficulty, NewDifficultyConfig())
}

func NewBlockchainWithDifficultyConfig(difficulty int, config DifficultyConfig) *Blockchain {
	bc := &Blockchain{
		Blocks:                 []*block.Block{block.NewGenesisBlock()},
		Difficulty:             difficulty,
		BaseDifficulty:         difficulty,
		DifficultyConfig:       config,
		PendingTransactions:    []block.Transaction{},
		RewardAmount:           50,
		FeePerTransaction:      0,
		RegisteredMinerAddress: []string{},
		MinerAddress:           "",
		MinerWorkers:           0,
		MinerTimeout:           10 * time.Second,
	}
	bc.RewardEngine = miner.NewRewardEngine(bc.RewardAmount, miner.NewFixedFeePolicy(bc.FeePerTransaction))
	bc.MinerRegistry = miner.NewRegistry()
	return bc
}

// new new new

func (bc *Blockchain) rehydrate() {
	if bc.RewardEngine == nil {
		bc.RewardEngine = miner.NewRewardEngine(bc.RewardAmount, miner.NewFixedFeePolicy(bc.FeePerTransaction))
	}
	if bc.MinerRegistry == nil {
		bc.MinerRegistry = miner.NewRegistry()
	}
	for _, address := range bc.RegisteredMinerAddress {
		if address == "" {
			continue
		}
		_ = bc.MinerRegistry.Register(address)
	}
	bc.RegisteredMinerAddress = bc.MinerRegistry.List()
	if bc.MinerAddress == "" && len(bc.RegisteredMinerAddress) > 0 {
		bc.MinerAddress = bc.RegisteredMinerAddress[0]
	}
}

func (bc *Blockchain) syncRegisteredMiners() {
	if bc.MinerRegistry == nil {
		return
	}
	bc.RegisteredMinerAddress = bc.MinerRegistry.List()
}

func (bc *Blockchain) RehydrateForLoad() {
	bc.rehydrate()
}

func (bc *Blockchain) PrepareForPersistence() {
	bc.rehydrate()
	if bc.RewardEngine != nil {
		bc.RewardAmount = bc.RewardEngine.BlockReward
		if feePolicy, ok := bc.RewardEngine.FeePolicy.(*miner.FixedFeePolicy); ok {
			bc.FeePerTransaction = feePolicy.FeePerTransaction
		}
	}
	bc.syncRegisteredMiners()
}

// new new new

// GetBlocks returns deep copies of blocks to prevent modification (true immutability)
func (bc *Blockchain) GetBlocks() []block.Block {
	copies := make([]block.Block, len(bc.Blocks))
	for i, b := range bc.Blocks {
		copies[i] = *b // Create deep copies
	}
	return copies
}

// TamperBlockForTesting allows modifying blocks (for testing only)
func (bc *Blockchain) TamperBlockForTesting(blockIndex int, modifyFunc func(*block.Block)) {
	if blockIndex < len(bc.Blocks) {
		modifyFunc(bc.Blocks[blockIndex])
	}
}

// ValidateBlockModification ensures blocks cannot be modified once committed
func (bc *Blockchain) ValidateBlockModification(blockIndex int) error {
	if blockIndex < len(bc.Blocks) && bc.Blocks[blockIndex].IsImmutable {
		return errors.New("FORBIDDEN: This block is immutable and cannot be modified")
	}
	return nil
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

// new

func (bc *Blockchain) RegisterMiner(address string) error {
	bc.rehydrate()
	if err := bc.MinerRegistry.Register(address); err != nil {
		return err
	}
	bc.MinerAddress = address
	bc.syncRegisteredMiners()
	return nil
}

func (bc *Blockchain) IsMinerRegistered(address string) bool {
	bc.rehydrate()
	if bc.MinerRegistry == nil {
		return false
	}
	return bc.MinerRegistry.IsRegistered(address)
}

func (bc *Blockchain) RegisteredMiners() []string {
	bc.rehydrate()
	if bc.MinerRegistry == nil {
		return []string{}
	}
	return bc.MinerRegistry.List()
}

//new

func (bc *Blockchain) MinePendingTransactions(maxBlockSize int) (*block.Block, error) {
	bc.rehydrate()
	if len(bc.PendingTransactions) == 0 {
		return nil, errors.New("no pending transactions to mine")
	}
	txCount := len(bc.PendingTransactions)
	if txCount > maxBlockSize {
		txCount = maxBlockSize
	}

	transactionsToMine := bc.PendingTransactions[:txCount]

	latestBlock := bc.GetLatestBlock()

	//new

	selectedMiner := bc.MinerAddress
	if selectedMiner == "" {
		registeredMiners := bc.RegisteredMiners()
		if len(registeredMiners) > 0 {
			selectedMiner = registeredMiners[0]
			bc.MinerAddress = selectedMiner
		}
	} else if !bc.IsMinerRegistered(selectedMiner) {
		registeredMiners := bc.RegisteredMiners()
		if len(registeredMiners) > 0 {
			return nil, errors.New("miner address is not registered")
		}
	}

	var coinbaseTx block.Transaction
	var reward miner.Reward
	if selectedMiner != "" {
		var err error
		coinbaseTx, reward, err = bc.RewardEngine.BuildCoinbaseTransaction(selectedMiner, transactionsToMine)
		if err != nil {
			return nil, err
		}
	}

	rewardedTransactions := append([]block.Transaction{}, transactionsToMine...)
	if selectedMiner != "" {
		rewardedTransactions = append(rewardedTransactions, coinbaseTx)
	}

	//new

	nextDifficulty := bc.calculateNextDifficulty()
	newBlock := &block.Block{
		Index:        latestBlock.Index + 1,
		Timestamp:    time.Now().Unix(),
		Transactions: rewardedTransactions,
		PreviousHash: latestBlock.Hash,
		Nonce:        0,
		Difficulty:   nextDifficulty,
	}
	targetPrefix := strings.Repeat("0", nextDifficulty)
	workerCount := bc.MinerWorkers
	if workerCount <= 0 {
		workerCount = determineMiningWorkers()
	}
	minedBlock, err := bc.mineBlockConcurrent(newBlock, targetPrefix, workerCount)
	if err != nil {
		return nil, err
	}

	minedBlock.Transactions = rewardedTransactions
	minedBlock.IsImmutable = true
	bc.Blocks = append(bc.Blocks, minedBlock)
	bc.PendingTransactions = bc.PendingTransactions[txCount:]
	_ = reward
	return minedBlock, nil
}

// new

func determineMiningWorkers() int {
	workers := runtime.NumCPU()
	if workers < 1 {
		return 1
	}
	return workers
}

func (bc *Blockchain) mineBlockConcurrent(template *block.Block, targetPrefix string, workers int) (*block.Block, error) {
	if workers <= 0 {
		workers = 1
	}

	type result struct {
		block *block.Block
		err   error
	}

	timeout := bc.MinerTimeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	results := make(chan result, 1)
	var once sync.Once

	closeResult := func(res result) {

		select {
		case results <- res:
		default:
		}
	}

	for workerID := 0; workerID < workers; workerID++ {
		go func(workerID int) {
			startNonce := workerID
			candidate := *template
			candidate.Nonce = startNonce
			candidate.Timestamp = template.Timestamp
			candidate.Transactions = append([]block.Transaction(nil), template.Transactions...)

			for {
				candidate.Hash = candidate.CalculateHash()
				if strings.HasPrefix(candidate.Hash, targetPrefix) {
					once.Do(func() {
						closeResult(result{block: &candidate})
						cancel()
					})
					return
				}

				select {
				case <-ctx.Done():
					return
				default:
				}

				candidate.Nonce += workers
			}
		}(workerID)
	}

	select {
	case res := <-results:
		if res.err != nil {
			return nil, res.err
		}
		return res.block, nil
	case <-ctx.Done():
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, fmt.Errorf("mining timed out: %w", ctx.Err())
		}

		return nil, fmt.Errorf("mining cancelled")
	}
}

// new

func (bc *Blockchain) IsValid() (bool, int) {

	if len(bc.Blocks) > 0 {
		genesisBlock := bc.Blocks[0]

		// expectedGenesisHash := "24f430256868bc93806e9ebd86a15ea50940b55fb99c8871e5119af4d9f72f36"
		expectedGenesisHash := block.NewGenesisBlock().Hash
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

		expectedDifficulty := bc.expectedDifficultyForBlock(current.Index)
		if current.Difficulty != expectedDifficulty {
			return false, current.Index
		}
		targetPrefix := strings.Repeat("0", current.Difficulty)

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
