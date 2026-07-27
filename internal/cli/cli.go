package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/ledger"
	"toy-blockchain/internal/storage"
	"toy-blockchain/internal/wallet"
)

const (
	Reset  = "\033[0m"
	Red    = "\033[31m"
	Green  = "\033[32m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
	Purple = "\033[35m"
	Cyan   = "\033[36m"
	White  = "\033[37m"
	Bold   = "\033[1m"
)

type CLI struct {
	dbFile                     string
	difficulty                 int
	maxBlockSize               int
	minerWorkers               int
	minerTimeout               time.Duration
	difficultyRetargetInterval int
	expectedBlockTimeSeconds   int
}

func NewCLI(dbFile string, difficulty int, maxBlockSize int, minerWorkers int, minerTimeout time.Duration, difficultyRetargetInterval int, expectedBlockTimeSeconds int) *CLI {
	return &CLI{
		dbFile:                     dbFile,
		difficulty:                 difficulty,
		maxBlockSize:               maxBlockSize,
		minerWorkers:               minerWorkers,
		minerTimeout:               minerTimeout,
		difficultyRetargetInterval: difficultyRetargetInterval,
		expectedBlockTimeSeconds:   expectedBlockTimeSeconds,
	}
}

func (cli *CLI) Run() {
	var bc *chain.Blockchain
	var err error

	if _, err = os.Stat(cli.dbFile); err == nil {
		bc, err = storage.LoadFromFile(cli.dbFile)
		if err != nil {
			fmt.Printf(Red+"Error loading blockchain: %v\n"+Reset, err)
			os.Exit(1)
		}
		if cli.minerWorkers > 0 {
			bc.MinerWorkers = cli.minerWorkers
		}
		if cli.minerTimeout > 0 {
			bc.MinerTimeout = cli.minerTimeout
		}
	} else {
		bc = chain.NewBlockchainWithDifficultyConfig(cli.difficulty, chain.DifficultyConfig{
			RetargetInterval:         cli.difficultyRetargetInterval,
			ExpectedBlockTimeSeconds: cli.expectedBlockTimeSeconds,
		})
		bc.MinerWorkers = cli.minerWorkers
		bc.MinerTimeout = cli.minerTimeout
		err = storage.SaveToFile(cli.dbFile, bc)
		if err != nil {
			fmt.Printf(Red+"Error initializing blockchain file: %v\n"+Reset, err)
			os.Exit(1)
		}
	}

	l := ledger.NewLedger(bc)

	addressBook, privateKeys := loadWallets(cli.dbFile)

	for {
		cli.printHeader()
		cli.printMenu(len(bc.PendingTransactions))

		fmt.Print(Bold + Cyan + "Choose an option (1-9): " + Reset)
		var choice int
		_, fmtScanErr := fmt.Scanln(&choice)

		if fmtScanErr != nil {
			fmt.Println(Red + "Invalid input! Please enter a number between 1 and 9.\n" + Reset)
			cli.waitForUser()
			continue
		}

		fmt.Println()

		switch choice {

		case 1:
			var name string
			fmt.Print("Enter a Name for this Wallet (e.g., Janindu): ")
			fmt.Scanln(&name)

			name = strings.TrimSpace(name)
			if name == "" {
				fmt.Println(Red + "Wallet name cannot be empty." + Reset)
				cli.waitForUser()
				continue
			}
			if hasWalletName(addressBook, name) {
				fmt.Printf(Red+"Wallet name '%s' already exists. Please choose a different name.\n"+Reset, name)
				cli.waitForUser()
				continue
			}

			fmt.Println(Yellow + "Generating a secure cryptographic key pair..." + Reset)
			privKey, pubKey, err := wallet.GenerateKeyPair()
			if err != nil {
				fmt.Println(Red + "Error generating keys!" + Reset)
				continue
			}
			normalizedName := normalizeWalletName(name)

			addressBook[normalizedName] = pubKey
			privateKeys[normalizedName] = privKey

			// new

			fmt.Print("Register this wallet as a miner? (y/n): ")
			var registerChoice string
			fmt.Scanln(&registerChoice)
			if strings.EqualFold(registerChoice, "y") || strings.EqualFold(registerChoice, "yes") {
				if err := bc.RegisterMiner(pubKey); err != nil {
					fmt.Printf(Red+"Failed to register miner: %v\n"+Reset, err)
				} else {
					fmt.Printf(Green+"SUCCESS! Wallet '%s' registered as a miner.\n"+Reset, name)
				}
			}

			// new

			saveWallets(addressBook, privateKeys, cli.dbFile)

			fmt.Printf(Green+"SUCCESS! Wallet created and securely saved for '%s'.\n"+Reset, name)
		case 2:
			var to string
			var amount int64

			fmt.Print("Enter Your Name / Account Name: ")
			fmt.Scanln(&to)
			fmt.Print("Enter Amount of Free Coins Needed: ")
			_, err := fmt.Scanln(&amount)

			pubKey, exists := addressBook[to]
			if !exists {
				fmt.Println(Red + "Error: Account name not found! Please Create a Wallet first (Option 1)." + Reset)
				cli.waitForUser()
				continue
			}

			if err != nil || amount <= 0 {
				fmt.Println(Red + "Error: Invalid amount!" + Reset)
				cli.waitForUser()
				continue
			}

			tx := block.Transaction{
				Sender:    "FAUCET",
				Recipient: pubKey,
				Amount:    amount,
				PublicKey: "SYSTEM_FAUCET_PUBLIC_KEY",
				Signature: "SYSTEM_AUTHORIZED_NO_SIGNATURE",
			}

			if err := bc.AddTransaction(tx); err != nil {
				fmt.Printf(Red+"TRANSACTION REJECTED: %v\n"+Reset, err)
				cli.waitForUser()
				continue
			}

			if err := storage.SaveToFile(cli.dbFile, bc); err != nil {
				fmt.Printf(Red+"Error saving blockchain: %v\n"+Reset, err)
				cli.waitForUser()
				continue
			}
			fmt.Println(Green + "SUCCESS: Faucet transaction added to the pending pool!" + Reset)

		case 3:
			var from, to string
			var amount int64
			var privKey string

			fmt.Print("Enter Sender Name (From): ")
			fmt.Scanln(&from)
			fmt.Print("Enter Recipient Name (To): ")
			fmt.Scanln(&to)
			fmt.Print("Enter Amount to Send: ")
			_, err := fmt.Scanln(&amount)

			// if from == "" || to == "" || err != nil || amount <= 0 {
			// 	fmt.Println(Red + "Error: Invalid accounts or amount!" + Reset)
			// 	cli.waitForUser()
			// 	continue
			// }

			// tx := block.Transaction{Sender: from, Recipient: to, Amount: amount}

			fmt.Print("Enter your Private Key to sign the transaction: ")
			fmt.Scanln(&privKey)

			pubKeyFrom, ok1 := resolveWalletName(addressBook, from)
			pubKeyTo, ok2 := resolveWalletName(addressBook, to)

			if !ok1 || !ok2 {
				fmt.Println(Red + "Error: Sender or Recipient not found in your local Address Book!" + Reset)
				cli.waitForUser()
				continue
			}
			if from == "" || to == "" || err != nil || amount <= 0 {
				fmt.Println(Red + "Error: Invalid accounts or amount!" + Reset)
				cli.waitForUser()
				continue
			}
			tx := block.Transaction{
				Sender:    pubKeyFrom,
				Recipient: pubKeyTo,
				Amount:    amount,
				PublicKey: pubKeyFrom,
			}

			signature, err := wallet.Sign(privKey, tx.Payload())
			if err != nil {
				fmt.Println(Red + "Error signing transaction!" + Reset)
				cli.waitForUser()
				continue
			}
			tx.Signature = signature

			if err := bc.AddTransaction(tx); err != nil {
				fmt.Printf(Red+"TRANSACTION REJECTED: %v\n"+Reset, err)
				cli.waitForUser()
				continue
			}

			if err := storage.SaveToFile(cli.dbFile, bc); err != nil {
				fmt.Printf(Red+"Error saving blockchain: %v\n"+Reset, err)
				cli.waitForUser()
				continue
			}
			fmt.Println(Green + "SUCCESS: Transaction added to the pending pool!" + Reset)

		case 4:

			// new

			if len(bc.RegisteredMiners()) == 0 {
				fmt.Println(Red + "No registered miners available. Create a wallet and register it as a miner first." + Reset)
				cli.waitForUser()
				continue
			}

			registeredMiners := bc.RegisteredMiners()

			pubToName := make(map[string]string)
			for name, pubKey := range addressBook {
				pubToName[pubKey] = name
			}

			fmt.Println(Yellow + "Select a registered miner wallet:" + Reset)
			for i, minerAddr := range registeredMiners {
				displayValue := minerAddr
				if name, exists := pubToName[minerAddr]; exists {
					displayValue = fmt.Sprintf("%s (%s)", name, minerAddr)
				}
				fmt.Printf("  %d. %s\n", i+1, displayValue)
			}
			fmt.Print("Enter the miner wallet name, address, or its index: ")
			var minerInput string
			fmt.Scanln(&minerInput)

			selectedMiner := ""
			if index := 0; len(minerInput) > 0 {
				if _, err := fmt.Sscanf(minerInput, "%d", &index); err == nil {
					if index > 0 && index <= len(registeredMiners) {
						selectedMiner = registeredMiners[index-1]
					}
				}
			}
			if selectedMiner == "" {
				selectedMiner = minerInput
			}
			if selectedMiner == "" {
				for _, minerAddr := range registeredMiners {
					if strings.EqualFold(minerInput, minerAddr) {
						selectedMiner = minerAddr
						break
					}
					if name, exists := pubToName[minerAddr]; exists && strings.EqualFold(minerInput, name) {
						selectedMiner = minerAddr
						break
					}
				}
			}

			if selectedMiner == "" || !bc.IsMinerRegistered(selectedMiner) {
				fmt.Println(Red + "Invalid miner wallet. Please choose a registered miner." + Reset)
				cli.waitForUser()
				continue
			}

			bc.MinerAddress = selectedMiner

			// new

			fmt.Println(Yellow + "Mining pending transactions into a new block... Please wait..." + Reset)
			startTime := time.Now()
			newBlock, err := bc.MinePendingTransactions(cli.maxBlockSize)
			duration := time.Since(startTime)

			if err != nil {
				fmt.Printf(Red+"Mining Failed: %v\n"+Reset, err)
				cli.waitForUser()
				continue
			}

			if err := storage.SaveToFile(cli.dbFile, bc); err != nil {
				fmt.Printf(Red+"Error saving blockchain: %v\n"+Reset, err)
				cli.waitForUser()
				continue
			}
			fmt.Println(Green + "SUCCESS: Block successfully mined and committed to the ledger!" + Reset)
			fmt.Printf("Time Taken: %v | Nonce: %d\n", duration, newBlock.Nonce)
			fmt.Printf("Block Hash: %s%s%s\n", Yellow, newBlock.Hash, Reset)

		case 5:
			balances := l.GetBalances()

			pubToName := make(map[string]string)
			for name, pubKey := range addressBook {
				pubToName[pubKey] = name
			}

			fmt.Println(Purple + "=== CURRENT ACCOUNT BALANCES ===" + Reset)
			if len(balances) == 0 {
				fmt.Println("  No accounts exist yet.")
			} else {
				for acc, bal := range balances {

					displayName := acc

					if name, exists := pubToName[acc]; exists {
						displayName = name
					} else if len(acc) > 15 && acc != "FAUCET" {

						displayName = acc[:15] + "..."
					}

					fmt.Printf("  %-15s -> %s%.2f COINS%s\n", displayName, Green, float64(bal), Reset)
					//new
				}
			}
			fmt.Println(Purple + "=================================" + Reset)

		case 6:
			fmt.Println(Blue + "=== Full Cryptographic Ledger History ===" + Reset)
			for _, b := range bc.GetBlocks() {
				blockColor := Cyan
				if b.Index == 0 {
					blockColor = Purple
				}
				fmt.Printf("%sBlock #%d %s\n", blockColor, b.Index, Reset)
				fmt.Printf("  Hash: %s%s%s\n", Green, b.Hash, Reset)

				fmt.Printf("  Prev Hash: %s%s%s\n", Yellow, b.PreviousHash, Reset)
				// MerkleRoot has a pointer receiver; take address of range variable
				fmt.Printf("  Merkle Root: %s%s%s\n", Cyan, (&b).MerkleRoot(), Reset)
				fmt.Printf("  Nonce: %d | Tx Count: %d\n", b.Nonce, len(b.Transactions))

				fmt.Println("  Transactions:")
				if len(b.Transactions) == 0 {
					fmt.Println("      [Genesis Block - System Initialized]")
				}
				for _, tx := range b.Transactions {
					fmt.Printf("      [%s -> %s : %.2f Coins]\n", tx.Sender, tx.Recipient, float64(tx.Amount))
				}
				fmt.Println(strings.Repeat("-", 50))
			}

		case 7:
			fmt.Println(Yellow + "Auditing data integrity and cross-linking hashes..." + Reset)
			isValid, faultyIndex := bc.IsValid()
			if isValid {
				fmt.Println(Green + "VALID: Blockchain integrity intact. All cryptographic links secure!" + Reset)
			} else {
				fmt.Printf(Red+"CORRUPTED: Validation failed at Block #%d!\n"+Reset, faultyIndex)
			}

		case 8:

			fmt.Println(Blue + "=== Pending Transaction Pool ===" + Reset)
			if len(bc.PendingTransactions) == 0 {
				fmt.Println("  No pending transactions in the pool.")
			} else {
				for i, tx := range bc.PendingTransactions {
					fmt.Printf("  %d: [%s -> %s : %.2f Coins]\n", i+1, tx.Sender, tx.Recipient, float64(tx.Amount))
				}
			}
			fmt.Println(Blue + strings.Repeat("=", 32) + Reset)

		case 9:

			fmt.Println(Green + "Thank you for using Toy Blockchain! Securing database and shutting down..." + Reset)
			os.Exit(0)

		default:
			fmt.Println(Red + "Invalid choice! Please select an option between 1 and 8." + Reset)
		}

		cli.waitForUser()
	}
}

