package assign

import (
	"context"

	"github.com/juju/errors"
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
	"github.com/projecteru2/barrel/types"
)

// IPAssign .
type IPAssign struct {
	*ctrtypes.Flags
	c           ctr.Ctr
	poolFlag    string
	nodeFlag    string
	ipArg       string
	fixedIPFlag bool
}

// IPCommand .
func IPCommand(flags *ctrtypes.Flags) *cli.Command {
	assignIP := IPAssign{Flags: flags}

	return &cli.Command{
		Name:      "ip",
		Usage:     "assign a ip",
		ArgsUsage: "ADDRESSV4",
		Action:    assignIP.run,
		Before:    assignIP.init,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "use poolname to specific from which pool to allocate the ip address",
				Required:    true,
				Destination: &assignIP.poolFlag,
			},
			&cli.StringFlag{
				Name:        "node",
				Usage:       "use nodename to specific on which host to allocate the ip address",
				Required:    true,
				Destination: &assignIP.nodeFlag,
			},
			&cli.BoolFlag{
				Name:        "fixed-ip",
				Usage:       "whether assign ip as a fixed-ip",
				Destination: &assignIP.fixedIPFlag,
			},
		},
	}
}

func (d *IPAssign) init(ctx *cli.Context) error {
	d.ipArg = ctx.Args().First()
	if d.ipArg == "" {
		return errors.New("must provide ip address")
	}
	return ctr.InitCtr(&d.c, func(init *ctr.Init) {
		init.InitAllocator(d.nodeFlag)
	})
}

func (d *IPAssign) run(ctx *cli.Context) (err error) {
	if err = d.assignIP(ctx.Context); err != nil {
		return err
	}
	ctr.Fprintln("assign ip success")
	return nil
}

func (d *IPAssign) assignIP(ctx context.Context) error {
	if d.fixedIPFlag {
		return d.c.AssignFixedIP(ctx, types.IP{
			PoolID:  d.poolFlag,
			Address: d.ipArg,
		})
	}
	return d.c.AssignIP(ctx, types.IP{
		PoolID:  d.poolFlag,
		Address: d.ipArg,
	})
}
