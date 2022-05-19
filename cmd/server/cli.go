package server

import "github.com/urfave/cli/v2"

func Command() *cli.Command {
	return &cli.Command{
		Name:        "server",
		Usage:       "archon server",
		Description: "Runs the archon server.",
		Action:      server,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config",
				Aliases: []string{"c"},
				Usage:   "Path to the directory containing the server config file",
				EnvVars: []string{"ARCHON_CONFIG"},
				Value:   "./",
			},
		},
	}
}