func (cli *CLI) printHeader() {
	fmt.Print("\033[H\033[2J")
	fmt.Println(Cyan + Bold + `
  ======================================================
  WELCOME TO THE TOY-BLOCKCHAIN & LEDGER SIMULATOR
  ======================================================` + Reset)
}

func (cli *CLI) printMenu(poolSize int) {
	fmt.Println(White + "\nWhat would you like to do today?" + Reset)
	fmt.Printf("  %s[1]%s Create a New Wallet (Saved in Address Book)\n", Bold+Purple, Reset)
	fmt.Printf("  %s[2]%s Request Free Coins (Faucet -> Pool)\n", Bold+Green, Reset)
	fmt.Printf("  %s[3]%s Send Coins (Auto-Signs with saved Key)\n", Bold+Green, Reset)
	fmt.Printf("  %s[4]%s Mine Block from Pending Pool %s[%d tx pending]%s\n", Bold+Yellow, Reset, Red, poolSize, Reset)
	fmt.Printf("  %s[5]%s View Everyone's Account Balances\n", Bold+Green, Reset)
	fmt.Printf("  %s[6]%s Audit Full Blockchain History (Print)\n", Bold+Green, Reset)
	fmt.Printf("  %s[7]%s Verify and Validate Blockchain Integrity\n", Bold+Cyan, Reset)
	fmt.Printf("  %s[8]%s View Pending Transaction Pool\n", Bold+Purple, Reset)
	fmt.Printf("  %s[9]%s Save & Exit Application\n", Bold+Red, Reset)
	fmt.Println()
}

