package list

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/juju/errors"
	cli "github.com/urfave/cli/v2"

	"github.com/projectcalico/libcalico-go/lib/backend/model"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// BlockList .
type BlockList struct {
	c             ctr.Ctr
	hostFlag      string
	poolFlag      string
	onlyEmptyFlag bool
}

// BlocksCommand .
func BlocksCommand(_ *ctrtypes.Flags) *cli.Command {
	listBlocks := BlockList{}

	return &cli.Command{
		Name:      "blocks",
		Usage:     "list blocks",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "host",
				Usage:       "use hostname to specific host which blocks belongs to",
				Destination: &listBlocks.hostFlag,
			},
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "use poolname to specific pool that blocks belongs to",
				Required:    true,
				Destination: &listBlocks.poolFlag,
			},
			&cli.BoolFlag{
				Name:        "only-empty",
				Usage:       "list empty blocks only",
				Destination: &listBlocks.onlyEmptyFlag,
			},
		},
		Before: listBlocks.init,
		Action: listBlocks.run,
	}
}

func (list *BlockList) init(ctx *cli.Context) (err error) {
	if list.poolFlag == "" && list.hostFlag == "" {
		return errors.New("must either provide host or pool on which to list blocks")
	}
	return ctr.InitCtr(&list.c, func(init *ctr.Init) {
		init.InitCalico()
		init.InitCalicoBackend()
	})
}

func (list *BlockList) run(cli *cli.Context) (err error) {

	blocks, emptyBlocks, err := list.getBlocks(cli.Context)
	if err != nil {
		return err
	}

	size := len(blocks)
	emptySize := len(emptyBlocks)
	ctr.Fprintlnf(
		"there are %d blocks in pool, %d blocks are empty\n",
		size+emptySize, emptySize,
	)
	if len(emptyBlocks) == 0 && (list.onlyEmptyFlag || len(blocks) == 0) {
		return
	}

	var content []byte
	if list.onlyEmptyFlag {
		content, err = json.MarshalIndent(formatAllBlocks(nil, emptyBlocks), "", "  ")
	} else {
		content, err = json.MarshalIndent(formatAllBlocks(blocks, emptyBlocks), "", "  ")
	}
	if err != nil {
		return err
	}
	_, err = fmt.Fprint(os.Stdout, string(content)+"\n")
	return err
}

func (list *BlockList) getBlocks(ctx context.Context) (
	nonEmptyBlocks []*model.AllocationBlock, emptyBlocks []*model.AllocationBlock, err error,
) {
	blocks, err := list.c.ListBlocks(ctx, ctr.ListBlockByHostAndPoolOpt{
		Hostname: list.hostFlag,
		Poolname: list.poolFlag,
	})
	if err != nil {
		return nil, nil, err
	}
	nonEmptyBlocks, emptyBlocks = splitBlocks(blocks)
	return
}

func formatAllBlocks(blocks []*model.AllocationBlock, emptyBlocks []*model.AllocationBlock) FormattedBlocks {
	return FormattedBlocks{
		Blocks:      formatBlocks(blocks),
		EmptyBlocks: formatBlocks(emptyBlocks),
	}
}

func formatBlocks(blocks []*model.AllocationBlock) (result []*ctr.FormattedBlock) {
	for _, block := range blocks {
		result = append(result, ctr.FormatBlock(block))
	}
	return
}

func splitBlocks(blocks []*model.AllocationBlock) (nonEmpbyBlocks []*model.AllocationBlock, emptyBlocks []*model.AllocationBlock) {
	for _, block := range blocks {
		if !isEmpty(block) {
			nonEmpbyBlocks = append(nonEmpbyBlocks, block)
			continue
		}
		emptyBlocks = append(emptyBlocks, block)
	}
	return
}

func isEmpty(block *model.AllocationBlock) bool {
	for _, val := range block.Allocations {
		if val != nil {
			return false
		}
	}
	return true
}

// FormattedBlocks .
type FormattedBlocks struct {
	Blocks      []*ctr.FormattedBlock `json:"blocks,omitempty"`
	EmptyBlocks []*ctr.FormattedBlock `json:"emptyBlocks"`
}
