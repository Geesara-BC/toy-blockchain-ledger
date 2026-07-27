# Toy Blockchain and Ledger Simulator

A small blockchain and ledger built from scratch in Go, with proof-of-work mining,
transaction signatures, a Merkle root, difficulty retargeting, and fork resolution.

## Requirements

- Go 1.22 or newer
- No external services, databases, or network connection are needed. The program
  runs entirely on your own machine.

> **Note:** `go.mod` in this repository currently specifies `go 1.26.4`. If your
> local Go toolchain is an earlier version and you don't have network access to
> fetch a newer one, edit the `go` line in `go.mod` down to your installed version
> (1.22 or later) before building.

## Build and Run

Clone or unzip the project, then from the project root:

```bash
# Run all tests
go test ./...

# Run the program directly
go run ./cmd/toy-blockchain

# Or build a binary first
go build -o toy-blockchain ./cmd/toy-blockchain
./toy-blockchain
```

The program starts an interactive menu. Just follow the on-screen options — no
extra setup is required. On first run it creates a `data/` folder next to the
binary to store the chain and wallet files.

### Command-line flags

These can be passed when starting the program, for example:
`go run ./cmd/toy-blockchain --difficulty 4 --max-block-size 3`

| Flag | Default | What it does |
|---|---|---|
| `--db-path` | `data/blockchain.json` | Where the chain is saved and loaded from |
| `--difficulty` | `3` | Starting mining difficulty (number of leading zero hex digits required) |
| `--max-block-size` | `5` | Maximum number of transactions per mined block |
| `--mining-workers` | `4` | Number of goroutines used to search for a valid nonce in parallel |
| `--difficulty-retarget-interval` | (built-in default) | How often (in blocks) the difficulty is automatically adjusted |
| `--expected-block-time` | (built-in default) | Target number of seconds per block, used by difficulty retargeting |
| `--benchmark` | `false` | Runs the mining benchmark (see below) instead of the normal menu, then exits |
| `--benchmark-max-difficulty` | `4` | Highest difficulty level to include when `--benchmark` is used |

Example — run the difficulty-vs-effort experiment used in the research report:

```bash
go run ./cmd/toy-blockchain --benchmark --benchmark-max-difficulty 8 --mining-workers 32
```

## Command-Line Commands (in-app menu)

Once the program is running, you'll see a numbered menu:

| Option | What it does |
|---|---|
| `[1]` | Create a new wallet (generates a key pair and saves it to the address book) |
| `[2]` | Request free coins from the faucet into the pending pool |
| `[3]` | Send coins (the transaction is automatically signed with your saved key) |
| `[4]` | Mine a block from the pending transaction pool |
| `[5]` | View everyone's account balances |
| `[6]` | Print the full blockchain history |
| `[7]` | Validate the blockchain and check its integrity |
| `[8]` | View the pending transaction pool |
| `[9]` | Save the chain to disk and exit |

## Design Decisions

- **Hashing.** Each block's hash is a single SHA-256 over a fixed set of fields —
  index, timestamp, Merkle root, previous hash, nonce, and difficulty — serialised
  with `encoding/json`. The block's own hash field is left out of this calculation.
- **Merkle root instead of raw transactions.** Transactions aren't hashed directly
  into the block. Each transaction is hashed on its own, and those hashes are
  combined pairwise into a single Merkle root, which is what actually goes into the
  block hash. This keeps the block header a fixed size and still means changing any
  one transaction changes the block's hash.
- **Digital signatures.** Every transaction carries a public key and a signature
  (Ed25519, from Go's standard library). A transaction is rejected during
  validation if its signature doesn't match its claimed sender, so nobody can
  forge a transaction on someone else's behalf.
- **Proof-of-work.** Mining searches for a nonce so the block hash starts with the
  required number of zero hex digits. The search runs across multiple goroutines
  (`--mining-workers`) and stops as soon as any one of them finds a valid nonce.
- **Difficulty retargeting.** Difficulty is automatically adjusted every few blocks
  to keep average block time close to a target, rather than staying fixed forever.
- **Fork resolution.** If two competing chains are ever compared, the longest
  chain that is still fully valid is accepted.
- **Persistence.** The chain and wallet data are saved as JSON files and reloaded
  automatically the next time the program starts, so progress isn't lost between
  runs.

## Known Limitations

- **Single process only.** This program does not connect to other computers or
  peers. There is no real network consensus — only one local copy of the chain
  that this program fully controls. The fork-resolution logic exists but is only
  exercised locally, not against real independent peers.
- **Wallet keys are stored in plain text.** Private keys are saved directly in the
  wallet JSON file on disk with no password or encryption. A production wallet
  would protect keys with encryption or dedicated secure storage.
- **Interactive menu only.** The CLI is a numbered menu rather than single-shot
  subcommands (e.g. there's no `toy-blockchain validate` you can run directly from
  a script) — you drive it by typing a number and pressing enter each time.
- **Benchmark near the mining timeout.** At high difficulty levels, mining can
  approach the built-in mining timeout before it succeeds. In that case the
  recorded time/hash count reflects effort spent up to the timeout rather than a
  guaranteed successful mine.
