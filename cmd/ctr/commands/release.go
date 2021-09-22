package commands

import (
	cli "github.com/urfave/cli/v2"

	"github.com/projecteru2/barrel/cmd/ctr/commands/release"
	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
)

// ReleaseCommands .
func ReleaseCommands(flags *ctrtypes.Flags) *cli.Command {
	return &cli.Command{
		Name:  "release",
		Usage: "release network resources",
		Subcommands: []*cli.Command{
			release.BlockCommand(flags),
			release.IPCommand(flags),
			release.WEPCommand(flags),
		},
	}
}