func (cli *CLI) waitForUser() {
	fmt.Print(White + "\nPress [ENTER] to return to the main menu..." + Reset)
	var discard string
	fmt.Scanln(&discard)
}

type WalletStore struct {
	AddressBook map[string]string `json:"address_book"`
	PrivateKeys map[string]string `json:"private_keys"`
}

func normalizeWalletName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
func hasWalletName(addressBook map[string]string, name string) bool {
	normalizedName := normalizeWalletName(name)
	for existingName := range addressBook {
		if normalizeWalletName(existingName) == normalizedName {
			return true
		}
	}
	return false
}

func resolveWalletName(addressBook map[string]string, name string) (string, bool) {
	normalizedName := normalizeWalletName(name)
	for existingName, pubKey := range addressBook {
		if normalizeWalletName(existingName) == normalizedName {
			return pubKey, true
		}
	}
	return "", false
}

func loadWallets(dbFile string) (map[string]string, map[string]string) {
	walletPath := walletStorePath(dbFile)
	data, err := os.ReadFile(walletPath)
	if err != nil {
		legacyPath := filepath.Join(".", "wallets.json")
		if legacyPath == walletPath {
			return make(map[string]string), make(map[string]string)
		}
		data, err = os.ReadFile(legacyPath)
		if err != nil {
			return make(map[string]string), make(map[string]string)
		}
	}
	var store WalletStore
	// json.Unmarshal(data, &store)

	if err := json.Unmarshal(data, &store); err != nil {
		return make(map[string]string), make(map[string]string)
	}

	addressBook := make(map[string]string, len(store.AddressBook))
	privateKeys := make(map[string]string, len(store.PrivateKeys))
	for name, pubKey := range store.AddressBook {
		normalizedName := normalizeWalletName(name)
		if normalizedName == "" {
			continue
		}
		addressBook[normalizedName] = pubKey
	}
	for name, privKey := range store.PrivateKeys {
		normalizedName := normalizeWalletName(name)
		if normalizedName == "" {
			continue
		}
		privateKeys[normalizedName] = privKey
	}

	return addressBook, privateKeys
}

func saveWallets(addresses, privates map[string]string, dbFile string) {
	store := WalletStore{AddressBook: addresses, PrivateKeys: privates}
	data, _ := json.MarshalIndent(store, "", "  ")
	walletPath := walletStorePath(dbFile)
	if err := os.MkdirAll(filepath.Dir(walletPath), 0o755); err == nil {
		_ = os.WriteFile(walletPath, data, 0644)
	}
}

func walletStorePath(dbFile string) string {
	if dbFile == "" {
		return filepath.Join("data", "wallets.json")
	}

	dir := filepath.Dir(dbFile)
	if dir == "." || dir == "" {
		return filepath.Join("data", "wallets.json")
	}

	return filepath.Join(dir, "wallets.json")
}
