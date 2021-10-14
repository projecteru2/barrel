package commands

import (
	cli "github.com/urfave/cli/v2"

	"github.com/projecteru2/barrel/cmd/ctr/commands/inspect"
	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
)

// InspectCommands .
func InspectCommands(flags *ctrtypes.Flags) *cli.Command {
	return &cli.Command{
		Name:  "inspect",
		Usage: "inspect network resources",
		Subcommands: []*cli.Command{
			inspect.BlockCommand(flags),
			inspect.IPCommand(flags),
		},
	}
}
