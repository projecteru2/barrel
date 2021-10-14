package inspect

import (
	"encoding/json"
	"fmt"
	"os"

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
	ipFlag   string
}

// IPCommand .
func IPCommand(_ *ctrtypes.Flags) *cli.Command {
	getIP := IPInspect{}

	return &cli.Command{
		Name:   "ip",
		Usage:  "inspect barrel fixed ip",
		Action: getIP.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "pool",
				Destination: &getIP.poolFlag,
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "ip",
				Usage:       "ip",
				Destination: &getIP.ipFlag,
				Required:    true,
			},
		},
		Before: getIP.init,
	}
}

func (g *IPInspect) init(ctx *cli.Context) error {
	return ctr.InitCtr(&g.c, func(init *ctr.Init) {
		init.InitStore()
	})
}

func (g *IPInspect) run(ctx *cli.Context) error {
	ip, exists, err := g.c.InspectFixedIP(ctx.Context, types.IP{
		PoolID:  ctx.String("poolname"),
		Address: ctx.String("ip"),
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
	_, err = fmt.Fprint(os.Stdout, content)
	return err
}
