package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	cli "github.com/urfave/cli/v2"

	"github.com/projecteru2/barrel/cmd/ctr/commands"
	ctrtypes "github.com/projecteru2/barrel/cmd/ctr/types"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/versioninfo"
)

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Print(versioninfo.VersionString())
	}

	flags := ctrtypes.Flags{}
	var configPath string

	app := &cli.App{
		Name:    "Barrel Ctr",
		Version: versioninfo.VERSION,
		Before: func(c *cli.Context) error {
			viper.AddConfigPath(configPath)
			viper.SetConfigName("ETCD_ENDPOINTS")
			viper.AutomaticEnv()

			if err := viper.ReadInConfig(); err != nil {
				return err
			}

			config := types.Config{}
			if err := viper.Unmarshal(&config); err != nil {
				return err
			}
			return os.Setenv("ETCD_ENDPOINTS", config.ETCDEndpoints)
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
				Name:        "config",
				Usage:       "environment config file for barrel",
				Value:       "/etc/eru/barrel.conf",
				Destination: &configPath,
			},
			&cli.StringFlag{
				Name:        "docker-host",
				Usage:       "docker host",
				Value:       "unix:///var/run/docker.sock",
				Destination: &flags.DockerHostFlag,
			},
			&cli.StringFlag{
				Name:        "docker-version",
				Usage:       "docker version",
				Destination: &flags.DockerVersionFlag,
			},
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
