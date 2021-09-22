package release

import (
	"github.com/juju/errors"

	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// DelWEP .
type DelWEP struct {
	c         ctr.Ctr
	host      string
	namespace string
}

// WEPCommand .
func WEPCommand(_ *ctrtypes.Flags) *cli.Command {
	delWEP := DelWEP{}

	return &cli.Command{
		Name:      "wep",
		Usage:     "release calico workload endpoints, relate ip will also be released",
		ArgsUsage: "workload endpoint name",
		Action:    delWEP.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "host",
				Usage:       "host",
				Destination: &delWEP.host,
			},
			&cli.StringFlag{
				Name:        "namespace",
				Usage:       "namespace",
				Destination: &delWEP.namespace,
			},
		},
		Before: delWEP.init,
	}
}

func (del *DelWEP) init(ctx *cli.Context) error {
	if err := ctr.InitHost(&del.host); err != nil {
		return err
	}
	return ctr.InitCtr(&del.c, func(init *ctr.Init) {
		init.InitCalico()
	})
}

func (del *DelWEP) run(ctx *cli.Context) error {
	args := ctx.Args()
	if args.Len() == 0 {
		return errors.New("must specific wep name")
	}
	wepName := args.First()

	return del.c.RecycleWorkloadEndpoint(ctx.Context, wepName, del.namespace)
}
