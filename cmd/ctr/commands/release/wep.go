package release

import (
	"github.com/juju/errors"

	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// DelWEP .
type DelWEP struct {
	c             ctr.Ctr
	namespaceFlag string
	wepNameArg    string
}

// WEPCommand .
func WEPCommand(_ *ctrtypes.Flags) *cli.Command {
	delWEP := DelWEP{}

	return &cli.Command{
		Name:      "wep",
		Usage:     "release calico workload endpoints, relate ip will also be released",
		ArgsUsage: "WORKLOAD_ENDPOINT_NAME",
		Action:    delWEP.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "namespace",
				Usage:       "specific namespace wep belongs to",
				Destination: &delWEP.namespaceFlag,
			},
		},
		Before: delWEP.init,
	}
}

func (del *DelWEP) init(ctx *cli.Context) error {
	del.wepNameArg = ctx.Args().First()
	if del.wepNameArg == "" {
		return errors.New("must specific wep name")
	}
	return ctr.InitCtr(&del.c, func(init *ctr.Init) {
		init.InitPoolManager()
		init.InitCalico()
	})
}

func (del *DelWEP) run(ctx *cli.Context) error {
	return del.c.RecycleWorkloadEndpointByName(ctx.Context, del.wepNameArg, del.namespaceFlag)
}
