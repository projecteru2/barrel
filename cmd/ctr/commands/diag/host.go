package diag

import (
	"github.com/juju/errors"
	cli "github.com/urfave/cli/v2"

	dockerTypes "github.com/docker/docker/api/types"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// DiagnoseHost .
type DiagnoseHost struct {
	*ctrtypes.Flags
	c        ctr.Ctr
	host     string
	poolFlag string
}

// HostCommand .
func HostCommand(flags *ctrtypes.Flags) *cli.Command {
	diagHost := DiagnoseHost{
		Flags: flags,
	}

	return &cli.Command{
		Name:  "host",
		Usage: "diagnose network resources on host",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "pool name",
				Required:    true,
				Destination: &diagHost.poolFlag,
			},
		},
		Action: diagHost.run,
		Before: diagHost.init,
	}
}

func (diag *DiagnoseHost) init(ctx *cli.Context) (err error) {
	diag.host = ctx.Args().First()
	if err := ctr.InitHost(&diag.host); err != nil {
		return err
	}
	return ctr.InitCtr(&diag.c, func(init *ctr.Init) {
		init.InitAllocator(diag.DockerHostFlag, diag.DockerVersionFlag, diag.host)
	})
}

func (diag *DiagnoseHost) run(ctx *cli.Context) error {
	weps, err := diag.c.ListWorkloadEndpoints(ctx.Context, diag.host, diag.poolFlag)
	if err != nil {
		return errors.Annotate(err, "list WorkloadEndpoints error")
	}

	containers, err := diag.c.ListContainers(ctx.Context)
	if err != nil {
		return errors.Annotate(err, "list Containers error")
	}

	m := make(map[string]dockerTypes.Container)
	for _, container := range containers {
		for _, network := range container.NetworkSettings.Networks {
			m[network.EndpointID] = container
		}
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
