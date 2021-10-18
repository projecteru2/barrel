package release

import (
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
	ipFlag      string
	hostFlag    string
	unallocFlag bool
}

// IPCommand .
func IPCommand(flags *ctrtypes.Flags) *cli.Command {
	delIP := DelIP{
		Flags: flags,
	}

	return &cli.Command{
		Name:   "ip",
		Usage:  "release ip, fixed ip will be set to unoccupied",
		Action: delIP.run,
		Before: delIP.init,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "pool",
				Required:    true,
				Destination: &delIP.poolFlag,
			},
			&cli.StringFlag{
				Name:        "host",
				Usage:       "hostname",
				Destination: &delIP.hostFlag,
			},
			&cli.BoolFlag{
				Name:        "unalloc",
				Usage:       "unalloc fixed ip",
				Destination: &delIP.unallocFlag,
			},
		},
	}
}

func (d *DelIP) init(ctx *cli.Context) error {
	if err := ctr.InitHost(&d.hostFlag); err != nil {
		return err
	}
	d.ipFlag = ctx.Args().First()
	if d.ipFlag == "" {
		return errors.New("must provide ip address")
	}
	return ctr.InitCtr(&d.c, func(init *ctr.Init) {
		init.InitAllocator(d.DockerHostFlag, d.DockerVersionFlag, "")
	})
}

func (d *DelIP) run(ctx *cli.Context) error {
	if err := d.c.UnassignFixedIP(ctx.Context, types.IP{
		PoolID:  d.poolFlag,
		Address: d.ipFlag,
	}, d.unallocFlag); err != nil {
		ctr.Fprintln("unassign fixed ip success")
	}
	return nil
}
