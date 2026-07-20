package chain

const (
	DefaultDifficultyRetargetInterval = 10
	DefaultExpectedBlockTimeSeconds   = 10
)

type DifficultyConfig struct {
	RetargetInterval         int
	ExpectedBlockTimeSeconds int
}

func NewDifficultyConfig() DifficultyConfig {
	return DifficultyConfig{
		RetargetInterval:         DefaultDifficultyRetargetInterval,
		ExpectedBlockTimeSeconds: DefaultExpectedBlockTimeSeconds,
	}
}

func (bc *Blockchain) initialDifficulty() int {
	if bc.BaseDifficulty > 0 {
		return bc.BaseDifficulty
	}
	if bc.Difficulty > 0 {
		return bc.Difficulty
	}
	return 1
}

func (bc *Blockchain) currentDifficulty() int {
	if bc.Difficulty > 0 {
		return bc.Difficulty
	}
	return bc.initialDifficulty()
}

func (bc *Blockchain) calculateNextDifficulty() int {
	if len(bc.Blocks) <= 1 {
		return bc.initialDifficulty()
	}

	retargetInterval := bc.DifficultyConfig.RetargetInterval
	if retargetInterval <= 0 {
		retargetInterval = DefaultDifficultyRetargetInterval
	}
	if (len(bc.Blocks)-1)%retargetInterval != 0 {
		return bc.currentDifficulty()
	}

	startIndex := len(bc.Blocks) - retargetInterval
	endIndex := len(bc.Blocks) - 1
	if startIndex < 1 || endIndex < startIndex {
		return bc.currentDifficulty()
	}

	return bc.calculateDifficultyFromWindow(startIndex, endIndex)
}

func (bc *Blockchain) calculateDifficultyFromWindow(startIndex, endIndex int) int {
	if startIndex < 1 || endIndex < startIndex {
		return bc.currentDifficulty()
	}

	firstBlock := bc.Blocks[startIndex]
	lastBlock := bc.Blocks[endIndex]
	if firstBlock == nil || lastBlock == nil {
		return bc.currentDifficulty()
	}

	if firstBlock.Timestamp == 0 || lastBlock.Timestamp <= firstBlock.Timestamp {
		return bc.currentDifficulty()
	}

	actualTime := lastBlock.Timestamp - firstBlock.Timestamp
	retargetInterval := bc.DifficultyConfig.RetargetInterval
	if retargetInterval <= 0 {
		retargetInterval = DefaultDifficultyRetargetInterval
	}
	expectedBlockTime := bc.DifficultyConfig.ExpectedBlockTimeSeconds
	if expectedBlockTime <= 0 {
		expectedBlockTime = DefaultExpectedBlockTimeSeconds
	}
	targetTime := int64(retargetInterval * expectedBlockTime)

	switch {
	case actualTime < targetTime/2:
		return bc.currentDifficulty() + 1
	case actualTime > targetTime*2:
		adjusted := bc.currentDifficulty() - 1
		if adjusted < 1 {
			return 1
		}
		return adjusted
	default:
		return bc.currentDifficulty()
	}
}

func (bc *Blockchain) expectedDifficultyForBlock(index int) int {
	if index <= 0 {
		return 0
	}
	if index == 1 {
		return bc.initialDifficulty()
	}
	retargetInterval := bc.DifficultyConfig.RetargetInterval
	if retargetInterval <= 0 {
		retargetInterval = DefaultDifficultyRetargetInterval
	}
	if index%retargetInterval == 1 {
		return bc.calculateDifficultyFromWindow(index-retargetInterval, index-1)
	}
	if index-1 < len(bc.Blocks) {
		return bc.Blocks[index-1].Difficulty
	}
	return bc.currentDifficulty()
}
