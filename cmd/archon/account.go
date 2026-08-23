// This script is a small convenience tool for manipulating user accounts in the
// configured server database.
package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	archon "github.com/dcrodman/archon/internal"
	"github.com/dcrodman/archon/internal/shipgate"
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

var accountPromoteCmd = &cobra.Command{
	Use:   "promote",
	Short: "Promotes an account to GM",
	Run:   AccountPromoteCommand,
}

var accountBanCmd = &cobra.Command{
	Use:   "ban",
	Short: "Bans an account from being used",
	Run:   AccountBanCommand,
}

var accountRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restores an account to playable status",
	Long:  "Restores a banned or soft deleted account to a playable status so that a player may log in",
	Run:   AccountBanCommand,
}

var PermanentFlag bool

func initShipgate() {
	archon.InitConfig(ConfigFlag)
	if err := archon.InitLogger(); err != nil {
		fmt.Println("error initializing logger:", err)
		os.Exit(1)
	}

	// Change to the same directory as the config file so that any relative
	// paths in the config file will resolve.
	if ConfigFlag != "" {
		if err := os.Chdir(ConfigFlag); err != nil {
			fmt.Println("error changing to config directory:", err)
			os.Exit(1)
		}
	}

	shipgate.Init(shipgate.DBConfig{
		Engine:         archon.Config.Database.Engine,
		Filename:       archon.Config.QualifiedPath(archon.Config.Database.Filename),
		URL:            archon.Config.DatabaseURL(),
		LoggingEnabled: archon.Config.Debugging.DatabaseLoggingEnabled,
	}, archon.Logger)
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

func AccountAddCommand(cmd *cobra.Command, args []string) {
	initShipgate()
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

	account, err := shipgate.Shipgate.CreateAccount(context.Background(), email, username, password)
	if err != nil {
		fmt.Println("error creating account:", err)
		return
	}

	fmt.Printf("created account for '%s' (ID: %d)\n", account.Username, account.ID)
}

func AccountDeleteCommand(cmd *cobra.Command, args []string) {
	initShipgate()

	usernameInput, _ := popArg(args, "Username")
	username := strings.ToLower(usernameInput)

	if err := shipgate.Shipgate.DeleteAccount(context.Background(), username, PermanentFlag); err != nil {
		fmt.Println("error deleting account:", err)
		return
	}
	fmt.Println("deleted account")
}

func AccountPromoteCommand(cmd *cobra.Command, args []string) {
	initShipgate()

	usernameInput, _ := popArg(args, "Username")
	username := strings.ToLower(usernameInput)

	if err := shipgate.Shipgate.PromoteAccount(context.Background(), username); err != nil {
		fmt.Println("error deleting account:", err)
		return
	}
	fmt.Println("promoted account to GM")
}

func AccountBanCommand(cmd *cobra.Command, args []string) {
	initShipgate()

	usernameInput, _ := popArg(args, "Username")
	username := strings.ToLower(usernameInput)

	if err := shipgate.Shipgate.BanAccount(context.Background(), username); err != nil {
		fmt.Println("error deleting account:", err)
		return
	}
	fmt.Println("banned account")
}

func AccountRestoreCommand(cmd *cobra.Command, args []string) {
	initShipgate()

	usernameInput, _ := popArg(args, "Username")
	username := strings.ToLower(usernameInput)

	if err := shipgate.Shipgate.RestoreAccount(context.Background(), username); err != nil {
		fmt.Println("error restoring account:", err)
		return
	}
	fmt.Println("account has been restored to normal status")
}
