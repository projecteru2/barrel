package ctr

import (
	"context"

	"github.com/juju/errors"

	"github.com/docker/docker/api/types"
	"github.com/projectcalico/libcalico-go/lib/options"
)

// ListContainers .
func (c *Ctr) ListContainers(ctx context.Context) ([]types.Container, error) {
	return c.dockerCli.ContainerList(ctx, types.ContainerListOptions{})
}

// ListNetworks .
func (c *Ctr) ListNetworks(ctx context.Context) ([]types.NetworkResource, error) {
	return c.dockerCli.NetworkList(ctx, types.NetworkListOptions{})
}

// ListContainerByPool .
func (c *Ctr) ListContainerByPool(ctx context.Context, poolname string) (map[string]types.EndpointResource, error) {
	ipPool, err := c.calico.IPPools().Get(ctx, poolname, options.GetOptions{})
	if err != nil {
		return nil, err
	}
	network, exists, err := c.getNetworkBySubnet(ctx, ipPool.Spec.CIDR)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.Errorf("network with subnet of %s not exists", ipPool.Spec.CIDR)
	}

	network, err = c.dockerCli.NetworkInspect(ctx, network.ID, types.NetworkInspectOptions{})
	if err != nil {
		return nil, err
	}
	return network.Containers, nil
}

func (c *Ctr) getNetworkBySubnet(ctx context.Context, subnet string) (net types.NetworkResource, exists bool, err error) {
	networks, err := c.dockerCli.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return net, false, err
	}
	for _, network := range networks {
		for _, ipamConfig := range network.IPAM.Config {
			if subnet == ipamConfig.Subnet {
				return network, true, nil
			}
		}
	}
	return net, false, nil
}
