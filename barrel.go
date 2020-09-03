package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"
	"time"

	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	calicov3 "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projecteru2/barrel/common"
	"github.com/projecteru2/barrel/versioninfo"
	barrelEtcdMeta "github.com/projecteru2/minions/barrel/etcd"

	dockerProxy "github.com/projecteru2/barrel/proxy"
	"github.com/projecteru2/barrel/utils"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	_ "go.uber.org/automaxprocs"
)

var (
	debug  bool
	config *apiconfig.CalicoAPIConfig
	calico calicov3.Interface
	etcd   *barrelEtcdMeta.Etcd
)

func setupLog(l string) error {
	if debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("[setupLog] Debug logging enabled")
	}
	level, err := log.ParseLevel(l)
	if err != nil {
		return err
	}
	log.SetLevel(level)

	formatter := &log.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	}
	log.SetFormatter(formatter)
	log.SetOutput(os.Stdout)
	return nil
}

func initialize(l string) error {
	var err error

	if config, err = apiconfig.LoadClientConfig(""); err != nil {
		return err
	}
	if calico, err = calicov3.New(*config); err != nil {
		return err
	}
	if etcd, err = barrelEtcdMeta.NewEtcdClient(context.Background(), *config); err != nil {
		return err
	}

	return setupLog(l)
}

func run(c *cli.Context) (err error) {
	var (
		dockerdSocketPath = c.String("dockerd-socket")
		dockerGid         int64
	)
	utils.Initialize(c.Int("buffer-size"))
	if err = initialize(c.String("log-level")); err != nil {
		return err
	}
	log.Printf("Hello Barrel, dockerdSocketPath = %s", dockerdSocketPath)

	var group *user.Group
	if group, err = user.LookupGroup("docker"); err != nil {
		return
	}
	log.Printf("The Gid of group docker is %s", group.Gid)
	if dockerGid, err = strconv.ParseInt(group.Gid, 10, 64); err != nil {
		return
	}

	hostEnvVars := c.StringSlice("host")
	driverEnvVar := c.String("driver")

	log.Printf("hostEnvVars = %v", hostEnvVars)
	log.Printf("driverEnvVar = %v", driverEnvVar)

	config := dockerProxy.Config{
		DockerdSocketPath: dockerdSocketPath,
		DialTimeout:       time.Duration(2) * time.Second,
		Driver:            driverEnvVar,
		Hosts:             hostEnvVars,
		CertFile:          c.String("tls-cert"),
		KeyFile:           c.String("tls-key"),
		DockerGid:         dockerGid,
	}

	var proxy dockerProxy.DisposableService
	if proxy, err = dockerProxy.NewProxy(config, etcd, calico); err != nil {
		return
	}

	go func() {
		if err = proxy.Service(); err != nil && err != common.ErrServiceShutdown {
			log.Errorf("[run] Proxy end with error, cause = %v.", err)
		}
	}()

	// wait for unix signals and try to GracefulStop
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	sig := <-sigs
	log.Infof("[run] Get signal %v.", sig)
	return proxy.Dispose()
}

func main() {
	cli.VersionPrinter = func(c *cli.Context) {
		fmt.Print(versioninfo.VersionString())
	}

	app := &cli.App{
		Name:    "Barrel",
		Usage:   "Dockerd with calico fixed IP feature",
		Action:  run,
		Version: versioninfo.VERSION,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Usage:       "debug or not",
				Destination: &debug,
				EnvVars:     []string{"BARREL_DEBUG"},
			},
			&cli.StringFlag{
				Name:    "dockerd-socket",
				Aliases: []string{"D"},
				Value:   "/var/run/dockerd.sock",
				Usage:   "dockerd socket",
				EnvVars: []string{"DOCKERD_SOCKET_PATH"},
			},
			&cli.StringSliceFlag{
				Name:    "host",
				Aliases: []string{"H"},
				Value:   cli.NewStringSlice("unix:///var/run/docker.sock"),
				Usage:   "host, can set multiple times",
				EnvVars: []string{"BARREL_HOSTS"},
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
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "INFO",
				Usage:   "set log level",
				EnvVars: []string{"BARREL_LOG_LEVEL"},
			},
			&cli.StringFlag{
				Name:    "driver",
				Aliases: []string{"ND"},
				Usage:   "network plugin driver name",
				Value:   "calico",
				EnvVars: []string{"BARREL_DRIVER"},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
