package commands

import (
	"github.com/projecteru2/barrel/cmd/ctr/commands/assign"
	"github.com/projecteru2/barrel/cmd/ctr/types"
	cli "github.com/urfave/cli/v2"
)

// AssignCommands .
func AssignCommands(flags *types.Flags) *cli.Command {
	return &cli.Command{
		Name:  "assign",
		Usage: "assign network resources",
		Subcommands: []*cli.Command{
			assign.IPCommand(flags),
		},
	}
}
