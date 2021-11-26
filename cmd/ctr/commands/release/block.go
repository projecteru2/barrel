package release

import (
	log "github.com/sirupsen/logrus"

	cnet "github.com/projectcalico/libcalico-go/lib/net"
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// BlockRelease .
type BlockRelease struct {
	c        ctr.Ctr
	hostFlag string
	poolFlag string
	cidrFlag string
	cidr     *cnet.IPNet
}

// BlockCommand .
func BlockCommand(_ *ctrtypes.Flags) *cli.Command {
	release := BlockRelease{}

	return &cli.Command{
		Name:      "block",
		Usage:     "release calico ip blocks of specific host",
		ArgsUsage: "BLOCK_CIDR",
		Action:    release.run,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "host",
				Usage:       "use hostname to specific host which block belongs to (not required)",
				Destination: &release.hostFlag,
			},
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "use poolname to specific pool that block belongs to",
				Required:    true,
				Destination: &release.poolFlag,
			},
			&cli.StringFlag{
				Name:        "cidr",
				Usage:       "cidr for delete",
				Required:    true,
				Destination: &release.cidrFlag,
			},
		},
		Before: release.init,
	}
}

func (release *BlockRelease) init(ctx *cli.Context) (err error) {
	if release.hostFlag == "" {
		block, err := release.c.GetBlock(ctx.Context, *release.cidr, release.poolFlag)
		if err != nil {
			return err
		}
		release.hostFlag = ctr.AllocationBlockHost(block)
	}

	_, release.cidr, err = cnet.ParseCIDR(release.cidrFlag)
	if err != nil {
		log.WithError(err).Errorf("Parse cidr %s failed", release.cidrFlag)
		return err
	}
	return ctr.InitCtr(&release.c, func(init *ctr.Init) {
		init.InitCalico()
		init.InitCalicoBackend()
	})
}

func (release *BlockRelease) run(ctx *cli.Context) (err error) {
	if err := release.c.ReleaseEmptyBlock(ctx.Context, *release.cidr, release.hostFlag); err != nil {
		return err
	}
	log.Info("release block success")
	return nil
}
