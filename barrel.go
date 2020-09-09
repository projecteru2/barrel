package main

import (
	"fmt"
	"os"
	"os/signal"
	"os/user"
	"strconv"
	"syscall"
	"time"

	dockerClient "github.com/docker/docker/client"
	"github.com/docker/go-plugins-helpers/network"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
	calicov3 "github.com/projectcalico/libcalico-go/lib/clientv3"
	"github.com/projecteru2/barrel/driver"
	"github.com/projecteru2/barrel/driver/calicoplus"
	"github.com/projecteru2/barrel/driver/fake"
	"github.com/projecteru2/barrel/service"
	dockerProxy "github.com/projecteru2/barrel/service/proxy"
	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/store/etcd"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/versioninfo"
	log "github.com/sirupsen/logrus"
	cli "github.com/urfave/cli/v2"

	_ "go.uber.org/automaxprocs"
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
		dockerGid   int64
		apiConfig   *apiconfig.CalicoAPIConfig
		calico      calicov3.Interface
		stor        store.Store
		dockerCli   *dockerClient.Client
	)
	utils.Initialize(c.Int("buffer-size"))
	if err = setupLog(c.String("log-level")); err != nil {
		return err
	}
	log.Printf("Hello Barrel, dockerdPath = %s", dockerdPath)

	var group *user.Group
	if group, err = user.LookupGroup("docker"); err != nil {
		return
	}
	log.Printf("The Gid of group docker is %s", group.Gid)
	if dockerGid, err = strconv.ParseInt(group.Gid, 10, 64); err != nil {
		return
	}
	if apiConfig, err = apiconfig.LoadClientConfig(""); err != nil {
		return err
	}
	if calico, err = calicov3.New(*apiConfig); err != nil {
		return err
	}
	if stor, err = etcd.NewClient(c.Context, *apiConfig); err != nil {
		return err
	}
	if dockerCli, err = dockerClient.NewClient(dockerdPath, "1.32", nil, nil); err != nil {
		return err
	}

	hostEnvVars := c.StringSlice("host")
	log.Printf("hostEnvVars = %v", hostEnvVars)

	disableNetworkPlugin := c.Bool("disable-network-plugin")
	var ipamDriver driver.AddressManager
	var netDriver network.Driver

	errChannel := utils.NewWriteOnceChannel()
	if !disableNetworkPlugin {
		ipamDriver, netDriver = calicoplus.NewDrivers(calico, stor, dockerCli)
		go func() {
			if err = calicoplus.RunNetworkPlugin(ipamDriver, netDriver); err != nil {
				errChannel.Send(err)
				log.Errorf("[run] Network plugin end with error, cause = %v", err)
			}
		}()
	} else {
		ipamDriver = fake.NewFakeCalicoPlugin()
	}

	config := types.DockerConfig{
		DockerdSocketPath: dockerdPath,
		DialTimeout:       c.Duration("dial-timeout"),
		Hosts:             hostEnvVars,
		CertFile:          c.String("tls-cert"),
		KeyFile:           c.String("tls-key"),
		DockerGid:         dockerGid,
	}

	var proxy service.DisposableService
	if proxy, err = dockerProxy.NewProxy(config, ipamDriver); err != nil {
		return
	}

	go func() {
		if err = proxy.Service(); err != nil {
			errChannel.Send(err)
			if err != types.ErrServiceShutdown {
				log.Errorf("[run] Proxy end with error, cause = %v.", err)
				return
			}
			log.Error("[run] Proxy shutdown.")
		}
	}()

	go func() {
		// wait for unix signals and try to GracefulStop
		sigs := make(chan os.Signal, 1)
		signal.Notify(sigs, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
		sig := <-sigs
		log.Infof("[run] Get signal %v.", sig)
		if err := proxy.Dispose(); err != nil {
			log.Errorf("[run] Proxy disposed with error, cause = %v", err)
		}
	}()

	return errChannel.Wait()
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
			&cli.StringFlag{
				Name:    "dockerd-path",
				Aliases: []string{"D"},
				Value:   "unix:///var/run/dockerd.sock",
				Usage:   "dockerd path",
				EnvVars: []string{"BARREL_DOCKERD_PATH"},
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
			&cli.DurationFlag{
				Name:  "dial-timeout",
				Usage: "for dial timeout",
				Value: time.Second * 2,
			},
			&cli.StringFlag{
				Name:    "log-level",
				Value:   "INFO",
				Usage:   "set log level",
				EnvVars: []string{"BARREL_LOG_LEVEL"},
			},
			&cli.BoolFlag{
				Name:  "disable-network-plugin",
				Usage: "disable network plugin",
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
