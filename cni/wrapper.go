package cni

import (
	"context"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/projecteru2/barrel/proxy/docker"
	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/store/etcd"
)

type Wrapper struct {
	bin string

	store.Store
}

func NewWrapper(bin string) (wrapper *Wrapper, err error) {
	// TODO@zc: endpoints from config
	store, err := etcd.NewClient(context.Background(), []string{"http://127.0.0.1:2379"})
	if err != nil {
		return
	}
	return &Wrapper{
		bin:   bin,
		Store: store,
	}, nil
}

func (w Wrapper) Run() (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	cancel()
	switch os.Getenv("CNI_COMMAND") {
	case "ADD":
		err = w.runAdd(ctx)
	case "DEL":
		err = w.runDel(ctx)
	default:
		err = w.ExecCNI()
	}
	return err
}

func (w Wrapper) RequireFixedIP() bool {
	for _, env := range os.Environ() {
		parts := strings.Split(env, "=")
		if len(parts) == 2 && parts[0] == docker.FixedIPLabel && parts[1] != "" {
			return true
		}
	}
	return false
}

func (w Wrapper) ExecCNI() error {
	return syscall.Exec(w.bin, []string{w.bin}, os.Environ())
}

func (w Wrapper) GetWorkloadID() string {
	return os.Getenv("CNI_CONTAINERID")
}

func (w Wrapper) GetNetnsPath() string {
	return os.Getenv("CNI_NETNS")
}

func (w Wrapper) GetRequestIPv4() string {
	for _, args := range strings.Split(os.Getenv("CNI_ARGS"), ";") {
		parts := strings.Split(args, "=")
		if len(parts) == 2 && parts[0] == "ipv4" {
			return parts[1]
		}
	}
	return ""
}
