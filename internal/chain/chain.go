package chain

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/miner"
	"toy-blockchain/internal/wallet"
	"unicode/utf8"
)

// MiningBenchmarkResult captures the cost of mining a block at a specific difficulty.
type MiningBenchmarkResult struct {
	Difficulty  int
	TimeTaken   time.Duration
	HashesTried int64
}

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
	bc.populateMerkleRoots()
}

// new new new

func (bc *Blockchain) populateMerkleRoots() {
	for _, b := range bc.Blocks {
		if b == nil {
			continue
		}
		b.MerkleRootValue = b.MerkleRoot()
	}
}

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
	minedBlock, _, err := bc.mineBlockConcurrent(newBlock, targetPrefix, workerCount)
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

// BenchmarkMining measures how mining effort scales with difficulty.
// It mines a simple block template at each requested difficulty and records the elapsed time and
// number of hash attempts required to find a valid nonce.
func (bc *Blockchain) BenchmarkMining(difficulties []int, maxBlockSize int, timeout time.Duration) []MiningBenchmarkResult {
	if len(difficulties) == 0 {
		return nil
	}

	results := make([]MiningBenchmarkResult, 0, len(difficulties))
	for _, difficulty := range difficulties {
		if difficulty < 1 {
			difficulty = 1
		}

		latestBlock := bc.GetLatestBlock()
		template := &block.Block{
			Index:        latestBlock.Index + 1,
			Timestamp:    time.Now().Unix(),
			Transactions: []block.Transaction{},
			PreviousHash: latestBlock.Hash,
			Nonce:        0,
			Difficulty:   difficulty,
		}

		start := time.Now()
		mined, attempts, err := bc.mineBlockConcurrent(template, strings.Repeat("0", difficulty), bc.effectiveMinerWorkers())
		elapsed := time.Since(start)
		// Always record attempts even if mining timed out or failed.
		if err != nil || mined == nil {
			results = append(results, MiningBenchmarkResult{Difficulty: difficulty, TimeTaken: elapsed, HashesTried: attempts})
			continue
		}

		results = append(results, MiningBenchmarkResult{Difficulty: difficulty, TimeTaken: elapsed, HashesTried: attempts})
	}
	return results
}

func (bc *Blockchain) effectiveMinerWorkers() int {
	if bc.MinerWorkers > 0 {
		return bc.MinerWorkers
	}
	return determineMiningWorkers()
}

func FormatMiningBenchmarkTable(results []MiningBenchmarkResult) string {
	if len(results) == 0 {
		return "No benchmark results"
	}
	// Prepare string representations
	headers := []string{"Difficulty", "Time Taken", "Hashes Tried"}
	rows := make([][3]string, len(results))
	for i, r := range results {
		rows[i][0] = fmt.Sprintf("%d", r.Difficulty)
		rows[i][1] = r.TimeTaken.String()
		rows[i][2] = fmt.Sprintf("%d", r.HashesTried)
	}

	// compute column widths
	colWidths := make([]int, 3)
	for i, h := range headers {
		colWidths[i] = utf8.RuneCountInString(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			rc := utf8.RuneCountInString(cell)
			if rc > colWidths[i] {
				colWidths[i] = rc
			}
		}
	}

	// helpers
	padCenter := func(s string, width int) string {
		rc := utf8.RuneCountInString(s)
		if rc >= width {
			return s
		}
		total := width - rc
		left := total / 2
		right := total - left
		return strings.Repeat(" ", left) + s + strings.Repeat(" ", right)
	}
	// padRight is intentionally omitted (not used)
	padLeft := func(s string, width int) string {
		rc := utf8.RuneCountInString(s)
		if rc >= width {
			return s
		}
		return strings.Repeat(" ", width-rc) + s
	}

	var b strings.Builder

	// header
	b.WriteString("| ")
	b.WriteString(padCenter(headers[0], colWidths[0]))
	b.WriteString(" | ")
	b.WriteString(padCenter(headers[1], colWidths[1]))
	b.WriteString(" | ")
	b.WriteString(padCenter(headers[2], colWidths[2]))
	b.WriteString(" |\n")

	// separator
	b.WriteString("|-")
	b.WriteString(strings.Repeat("-", colWidths[0]))
	b.WriteString("-|-" + strings.Repeat("-", colWidths[1]) + "-|-" + strings.Repeat("-", colWidths[2]) + "-|\n")

	// rows: center difficulty and time, right-align hashes
	for _, row := range rows {
		b.WriteString("| ")
		b.WriteString(padCenter(row[0], colWidths[0]))
		b.WriteString(" | ")
		b.WriteString(padCenter(row[1], colWidths[1]))
		b.WriteString(" | ")
		b.WriteString(padLeft(row[2], colWidths[2]))
		b.WriteString(" |\n")
	}

	return b.String()
}

func (bc *Blockchain) mineBlockConcurrent(template *block.Block, targetPrefix string, workers int) (*block.Block, int64, error) {
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
	var attempts int64

	closeResult := func(res result) {
		results <- res
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
				atomic.AddInt64(&attempts, 1)
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
			return nil, attempts, res.err
		}
		return res.block, attempts, nil
	case <-ctx.Done():
		// if a worker just found a result around the same time as ctx.Done,
		// try to read it before returning to avoid missing a valid result.
		select {
		case res := <-results:
			if res.err != nil {
				return nil, attempts, res.err
			}
			return res.block, attempts, nil
		default:
		}

		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, attempts, fmt.Errorf("mining timed out: %w", ctx.Err())
		}

		return nil, attempts, fmt.Errorf("mining cancelled")
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
