package release

import (
	"context"
	"fmt"
	"os"

	"github.com/juju/errors"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// BlocksRelease .
type BlocksRelease struct {
	c        ctr.Ctr
	nodeFlag string
	poolFlag string
}

// BlocksCommand .
func BlocksCommand(_ *ctrtypes.Flags) *cli.Command {
	release := BlocksRelease{}

	return &cli.Command{
		Name:      "blocks",
		Usage:     "release calico ip blocks",
		ArgsUsage: " ",
		Action:    release.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "node",
				Usage:       "use nodename to specific node which blocks belongs to",
				Destination: &release.nodeFlag,
			},
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "use poolname to specific pool that blocks belongs to",
				Destination: &release.poolFlag,
			},
		},
		Before: release.init,
	}
}

func (release *BlocksRelease) init(ctx *cli.Context) (err error) {
	if release.poolFlag == "" && release.nodeFlag == "" {
		return errors.New("must either provide node or pool on which to release blocks")
	}
	return ctr.InitCtr(&release.c, func(init *ctr.Init) {
		init.InitCalico()
		init.InitCalicoBackend()
	})
}

func (release *BlocksRelease) run(ctx *cli.Context) (err error) {
	blocks, err := release.getEmptyBlocks(ctx.Context)
	if err != nil {
		return err
	}
	if len(blocks) == 0 {
		fmt.Fprintln(os.Stderr, "no empty blocks to release")
		return nil
	}
	for _, block := range blocks {
		if err := release.c.ReleaseEmptyBlock(ctx.Context, block.CIDR, ctr.AllocationBlockHost(block)); err != nil {
			fmt.Fprintf(os.Stderr, "release block(%v) error, cause = %v\n", block.CIDR, err)
			continue
		}
		fmt.Fprintf(os.Stdout, "release block success, cidr = %v, host = %v\n", block.CIDR, ctr.AllocationBlockHost(block))
	}
	return nil
}

func (release *BlocksRelease) getEmptyBlocks(ctx context.Context) (result []*model.AllocationBlock, err error) {
	blocks, err := release.c.ListBlocks(ctx, ctr.ListBlockByHostAndPoolOpt{
		Hostname: release.nodeFlag,
		Poolname: release.poolFlag,
	})
	if err != nil {
		return nil, err
	}
	for _, block := range blocks {
		if ctr.BlockIsEmpty(block) {
			result = append(result, block)
		}
	}
	return
}
