package diag

import (
	"github.com/juju/errors"
	cli "github.com/urfave/cli/v2"

	dockerTypes "github.com/docker/docker/api/types"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// DiagnosePool .
type DiagnosePool struct {
	*ctrtypes.Flags
	c    ctr.Ctr
	pool string
}

// PoolCommand .
func PoolCommand(flags *ctrtypes.Flags) *cli.Command {
	diagPool := DiagnosePool{
		Flags: flags,
	}

	return &cli.Command{
		Name:   "pool",
		Usage:  "diagnose network resources on pool",
		Action: diagPool.run,
		Before: diagPool.init,
	}
}

func (diag *DiagnosePool) init(ctx *cli.Context) (err error) {
	diag.pool = ctx.Args().First()
	if diag.pool == "" {
		return errors.New("must provide ip pool name")
	}
	return ctr.InitCtr(&diag.c, func(init *ctr.Init) {
		init.InitAllocator(diag.DockerHostFlag, diag.DockerVersionFlag, "")
	})
}

func (diag *DiagnosePool) run(ctx *cli.Context) error {
	weps, err := diag.c.ListWorkloadEndpoints(ctx.Context, "", diag.pool)
	if err != nil {
		return errors.Annotate(err, "list WorkloadEndpoints error")
	}

	containers, err := diag.c.ListContainerByPool(ctx.Context, diag.pool)
	if err != nil {
		return errors.Annotate(err, "list Containers error")
	}

	m := make(map[string]dockerTypes.EndpointResource)
	for _, container := range containers {
		m[container.EndpointID] = container
	}

	var leaked []v3.WorkloadEndpoint
	for _, wep := range weps {
		if _, ok := m[wep.Spec.Endpoint]; !ok {
			leaked = append(leaked, wep)
		}
	}

	for _, wep := range leaked {
		ctr.Fprintlnf("WorloadEndpoint(Name = %s, EndpointID = %s) is leaked or released, releated ip address = %v",
			wep.Name, wep.Spec.Endpoint, wep.Spec.IPNetworks,
		)
	}

	return nil
}

// InspectPoolResult .
type InspectPoolResult struct {
	FixedIP []string
}
