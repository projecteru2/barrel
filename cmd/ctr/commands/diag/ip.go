package diag

import (
	"strings"

	"github.com/juju/errors"
	cli "github.com/urfave/cli/v2"

	cnet "github.com/projectcalico/libcalico-go/lib/net"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
	"github.com/projecteru2/barrel/types"
)

// DiagnoseIP .
type DiagnoseIP struct {
	*ctrtypes.Flags
	c        ctr.Ctr
	poolFlag string
	ip       string
}

// IPCommand .
func IPCommand(flags *ctrtypes.Flags) *cli.Command {
	diagIP := DiagnoseIP{
		Flags: flags,
	}

	return &cli.Command{
		Name:  "ip",
		Usage: "diagnose ip problem",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "pool",
				Destination: &diagIP.poolFlag,
				Required:    true,
			},
		},
		Action: diagIP.run,
		Before: diagIP.init,
	}
}

func (diag *DiagnoseIP) init(ctx *cli.Context) (err error) {
	diag.ip = ctx.Args().First()
	if diag.ip == "" {
		return errors.New("must provide ip address")
	}
	return ctr.InitCtr(&diag.c, func(init *ctr.Init) {
		init.InitDocker(diag.DockerHostFlag, diag.DockerVersionFlag)
		init.InitCalico()
		init.InitStore()
	})
}

func (diag *DiagnoseIP) run(ctx *cli.Context) error {
	cnetIP := cnet.ParseIP(diag.ip)
	if cnetIP == nil {
		return errors.New("arg0 is not a valid ip address")
	}

	ipInfo, exists, err := diag.c.InspectFixedIP(ctx.Context, types.IP{
		PoolID:  diag.poolFlag,
		Address: diag.ip,
	})
	if err != nil {
		return err
	}
	if exists {
		if !ipInfo.Status.Match(types.IPStatusInUse) {
			ctr.Fprintln(
				"the ip is a fixed ip, but it's not in use",
			)
		} else {
			ctr.Fprintln("the ip is a fixed ip, it's in use currently")
		}
	} else {
		ctr.Fprintln("the ip is not a fixed ip")
	}

	assigned, err := diag.c.Assigned(ctx.Context, *cnetIP)
	if err != nil {
		return errors.Annotate(err, "check ip assigned status error")
	}
	if assigned {
		ctr.Fprintln("the ip is assigned from calico ip pool")
	}

	weps, err := diag.c.ListWorkloadEndpoints(ctx.Context, "", diag.poolFlag)
	if err != nil {
		return errors.Annotate(err, "list WorkloadEndpoints error")
	}

	wepAssigned := false
	for _, wep := range weps {
		for _, ipNet := range wep.Spec.IPNetworks {
			if strings.HasPrefix(ipNet, diag.ip) {
				wepAssigned = true
				ctr.Fprintlnf("the ip is assigned to WorkloadEndpoint %s, namespace = %s",
					wep.Name, wep.Namespace,
				)
			}
		}
	}
	if !wepAssigned {
		ctr.Fprintlnf("the ip is not assigned to WorkloadEndpoint")
	}

	containers, err := diag.c.ListContainerByPool(ctx.Context, diag.poolFlag)
	if err != nil {
		return errors.Annotate(err, "list containers on subnet error")
	}
	occupied := false
	for containerID, container := range containers {
		if strings.HasPrefix(container.IPv4Address, diag.ip) {
			occupied = true
			ctr.Fprintlnf("the ip is occupied by container %s(id = %s), workloadendpointID = %s",
				container.Name, containerID, container.EndpointID,
			)
		}
	}
	if !occupied {
		ctr.Fprintln("the ip is not occupied by container")
	}
	return nil
}
