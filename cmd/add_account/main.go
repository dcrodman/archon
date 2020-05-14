// This script is a small convenience tool for creating user accounts in the
// configured server database.
package main

import (
	"bufio"
	"fmt"
	_ "github.com/dcrodman/archon"
	"github.com/dcrodman/archon/internal/auth"
	"github.com/dcrodman/archon/internal/data"
	"github.com/spf13/viper"
	"os"
)

func main() {
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
		fmt.Println(err.Error())
		os.Exit(1)
	}
	defer data.Shutdown()

	scanner := bufio.NewScanner(os.Stdin)

	fmt.Printf("Username: ")
	scanner.Scan()
	username := scanner.Text()

	fmt.Printf("Password: ")
	scanner.Scan()
	password := scanner.Text()

	fmt.Printf("Email: ")
	scanner.Scan()
	email := scanner.Text()

	account, err := auth.CreateAccount(username, password, email)
	if err == nil {
		fmt.Println("created account with ID", account.ID)
	} else {
		fmt.Println("failed to create account:", err)
	}
}
