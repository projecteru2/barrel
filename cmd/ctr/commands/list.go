package commands

import (
	cli "github.com/urfave/cli/v2"

	"github.com/projecteru2/barrel/cmd/ctr/commands/list"
	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
)

// ListCommands .
func ListCommands(flags *ctrtypes.Flags) *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list network resources",
		Subcommands: []*cli.Command{
			list.BlocksCommand(flags),
			list.IPSCommand(flags),
		},
	}
}
