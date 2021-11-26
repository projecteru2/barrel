package diag

import (
	"github.com/juju/errors"
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// DiagnosePool .
type DiagnosePool struct {
	*ctrtypes.Flags
	c       ctr.Ctr
	poolArg string
}

// PoolCommand .
func PoolCommand(flags *ctrtypes.Flags) *cli.Command {
	diagPool := DiagnosePool{
		Flags: flags,
	}

	return &cli.Command{
		Name:      "pool",
		Usage:     "diagnose network resources on pool",
		ArgsUsage: "POOLNAME",
		Action:    diagPool.run,
		Before: func(ctx *cli.Context) error {
			return diagPool.init(ctx, flags)
		},
	}
}

func (diag *DiagnosePool) init(ctx *cli.Context, flags *ctrtypes.Flags) (err error) {
	diag.poolArg = ctx.Args().First()
	if diag.poolArg == "" {
		return errors.New("must provide ip pool name")
	}
	return ctr.InitCtr(&diag.c, func(init *ctr.Init) {
		init.InitPoolManager()
		init.InitDocker(flags.DockerHostFlag, flags.DockerVersionFlag)
	})
}

func (diag *DiagnosePool) run(ctx *cli.Context) error {
	leaked, err := diag.c.ListLeakedWorkloadEndpoints(ctx.Context, "", diag.poolArg)
	if err != nil {
		return errors.Annotate(err, "list leaked WorkloadEndpoints error")
	}

	for _, wep := range leaked {
		ctr.Fprintlnf("WorkloadEndpoint(Name = %s, EndpointID = %s, Namespace = %s) is leaked or released, releated ip address = %v",
			wep.Name, wep.Spec.Endpoint, wep.Namespace, wep.Spec.IPNetworks,
		)
	}

	return nil
}

// InspectPoolResult .
type InspectPoolResult struct {
	FixedIP []string
}
