package list

import (
	"encoding/json"
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/ctr"
)

// IPList .
type IPList struct {
	c        ctr.Ctr
	poolFlag string
}

// IPSCommand .
func IPSCommand(_ *ctrtypes.Flags) *cli.Command {
	list := IPList{}

	return &cli.Command{
		Name:      "ips",
		Usage:     "list barrel fixed ip",
		ArgsUsage: " ",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "pool",
				Usage:       "use poolname to specific pool that fixed ip belongs to",
				Destination: &list.poolFlag,
				Required:    true,
			},
		},
		Before: list.init,
		Action: list.run,
	}
}

func (list *IPList) init(ctx *cli.Context) error {
	return ctr.InitCtr(&list.c, func(init *ctr.Init) {
		init.InitStore()
	})
}

func (list *IPList) run(ctx *cli.Context) error {
	ips, err := list.c.ListFixedIP(ctx.Context, list.poolFlag)
	if err != nil {
		return err
	}
	var contents [][]byte
	for _, ip := range ips {
		content, err := json.Marshal(ip)
		if err != nil {
			log.WithError(err).Error("marshal ipinfo error")
			continue
		}
		contents = append(contents, content)
	}

	for _, content := range contents {
		if _, err := fmt.Fprintf(os.Stdout, "%s\n", content); err != nil {
			return err
		}
	}
	return nil
}
