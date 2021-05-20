package cni

import (
	"os/exec"
	"regexp"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/projecteru2/docker-cni/oci"
)

type NetEndpoint struct {
	IPv4  string
	Netns string
}

type ContainerMeta struct {
	Meta oci.ContainerMeta
}

var (
	ipv4Pattern *regexp.Regexp
)

func init() {
	ipv4Pattern = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
}

func (c ContainerMeta) RequiresFixedIP() bool {
	for _, env := range c.Meta.Spec.Process.Env {
		parts := strings.Split(env, "=")
		if len(parts) == 2 && parts[0] == "fixed-ip" && parts[1] != "" {
			return true
		}
	}
	return false
}

func (c ContainerMeta) SpecificIP() string {
	for _, env := range c.Meta.Spec.Process.Env {
		parts := strings.Split(env, "=")
		if len(parts) == 2 && parts[0] == "ipv4" && parts[1] != "" {
			return parts[1]
		}
	}
	return ""
}

func (c ContainerMeta) RequiresSpecificIP() bool {
	return c.SpecificIP() != ""
}

func (c *ContainerMeta) SetNetns(netnsPath string) {
	for i, ns := range c.Meta.Spec.Linux.Namespaces {
		if ns.Type == specs.NetworkNamespace {
			c.Meta.Spec.Linux.Namespaces[i].Path = netnsPath
		}
	}
}

func (c ContainerMeta) ID() string {
	return c.Meta.ID
}

func (c ContainerMeta) Netns() (path string) {
	for _, ns := range c.Meta.Spec.Linux.Namespaces {
		if ns.Type == specs.NetworkNamespace {
			return ns.Path
		}
	}
	return
}

func (c ContainerMeta) IPv4() (ip string, err error) {
	args := []string{"ip", "net", "e", c.Netns(), "ip", "-4", "a", "sh", "eth0"}
	out, err := exec.Command("ip", args...).Output()
	if err != nil {
		return
	}
	return string(ipv4Pattern.Find(out)), err
}

func (c ContainerMeta) Save() error {
	return c.Meta.Save()
}
