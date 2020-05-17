// This script is a small convenience tool for manipulating user accounts in the
// configured server database.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"os"

	_ "github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/auth"
	"github.com/dcrodman/archon/internal/data"
	"github.com/spf13/viper"
)

var (
	add        = flag.Bool("add", false, "Add an account.")
	pd         = flag.Bool("perm-delete", false, "Delete an account permanently.")
	softDelete = flag.Bool("delete", false, "Soft delete an account.")
	help       = flag.Bool("help", false, "Print this usage info.")
)

// initDataSource creates the connection to the database, and returns a func
// which should be deferred for cleanup.
func initDataSource() (func(), error) {
	dataSource := fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s sslmode=%s",
		viper.GetString("database.host"),
		viper.GetInt("database.port"),
		viper.GetString("database.name"),
		viper.GetString("database.username"),
		viper.GetString("database.password"),
		viper.GetString("database.sslmode"),
	)
	if err := data.Initialize(dataSource); err != nil {
		return nil, err
	}
	return data.Shutdown, nil
}

func main() {
	flag.Parse()

	if help != nil && *help {
		flag.Usage()
		os.Exit(0)
	}
	if flag.NFlag() != 1 {
		flag.Usage()
		os.Exit(1)
	}

	cleanup, err := initDataSource()
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer cleanup()

	// defer so os.Exit doesn't prevent our clean up.
	retCode := 0
	defer func() {
		if err != nil {
			fmt.Println(err.Error())
		}
		os.Exit(retCode)
	}()

	switch {
	case add != nil && *add:
		u := scanInput("Username")
		p := scanInput("Password")
		e := scanInput("Email")
		if err = addAccount(u, p, e); err != nil {
			retCode = 1
			fmt.Println(err.Error())
		}
	case softDelete != nil && *softDelete:
		u := scanInput("Username")
		if err = softDeleteAccount(u); err != nil {
			retCode = 1
			fmt.Println(err.Error())
		}
	case pd != nil && *pd:
		u := scanInput("Username")
		if err = permanentlyDeleteAccount(u); err != nil {
			retCode = 1
			fmt.Println(err.Error())
		}
	default:
		flag.Usage()
		retCode = 1
	}
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
