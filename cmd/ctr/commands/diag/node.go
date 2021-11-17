package diag

import (
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// DiagnoseNode .
type DiagnoseNode struct {
	*ctrtypes.Flags
	c        ctr.Ctr
	nodeArg  string
	poolFlag string
}

// NodeCommand .
func NodeCommand(flags *ctrtypes.Flags) *cli.Command {
	diagHost := DiagnoseNode{
		Flags: flags,
	}

	return &cli.Command{
		Name:      "node",
		Usage:     "diagnose network resources on node (in common case, it is the same as host), if NODENAME isn't give, local hostname will be used",
		ArgsUsage: "[NODENAME]",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "use poolname to specific on which pool to diagnose",
				Required:    true,
				Destination: &diagHost.poolFlag,
			},
		},
		Action: diagHost.run,
		Before: diagHost.init,
	}
}

func (diag *DiagnoseNode) init(ctx *cli.Context) (err error) {
	diag.nodeArg = ctx.Args().First()
	if err := ctr.InitHost(&diag.nodeArg); err != nil {
		return err
	}
	return ctr.InitCtr(&diag.c, func(init *ctr.Init) {
		init.InitAllocator(diag.nodeArg)
	})
}

func (diag *DiagnoseNode) run(ctx *cli.Context) error {
	leaked, err := diag.c.ListLeakedWorkloadEndpoints(ctx.Context, diag.nodeArg, diag.poolFlag)
	if err != nil {
		return err
	}

	for _, wep := range leaked {
		ctr.Fprintlnf("WorloadEndpoint(Name = %s, EndpointID = %s, Namespace = %s) is leaked or released, releated ip address = %v",
			wep.Name, wep.Spec.Endpoint, wep.Namespace, wep.Spec.IPNetworks,
		)
	}
	return nil
}
