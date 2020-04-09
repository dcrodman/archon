package main

import (
	"flag"
	"fmt"
	_ "github.com/dcrodman/archon"
	"github.com/dcrodman/archon/auth"
	"github.com/dcrodman/archon/data"
	"github.com/spf13/viper"
)

func main() {
	flag.Parse()
	if flag.NArg() < 3 {
		fmt.Println("Usage: add_account [username] [password] [email]")
		return
	}

	initializeDB()
	defer data.Shutdown()

	account, err := auth.CreateAccount(flag.Arg(0), flag.Arg(1), flag.Arg(2))
	if err == nil {
		fmt.Println("created account with ID", account.ID)
	} else {
		fmt.Println("failed to create account:", err)
	}
}

func initializeDB() {
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
	}
}
