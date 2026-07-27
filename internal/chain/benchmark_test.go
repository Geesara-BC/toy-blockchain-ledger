package chain

import (
	"strings"
	"testing"
	"time"
)

func TestBenchmarkMiningProducesMeasuredResults(t *testing.T) {
	bc := NewBlockchain(1)
	bc.MinerWorkers = 1
	bc.MinerTimeout = 2 * time.Second

	results := bc.BenchmarkMining([]int{1, 2}, 1, 2*time.Second)
	if len(results) != 2 {
		t.Fatalf("expected 2 benchmark results, got %d", len(results))
	}

	for _, result := range results {
		if result.HashesTried == 0 {
			t.Fatalf("expected positive hash count for difficulty %d", result.Difficulty)
		}
		if result.TimeTaken < 0 {
			t.Fatalf("expected non-negative timing for difficulty %d", result.Difficulty)
		}
	}

	if results[1].HashesTried <= results[0].HashesTried {
		t.Fatalf("expected higher difficulty to require more hashes: %d <= %d", results[1].HashesTried, results[0].HashesTried)
	}

	formatted := FormatMiningBenchmarkTable(results)
	if !strings.Contains(formatted, "Difficulty") || !strings.Contains(formatted, "Hashes Tried") {
		t.Fatalf("expected formatted benchmark table to include headers, got %q", formatted)
	}
}
