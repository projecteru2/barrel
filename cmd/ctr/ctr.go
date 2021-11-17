package main

import (
	"fmt"
	"os"

	"github.com/juju/errors"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	"github.com/projecteru2/barrel/cmd/ctr/commands"
	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/versioninfo"
)

const envETCDEndpoints = "ETCD_ENDPOINTS"

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Print(versioninfo.VersionString())
	}

	flags := ctrtypes.Flags{}
	var etcdEndpoints string

	app := &cli.App{
		Name:    "eru-barrel-utils",
		Version: versioninfo.VERSION,
		Before: func(c *cli.Context) error {
			if os.Getenv(envETCDEndpoints) != "" {
				return nil
			}

			if etcdEndpoints != "" {
				return os.Setenv("ETCD_ENDPOINTS", etcdEndpoints)
			}

			return errors.New("must specific etcd endpoints from options or environment variable")
		},
		Commands: []*cli.Command{
			commands.AssignCommands(&flags),
			commands.ReleaseCommands(&flags),
			commands.DiagCommands(&flags),
			commands.InspectCommands(&flags),
			commands.ListCommands(&flags),
		},
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "etcd-endpoints",
				Usage:       "etcd endpoints",
				Value:       "http://127.0.0.1:2379",
				Destination: &etcdEndpoints,
			},
			&cli.StringFlag{
				Name:        "docker-host",
				Usage:       "docker host",
				Value:       "unix:///var/run/docker.sock",
				Destination: &flags.DockerHostFlag,
			},
			&cli.StringFlag{
				Name:        "docker-version",
				Usage:       "docker version (could be empty)",
				Destination: &flags.DockerVersionFlag,
				Value:       "1.37",
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
