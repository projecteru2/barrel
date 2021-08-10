package utils

import (
	"os/exec"
	"regexp"

	"github.com/pkg/errors"
	"github.com/projecteru2/barrel/utils"
)

var ipv4Pattern = regexp.MustCompile(`\d+\.\d+\.\d+\.\d+`)

// ProbeIPv4 investigate eth0 inside a netns
func ProbeIPv4(netns string) (ip string, err error) {
	args := []string{"ip", "-4", "a", "sh", "eth0"}
	var out []byte
	if err = utils.WithNetns(netns, func() error {
		out, err = exec.Command(args[0], args[1:]...).Output() // nolint
		return errors.WithStack(err)
	}); err != nil {
		return
	}
	return string(ipv4Pattern.Find(out)), nil
}
