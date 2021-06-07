package cni

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"

	"github.com/opencontainers/runtime-spec/specs-go"
	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/docker-cni/oci"
)

// NetEndpoint is the minimalist network unit
type NetEndpoint struct {
	IPv4  string
	Netns string
	Owner string
}

// ContainerMeta .
type ContainerMeta struct {
	Meta oci.ContainerMeta
}

var (
	ipv4Pattern *regexp.Regexp
)

func init() { // nolint
	ipv4Pattern = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)
}

// RequiresFixedIP .
func (c ContainerMeta) RequiresFixedIP() bool {
	for _, env := range c.Meta.Spec.Process.Env {
		parts := strings.Split(env, "=")
		if len(parts) == 2 && parts[0] == "fixed-ip" && parts[1] != "" {
			return true
		}
	}
	return false
}

// SpecificIP .
func (c ContainerMeta) SpecificIP() string {
	return c.Meta.SpecificIP()
}

// RequiresSpecificIP .
func (c ContainerMeta) RequiresSpecificIP() bool {
	return c.SpecificIP() != ""
}

// SetNetns .
func (c *ContainerMeta) SetNetns(netnsPath string) {
	for i, ns := range c.Meta.Spec.Linux.Namespaces {
		if ns.Type == specs.NetworkNamespace {
			c.Meta.Spec.Linux.Namespaces[i].Path = netnsPath
		}
	}
}

// ID .
func (c ContainerMeta) ID() string {
	return c.Meta.ID
}

// Netns .
func (c ContainerMeta) Netns() (path string) {
	return fmt.Sprintf("/proc/%d/ns/net", c.Meta.InitPid)
}

// IPv4 .
func (c ContainerMeta) IPv4() (ip string, err error) {
	args := []string{"ip", "-4", "a", "sh", "eth0"}
	var out []byte
	if err = utils.WithNetns(c.Netns(), func() error {
		out, err = exec.Command(args[0], args[1:]...).Output() // nolint
		return errors.WithStack(err)
	}); err != nil {
		return
	}
	return string(ipv4Pattern.Find(out)), nil
}

// Save .
func (c ContainerMeta) Save() error {
	return c.Meta.Save()
}
