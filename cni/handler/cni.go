package handler

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/projectcalico/cni-plugin/pkg/types"
)

// HandleCNIConfig .
func (h *BarrelHandler) HandleCNIConfig(config []byte) (newConfig []byte, err error) {
	cniArgs := os.Getenv("CNI_ARGS")
	ippool := ""
	for _, args := range strings.Split(cniArgs, ";") {
		if args == "" {
			continue
		}
		parts := strings.Split(args, "=")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid CNI_ARGS: '%s'", cniArgs)
		}
		if parts[0] == "IPPOOL" {
			ippool = parts[1]
		}
	}

	if ippool == "" {
		return config, nil
	}

	cniConfig := &types.NetConf{}
	if err = json.Unmarshal(config, cniConfig); err != nil {
		return nil, err
	}

	cniConfig.IPAM.IPv4Pools = []string{ippool}
	return json.Marshal(cniConfig)
}
