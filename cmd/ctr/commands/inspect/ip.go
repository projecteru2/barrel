package inspect

import (
	"encoding/json"
	"net"

	"github.com/juju/errors"
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
	"github.com/projecteru2/barrel/types"
)

// IPInspect .
type IPInspect struct {
	c        ctr.Ctr
	poolFlag string
	ipArg    string
}

// IPCommand .
func IPCommand(_ *ctrtypes.Flags) *cli.Command {
	getIP := IPInspect{}

	return &cli.Command{
		Name:      "ip",
		Usage:     "inspect barrel fixed ip",
		ArgsUsage: "ADDRESSV4",
		Action:    getIP.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "pool",
				Destination: &getIP.poolFlag,
				Required:    true,
			},
		},
		Before: getIP.init,
	}
}

func (g *IPInspect) init(ctx *cli.Context) error {
	g.ipArg = ctx.Args().First()
	if g.ipArg == "" {
		return errors.New("must provide ip address")
	}
	if ip := net.ParseIP(g.ipArg); ip == nil || ip.To4() == nil {
		return errors.New("must provide valid ipv4 address")
	}

	return ctr.InitCtr(&g.c, func(init *ctr.Init) {
		init.InitStore()
	})
}

func (g *IPInspect) run(ctx *cli.Context) error {
	ip, exists, err := g.c.InspectFixedIP(ctx.Context, types.IP{
		PoolID:  g.poolFlag,
		Address: g.ipArg,
	})
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("fixed ip is not assigned")
	}
	content, err := json.Marshal(&ip)
	if err != nil {
		return err
	}
	return ctr.Fprintln(string(content))
}
