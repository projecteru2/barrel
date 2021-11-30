package vessel

import (
	"context"

	dockerTypes "github.com/docker/docker/api/types"
	dockerClient "github.com/docker/docker/client"

	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
)

// DockerNetworkManager .
type DockerNetworkManager interface {
	GetPoolsByNetworkName(ctx context.Context, name string) ([]types.Pool, error)
}

type dockerNetworkManager struct {
	allocator CalicoIPAllocator
	utils.LoggerFactory
	driverName string
	dockerCli  *dockerClient.Client
}

// NewDockerNetworkManager .
func NewDockerNetworkManager(dockerCli *dockerClient.Client, driverName string, allocator CalicoIPAllocator) DockerNetworkManager {
	return dockerNetworkManager{
		allocator:     allocator,
		driverName:    driverName,
		LoggerFactory: utils.NewObjectLogger("dockerNetworkManager"),
		dockerCli:     dockerCli,
	}
}

// GetIPPoolsByNetworkName .
func (m dockerNetworkManager) GetPoolsByNetworkName(ctx context.Context, name string) ([]types.Pool, error) {
	var (
		network dockerTypes.NetworkResource
		err     error
	)
	if network, err = m.dockerCli.NetworkInspect(ctx, name, dockerTypes.NetworkInspectOptions{}); err != nil {
		return nil, err
	}
	if network.Driver != m.driverName {
		return nil, types.ErrUnsupervisedNetwork
	}
	if len(network.IPAM.Config) == 0 {
		return nil, types.ErrConfiguredPoolUnfound
	}
	var cidrs []string
	for _, config := range network.IPAM.Config {
		cidrs = append(cidrs, config.Subnet)
	}
	return m.allocator.GetPoolsByCIDRS(ctx, cidrs)
}
