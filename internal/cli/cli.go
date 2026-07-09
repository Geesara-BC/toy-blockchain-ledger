package cli

import (
	"fmt"
	"os"
	"strings"
	"time"
	"toy-blockchain/internal/block"
	"toy-blockchain/internal/chain"
	"toy-blockchain/internal/ledger"
	"toy-blockchain/internal/storage"
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
	dbFile       string
	difficulty   int
	maxBlockSize int
}

func NewCLI(dbFile string, difficulty int, maxBlockSize int) *CLI {
	return &CLI{
		dbFile:       dbFile,
		difficulty:   difficulty,
		maxBlockSize: maxBlockSize,
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
	} else {
		bc = chain.NewBlockchain(cli.difficulty)
		err = storage.SaveToFile(cli.dbFile, bc)
		if err != nil {
			fmt.Printf(Red+"Error initializing blockchain file: %v\n"+Reset, err)
			os.Exit(1)
		}
	}

	l := ledger.NewLedger(bc)

	for {
		cli.printHeader()
		cli.printMenu(len(bc.PendingTransactions))

		fmt.Print(Bold + Cyan + "Choose an option (1-8): " + Reset)
		var choice int
		_, fmtScanErr := fmt.Scanln(&choice)

		if fmtScanErr != nil {
			fmt.Println(Red + "Invalid input! Please enter a number between 1 and 8.\n" + Reset)
			cli.waitForUser()
			continue
		}

		fmt.Println()

		switch choice {
		case 1:
			var to string
			var amount float64

			fmt.Print("Enter Your Name / Account Name: ")
			fmt.Scanln(&to)
			fmt.Print("Enter Amount of Free Coins Needed: ")
			_, err := fmt.Scanln(&amount)

			if to == "" || err != nil || amount <= 0 {
				fmt.Println(Red + "Error: Invalid name or amount!" + Reset)
				cli.waitForUser()
				continue
			}

			tx := block.Transaction{Sender: "FAUCET", Recipient: to, Amount: amount}
			bc.AddTransaction(tx)
			storage.SaveToFile(cli.dbFile, bc)
			fmt.Println(Green + "SUCCESS: Faucet transaction added to the pending pool!" + Reset)

		case 2:
			var from, to string
			var amount float64

			fmt.Print("Enter Sender Name (From): ")
			fmt.Scanln(&from)
			fmt.Print("Enter Recipient Name (To): ")
			fmt.Scanln(&to)
			fmt.Print("Enter Amount to Send: ")
			_, err := fmt.Scanln(&amount)

			if from == "" || to == "" || err != nil || amount <= 0 {
				fmt.Println(Red + "Error: Invalid accounts or amount!" + Reset)
				cli.waitForUser()
				continue
			}

			tx := block.Transaction{Sender: from, Recipient: to, Amount: amount}

			if err := l.VerifyTransaction(tx); err != nil {
				fmt.Printf(Red+"TRANSACTION REJECTED: %v\n"+Reset, err)
				cli.waitForUser()
				continue
			}

			bc.AddTransaction(tx)
			storage.SaveToFile(cli.dbFile, bc)
			fmt.Println(Green + "SUCCESS: Transaction added to the pending pool!" + Reset)

		case 3:
			fmt.Println(Yellow + "Mining pending transactions into a new block... Please wait..." + Reset)
			startTime := time.Now()
			newBlock, err := bc.MinePendingTransactions(cli.maxBlockSize)
			duration := time.Since(startTime)

			if err != nil {
				fmt.Printf(Red+"Mining Failed: %v\n"+Reset, err)
				cli.waitForUser()
				continue
			}

			storage.SaveToFile(cli.dbFile, bc)
			fmt.Println(Green + "SUCCESS: Block successfully mined and committed to the ledger!" + Reset)
			fmt.Printf("Time Taken: %v | Nonce: %d\n", duration, newBlock.Nonce)
			fmt.Printf("Block Hash: %s%s%s\n", Yellow, newBlock.Hash, Reset)

		case 4:
			balances := l.GetBalances()
			fmt.Println(Purple + "=== CURRENT ACCOUNT BALANCES ===" + Reset)
			if len(balances) == 0 {
				fmt.Println("  No accounts exist yet.")
			} else {
				for acc, bal := range balances {
					fmt.Printf("  %-12s -> %s%.2f COINS%s\n", acc, Green, bal, Reset)
				}
			}
			fmt.Println(Purple + "=================================" + Reset)

		case 5:
			fmt.Println(Blue + "=== Full Cryptographic Ledger History ===" + Reset)
			for _, b := range bc.Blocks {
				blockColor := Cyan
				if b.Index == 0 {
					blockColor = Purple
				}
				fmt.Printf("%sBlock #%d %s\n", blockColor, b.Index, Reset)
				fmt.Printf("  Hash: %s%s%s\n", Green, b.Hash, Reset)
				fmt.Println("  Transactions:")
				if len(b.Transactions) == 0 {
					fmt.Println("      [Genesis Block - System Initialized]")
				}
				for _, tx := range b.Transactions {
					fmt.Printf("      [%s -> %s : %.2f Coins]\n", tx.Sender, tx.Recipient, tx.Amount)
				}
				fmt.Println(strings.Repeat("-", 50))
			}

		case 6:
			fmt.Println(Yellow + "Auditing data integrity and cross-linking hashes..." + Reset)
			isValid, faultyIndex := bc.IsValid()
			if isValid {
				fmt.Println(Green + "VALID: Blockchain integrity intact. All cryptographic links secure!" + Reset)
			} else {
				fmt.Printf(Red+"CORRUPTED: Validation failed at Block #%d!\n"+Reset, faultyIndex)
			}

		case 7:

			fmt.Println(Blue + "=== Pending Transaction Pool ===" + Reset)
			if len(bc.PendingTransactions) == 0 {
				fmt.Println("  No pending transactions in the pool.")
			} else {
				for i, tx := range bc.PendingTransactions {
					fmt.Printf("  %d: [%s -> %s : %.2f Coins]\n", i+1, tx.Sender, tx.Recipient, tx.Amount)
				}
			}
			fmt.Println(Blue + strings.Repeat("=", 32) + Reset)

		case 8:

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
	fmt.Printf("  %s[1]%s Request Free Coins (Faucet -> Pool)\n", Bold+Green, Reset)
	fmt.Printf("  %s[2]%s Create Peer-to-Peer Transaction (-> Pool)\n", Bold+Green, Reset)
	fmt.Printf("  %s[3]%s Mine Block from Pending Pool %s[%d tx pending]%s\n", Bold+Yellow, Reset, Red, poolSize, Reset)
	fmt.Printf("  %s[4]%s View Everyone's Account Balances\n", Bold+Green, Reset)
	fmt.Printf("  %s[5]%s Audit Full Blockchain History (Print)\n", Bold+Green, Reset)
	fmt.Printf("  %s[6]%s Verify and Validate Blockchain Integrity\n", Bold+Cyan, Reset)
	fmt.Printf("  %s[7]%s View Pending Transaction Pool\n", Bold+Purple, Reset)
	fmt.Printf("  %s[8]%s Save & Exit Application\n", Bold+Red, Reset)
	fmt.Println()
}

func (cli *CLI) waitForUser() {
	fmt.Print(White + "\nPress [ENTER] to return to the main menu..." + Reset)
	var discard string
	fmt.Scanln(&discard)
}
