package main

import (
	"fmt"
	"os"

	"github.com/urfave/cli/v2"

	"github.com/dcrodman/archon/cmd/server"
)

func main() {
	if err := app().Run(os.Args); err != nil {
		fmt.Printf("archon error: %v", err)
	}
}

func app() *cli.App {
	app := cli.NewApp()
	app.Name = "archon"
	app.Commands = []*cli.Command{
		server.Command(),
	}
	app.Action = menu

	return app
}
