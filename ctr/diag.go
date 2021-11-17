package ctr

import (
	"context"

	dtypes "github.com/docker/docker/api/types"
	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
)

// ListLeakedWorkloadEndpoints .
func (c *Ctr) ListLeakedWorkloadEndpoints(ctx context.Context, nodename string, poolname string) ([]v3.WorkloadEndpoint, error) {
	weps, err := c.ListWorkloadEndpoints(ctx, "", poolname)
	if err != nil {
		return nil, err
	}

	ctns, err := c.ListContainersByPool(ctx, poolname)
	if err != nil {
		return nil, err
	}

	m := make(map[string]dtypes.EndpointResource)
	for _, ctn := range ctns {
		m[ctn.EndpointID] = ctn
	}

	var leakedEndpoints []v3.WorkloadEndpoint
	for _, wep := range weps {
		if nodename != "" && wep.Spec.Node != nodename {
			continue
		}
		if _, ok := m[wep.Spec.Endpoint]; !ok {
			leakedEndpoints = append(leakedEndpoints, wep)
		}
	}
	return leakedEndpoints, nil
}
