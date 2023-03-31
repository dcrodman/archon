// This script is a small convenience tool for manipulating user accounts in the
// configured server database.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/data"
	"github.com/dcrodman/archon/internal/shipgate"
	"github.com/glebarez/sqlite"
)

var (
	config   = flag.String("config", "./", "Path to the directory containing the server config file")
	username = flag.String("username", "", "Username for user operation")
	password = flag.String("password", "", "Password for user operation")
	email    = flag.String("email", "", "Email for user operation")
)

func main() {
	flag.Usage = usage
	flag.Parse()

	cfg := core.LoadConfig(*config)
	var dialector gorm.Dialector
	switch strings.ToLower(cfg.Database.Engine) {
	case "sqlite":
		dialector = sqlite.Open(filepath.Join(*config, "archon.db"))
	case "postgres":
		dialector = postgres.Open(cfg.DatabaseURL())
	default:
		fmt.Println("unsupported database engine:", cfg.Database.Engine)
		os.Exit(1)
	}

	db, err := gorm.Open(dialector, &gorm.Config{Logger: nil})
	if err != nil {
		os.Exit(1)
	}

	// defer so os.Exit doesn't prevent our clean up.
	retCode := 0
	defer func() {
		if err != nil {
			fmt.Println(err.Error())
		}
		os.Exit(retCode)
	}()

	usernameFlag := checkFlag(username, "Username")
	u := strings.ToLower(usernameFlag)
	if u != usernameFlag {
		fmt.Println("Warning: PSOBB client does not support capital letters in usernames. Using lowercase version")
	}

	switch flag.Arg(0) {
	case "add":
		// The PSOBB client always sends credentials in lowercase,
		p := checkFlag(password, "Password")
		e := checkFlag(email, "Email")
		if err = addAccount(db, u, p, e); err != nil {
			retCode = 1
			fmt.Println(err.Error())
		}
	case "delete":
		if err = softDeleteAccount(db, u); err != nil {
			retCode = 1
			fmt.Println(err.Error())
		}
	case "perm-delete":
		if err = permanentlyDeleteAccount(db, u); err != nil {
			retCode = 1
			fmt.Println(err.Error())
		}
	default:
		flag.Usage()
		retCode = 1
	}
}

func usage() {
	exName := os.Args[0]
	commands := map[string]string{
		"add":         "add an account",
		"delete":      "soft delete an account",
		"perm-delete": "permanently delete an account",
		"help":        "show this usage info",
	}

	fmt.Printf("%s <command>\n", exName)
	fmt.Println("The commands are:")
	for cmd, usage := range commands {
		fmt.Printf("\t%-13s%s\n", cmd, usage)
	}
}

func checkFlag(flag *string, prompt string) string {
	if *flag == "" {
		return scanInput(prompt)
	}
	return *flag
}

func scanInput(prompt string) string {
	fmt.Printf("%s: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return scanner.Text()
}

func addAccount(db *gorm.DB, username, password, email string) error {
	usernameLowered := strings.ToLower(username)
	if usernameLowered != username {
		fmt.Println("Warning: PSOBB client does not support capital letters in usernames. Using lowercase version")
	}

	account, err := findAccount(db, username)
	if err != nil {
		return err
	}

	if account == nil {
		if err := data.CreateAccount(db, &data.Account{
			Username: usernameLowered,
			Password: shipgate.HashPassword(password),
			Email:    email,
		}); err != nil {
			return fmt.Errorf("error creating account: %v", err)
		}
		fmt.Printf("created account for '%s' (ID: %d)\n", account.Username, account.ID)
	} else {
		fmt.Printf("account '%s' already exists; skipping\n", usernameLowered)
	}

	return nil
}

func findAccount(db *gorm.DB, username string) (*data.Account, error) {
	account, err := data.FindAccountByUsername(db, username)
	if err != nil {
		return nil, fmt.Errorf("error looking up account: %v", err)
	}
	return account, nil
}

func softDeleteAccount(db *gorm.DB, username string) error {
	account, err := findAccount(db, username)
	if err != nil {
		return err
	}

	if err := data.DeleteAccount(db, account); err != nil {
		return fmt.Errorf("error deleting account: %v", err)
	}
	fmt.Println("deleted account")
	return nil
}

func permanentlyDeleteAccount(db *gorm.DB, username string) error {
	account, err := findAccount(db, username)
	if err != nil {
		return err
	}

	if err := data.PermanentlyDeleteAccount(db, account); err != nil {
		return fmt.Errorf("error deleting account: %v", err)
	}
	fmt.Println("deleted account")
	return nil
}
