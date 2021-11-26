package commands

import (
	cli "github.com/urfave/cli/v2"

	"github.com/projecteru2/barrel/cmd/ctr/commands/diag"
	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
)

// DiagCommands .
func DiagCommands(flags *ctrtypes.Flags) *cli.Command {
	return &cli.Command{
		Name:  "diag",
		Usage: "diagnose network problems",
		Subcommands: []*cli.Command{
			diag.PoolCommand(flags),
			diag.IPCommand(flags),
			diag.NodeCommand(flags),
		},
	}
}
