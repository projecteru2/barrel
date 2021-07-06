package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	_ "github.com/googleapis/gnostic/OpenAPIv2"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"
	_ "go.uber.org/automaxprocs"

	"github.com/projecteru2/barrel/app"
	"github.com/projecteru2/barrel/cni/handler"
	"github.com/projecteru2/barrel/cni/store/filesystem"
	"github.com/projecteru2/barrel/cni/subhandler"
	"github.com/projecteru2/barrel/driver"
	"github.com/projecteru2/barrel/resources"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/versioninfo"
	cniapp "github.com/projecteru2/docker-cni/app"
	"github.com/projecteru2/docker-cni/config"
)

func setupLog(l string) error {
	level, err := log.ParseLevel(l)
	if err != nil {
		return err
	}
	log.SetLevel(level)
	log.Infof("[SetupLog] log level: %s", l)

	formatter := &log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	}
	log.SetFormatter(formatter)
	log.SetOutput(os.Stdout)
	return nil
}

func run(c *cli.Context) (err error) {
	var (
		dockerdPath = c.String("dockerd-path")
	)
	utils.Initialize(c.Int("buffer-size"))
	if err = setupLog(c.String("log-level")); err != nil {
		return err
	}
	log.Printf("Hello Barrel, dockerdPath = %s", dockerdPath)

	hostEnvVars := c.StringSlice("host")
	log.Printf("hostEnvVars = %v", hostEnvVars)

	resources.Init(c.StringSlice("res-path-prefix"))

	hostname := c.String("hostname")
	if hostname == "" {
		if hostname, err = os.Hostname(); err != nil {
			return
		}
	}

	cniStore, err := filesystem.NewStore("/var/lib/barrel/cni")
	if err != nil {
		return
	}
	cniConf := config.Config{}
	if c.Bool("enable-cni") {
		cniConf, err = config.LoadConfig(c.String("cni-config"))
		if err != nil {
			return
		}
	}

	barrel := app.Application{
		Hostname:               hostname,
		Mode:                   strings.ToLower(c.String("mode")),
		DockerDaemonUnixSocket: dockerdPath,
		DockerAPIVersion:       "1.32",
		Hosts:                  hostEnvVars,
		DriverName:             driver.DriverName,
		IpamDriverName:         driver.DriverName + driver.IpamSuffix,
		DialTimeout:            time.Duration(6) * time.Second,
		RequestTimeout:         c.Duration("request-timeout"),
		CertFile:               c.String("tls-cert"),
		KeyFile:                c.String("tls-key"),
		ShutdownTimeout:        time.Duration(30) * time.Second,
		EnableCNMAgent:         c.Bool("enable-cnm-agent"),
		CNIBase:                subhandler.NewBase(cniConf, cniStore),
	}
	return barrel.Run()
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Print(versioninfo.VersionString())
	}

	var app *cli.App
	switch path.Base(os.Args[0]) {
	case "barrel-cni":
		store, err := filesystem.NewStore("/var/lib/barrel/cni")
		if err != nil {
			log.Fatalf("failed to new store: %+v", err)
		}
		app = cniapp.NewApp(handler.NewBarrelHandler(store), nil)
		if err := app.Run(os.Args); err != nil {
			os.Stdout.WriteString(errors.Unwrap(err).Error())
			log.Fatal(err)
		}
		return

	default:
		app = &cli.App{
			Name:    "Barrel",
			Usage:   "Dockerd with calico fixed IP feature",
			Action:  run,
			Version: versioninfo.VERSION,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "hostname",
					Usage:   "hostname",
					EnvVars: []string{"HOSTNAME"},
				},
				&cli.StringFlag{
					Name:    "mode",
					Aliases: []string{"m"},
					Value:   "default",
					Usage:   "proxy-only | network-plugin-only | default",
					EnvVars: []string{"BARREL_MODE"},
				},
				&cli.StringFlag{
					Name:    "dockerd-path",
					Aliases: []string{"D"},
					Value:   "unix:///var/run/docker.sock",
					Usage:   "dockerd path",
					EnvVars: []string{"BARREL_DOCKERD_PATH"},
				},
				&cli.StringSliceFlag{
					Name:    "host",
					Aliases: []string{"H"},
					Value:   cli.NewStringSlice("unix:///var/run/barrel.sock"),
					Usage:   "host, can set multiple times",
					EnvVars: []string{"BARREL_HOSTS"},
				},
				&cli.StringSliceFlag{
					Name:    "res-path-prefix",
					Usage:   "resource path prefix, can set multiple times",
					EnvVars: []string{"BARREL_RESOURCE_PATH_PREFIXES"},
				},
				&cli.StringFlag{
					Name:    "tls-cert",
					Aliases: []string{"TC"},
					Usage:   "tls-cert-file-path",
					EnvVars: []string{"BARREL_TLS_CERT_FILE_PATH"},
				},
				&cli.StringFlag{
					Name:    "tls-key",
					Aliases: []string{"TK"},
					Usage:   "tls-key-file-path",
					EnvVars: []string{"BARREL_TLS_KEY_FILE_PATH"},
				},
				&cli.IntFlag{
					Name:    "buffer-size",
					Usage:   "set buffer size",
					Value:   256,
					EnvVars: []string{"BARREL_BUFFER_SIZE"},
				},
				&cli.DurationFlag{
					Name:  "dial-timeout",
					Usage: "for dial timeout",
					Value: time.Second * 2,
				},
				&cli.DurationFlag{
					Name:  "request-timeout",
					Usage: "for barrel request services(docker, etcd, etc.) timeout",
					Value: time.Second * 120,
				},
				&cli.StringFlag{
					Name:    "log-level",
					Value:   "INFO",
					Usage:   "set log level",
					EnvVars: []string{"BARREL_LOG_LEVEL"},
				},
				&cli.BoolFlag{
					Name:    "enable-cnm-agent",
					Value:   false,
					Usage:   "enable cnm agent",
					EnvVars: []string{"BARREL_ENABLE_CNM_AGENT"},
				},
				&cli.BoolFlag{
					Name:    "enable-cni",
					Value:   false,
					Usage:   "enable cni network",
					EnvVars: []string{"BARREL_ENABLE_CNI"},
				},
				&cli.StringFlag{
					Name:    "cni-config",
					Value:   "/etc/docker/cni.yaml",
					Usage:   "cni config filepath",
					EnvVars: []string{"CNI_CONFIG_PATH"},
				},
			},
		}
		if err := app.Run(os.Args); err != nil {
			log.Fatal(err)
		}
	}

}
