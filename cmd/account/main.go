// This script is a small convenience tool for manipulating user accounts in the
// configured server database.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	"github.com/dcrodman/archon/internal/core"
	"github.com/dcrodman/archon/internal/core/auth"
	"github.com/dcrodman/archon/internal/core/data"
)

var config = flag.String("config", "./", "Path to the directory containing the server config file")
var username = flag.String("username", "", "Username for user operation")
var password = flag.String("password", "", "Password for user operation")
var email = flag.String("email", "", "Email for user operation")

func main() {
	flag.Usage = usage
	flag.Parse()

	cleanup, err := initDataSource()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer func() {
		if err := cleanup(); err != nil {
			fmt.Println(err)
		}
	}()

	// defer so os.Exit doesn't prevent our clean up.
	retCode := 0
	defer func() {
		if err != nil {
			fmt.Println(err.Error())
		}
		os.Exit(retCode)
	}()

	switch flag.Arg(0) {
	case "add":
		u := checkFlag(username, "Username")
		p := checkFlag(password, "Password")
		e := checkFlag(email, "Email")
		if err = addAccount(u, p, e); err != nil {
			retCode = 1
			fmt.Println(err.Error())
		}
	case "delete":
		u := checkFlag(username, "Username")
		if err = softDeleteAccount(u); err != nil {
			retCode = 1
			fmt.Println(err.Error())
		}
	case "perm-delete":
		u := checkFlag(username, "Username")
		if err = permanentlyDeleteAccount(u); err != nil {
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

// initDataSource creates the connection to the database, and returns a func
// which should be deferred for cleanup.
func initDataSource() (func() error, error) {
	config := core.LoadConfig(*config)
	if err := data.Initialize(config.DatabaseURL(), config.Debugging.Enabled); err != nil {
		return nil, err
	}
	return data.Shutdown, nil
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

func addAccount(username, password, email string) error {
	account, err := auth.CreateAccount(username, password, email)
	if err != nil {
		return fmt.Errorf("failed to create account: %v", err)
	}
	fmt.Println("created account with ID: ", account.ID)
	return nil
}

func softDeleteAccount(username string) error {
	if err := auth.DeleteAccount(username); err != nil {
		return fmt.Errorf("failed to delete account: %v", err)
	}
	fmt.Println("deleted account")
	return nil
}

func permanentlyDeleteAccount(username string) error {
	if err := auth.PermanentlyDeleteAccount(username); err != nil {
		return fmt.Errorf("failed to delete account: %v", err)
	}
	fmt.Println("deleted account")
	return nil
}
