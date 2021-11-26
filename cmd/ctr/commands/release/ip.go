package release

import (
	"net"

	"github.com/juju/errors"
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
	"github.com/projecteru2/barrel/types"
)

// DelIP .
type DelIP struct {
	*ctrtypes.Flags
	c           ctr.Ctr
	poolFlag    string
	ipArg       string
	unallocFlag bool
}

// IPCommand .
func IPCommand(flags *ctrtypes.Flags) *cli.Command {
	delIP := DelIP{
		Flags: flags,
	}

	return &cli.Command{
		Name:      "ip",
		Usage:     "release ip, fixed ip will be set to unoccupied",
		ArgsUsage: "ADDRESSV4",
		Action:    delIP.run,
		Before:    delIP.init,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "use poolname to specific pool that ip belongs to",
				Required:    true,
				Destination: &delIP.poolFlag,
			},
			&cli.BoolFlag{
				Name:        "unalloc",
				Usage:       "if set, fixed ip will not only be unassigned but also unalloced",
				Destination: &delIP.unallocFlag,
			},
		},
	}
}

func (d *DelIP) init(ctx *cli.Context) error {
	d.ipArg = ctx.Args().First()
	if d.ipArg == "" {
		return errors.New("must provide ip address")
	}
	if ip := net.ParseIP(d.ipArg); ip == nil || ip.To4() == nil {
		return errors.New("must provide valid ipv4 address")
	}
	return ctr.InitCtr(&d.c, func(init *ctr.Init) {
		init.InitPoolManager()
	})
}

func (d *DelIP) run(ctx *cli.Context) error {
	if err := d.c.UnassignFixedIP(ctx.Context, types.IP{
		PoolID:  d.poolFlag,
		Address: d.ipArg,
	}, d.unallocFlag); err != nil {
		ctr.Fprintln("unassign fixed ip success")
	}
	return nil
}
