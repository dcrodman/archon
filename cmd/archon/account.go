// This script is a small convenience tool for manipulating user accounts in the
// configured server database.
package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/shipgate"
	"github.com/glebarez/sqlite"
	"github.com/spf13/cobra"
)

var accountCmd = &cobra.Command{
	Use:   "account",
	Short: "Account management tools",
}

var accountAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Registers new accounts in the database",
	Run:   AccountAddCommand,
}

var accountDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes accounts from the database",
	Run:   AccountDeleteCommand,
}

var PermanentFlag bool

func initDB() *gorm.DB {
	// Change to the same directory as the config file so that any relative
	// paths in the config file will resolve.
	if ConfigFlag != "" {
		if err := os.Chdir(ConfigFlag); err != nil {
			fmt.Println("error changing to config directory:", err)
			os.Exit(1)
		}
	}

	cfg := core.LoadConfig(ConfigFlag)
	var dialector gorm.Dialector
	switch strings.ToLower(cfg.Database.Engine) {
	case "sqlite":
		dialector = sqlite.Open(cfg.QualifiedPath("archon.db"))
	case "postgres":
		dialector = postgres.Open(cfg.DatabaseURL())
	default:
		fmt.Println("unsupported database engine:", cfg.Database.Engine)
		os.Exit(1)
	}

	db, err := gorm.Open(dialector, &gorm.Config{Logger: gormlogger.Discard})
	if err != nil {
		fmt.Println("error connecting to database:", err.Error())
		os.Exit(1)
	}
	return db
}

func AccountAddCommand(cmd *cobra.Command, args []string) {
	db := initDB()
	var (
		usernameInput string
		username      string

		passwordInput string
		password      string

		email string
	)

	usernameInput, args = popArg(args, "Username")
	username = strings.ToLower(usernameInput)
	if username != usernameInput {
		fmt.Println("Warning: PSOBB client does not support capital letters in usernames. Using lowercase version")
	}

	passwordInput, args = popArg(args, "Password")
	password = strings.ToLower(passwordInput)
	if password != usernameInput {
		fmt.Println("Warning: PSOBB client does not support capital letters in passwords. Using lowercase version")
	}

	email, _ = popArg(args, "Email")

	account, err := findAccount(db, username)
	if err != nil {
		fmt.Println("error finding account:", err)
		return
	} else if account != nil {
		fmt.Printf("account '%s' already exists; skipping\n", username)
		return
	}

	if err := data.CreateAccount(db, &data.Account{
		Username: username,
		Password: shipgate.HashPassword(password),
		Email:    email,
	}); err != nil {
		fmt.Println("error creating account:", err)
		return
	}

	account, err = findAccount(db, username)
	if err != nil {
		fmt.Println("error finding account:", err)
		return
	}
	fmt.Printf("created account for '%s' (ID: %d)\n", account.Username, account.ID)
}

func AccountDeleteCommand(cmd *cobra.Command, args []string) {
	db := initDB()

	usernameInput, _ := popArg(args, "Username")
	username := strings.ToLower(usernameInput)
	if username != usernameInput {
		fmt.Println("Warning: PSOBB client does not support capital letters in usernames. Using lowercase version")
	}

	account, err := findAccount(db, username)
	if err != nil {
		fmt.Println("error finding account:", err)
		return
	}

	if PermanentFlag {
		if err := data.PermanentlyDeleteAccount(db, account); err != nil {
			fmt.Println("error deleting account:", err)
			return
		}
	} else {
		if err := data.DeleteAccount(db, account); err != nil {
			fmt.Println("error deleting account:", err)
			return
		}
	}
	fmt.Println("deleted account")
}

func popArg(args []string, prompt string) (string, []string) {
	if len(args) == 1 {
		return args[0], nil
	} else if len(args) > 1 {
		return args[0], args[1:]
	}

	fmt.Printf("%s: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text(), args
}

func findAccount(db *gorm.DB, username string) (*data.Account, error) {
	account, err := data.FindAccountByUsername(db, username)
	if err != nil {
		return nil, fmt.Errorf("error looking up account: %v", err)
	}
	return account, nil
}
