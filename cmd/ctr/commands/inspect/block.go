package inspect

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/juju/errors"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// BlockInspect .
type BlockInspect struct {
	c             ctr.Ctr
	block         *cnet.IPNet
	poolFlag      string
	onlyEmptyFlag bool
}

// BlockCommand .
func BlockCommand(_ *ctrtypes.Flags) *cli.Command {
	inspectBlock := BlockInspect{}

	return &cli.Command{
		Name:   "block",
		Usage:  "inspect calico block",
		Action: inspectBlock.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "pool",
				Required:    true,
				Destination: &inspectBlock.poolFlag,
			},
			&cli.BoolFlag{
				Name:        "only-empty",
				Usage:       "list empty blocks only",
				Destination: &inspectBlock.onlyEmptyFlag,
			},
		},
		Before: inspectBlock.init,
	}
}

func (gb *BlockInspect) init(cli *cli.Context) (err error) {
	blockCidr := cli.Args().First()
	if blockCidr == "" {
		return errors.New("")
	}
	_, gb.block, err = cnet.ParseCIDR(blockCidr)
	if err != nil {
		return err
	}
	return ctr.InitCtr(&gb.c, func(init *ctr.Init) {
		init.InitCalico()
		init.InitCalicoBackend()
	})
}

func (gb *BlockInspect) run(cli *cli.Context) (err error) {
	block, err := gb.c.GetBlock(cli.Context, *gb.block, gb.poolFlag)
	if err != nil {
		return err
	}

	content, err := json.MarshalIndent(ctr.FormatBlock(block), "", "  ")
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, string(content)+"\n")
	return err
}
