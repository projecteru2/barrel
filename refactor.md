# 

```golang
type ReservedAddressManager interface {
	ReserveAddressFromPools(pools []types.Pool) (types.AddressWithVersion, error)
	ReserveAddress(address types.Address) error
  InitContainerInfoRecord(containerInfo types.ContainerInfo) error
  // will mark address first and then add address to container
	ReserveAddressForContainer(containerID string, address types.Address) error
	ReleaseContainerAddresses(containerID string) error
	ReleaseContainerAddressesByIPPools(containerID string, pools []types.Pool) error
	ReleaseReservedAddress(address types.Address) error
	GetIPPoolsByNetworkName(name string) ([]types.Pool, error)
	IsAddressReserved(address *types.Address) (bool, error)
	AquireIfReserved(address *types.Address) (bool, error)
}
```


```golang

type BarrelPoolManager interface {
	ReserveAddressFromPools(pools []types.Pool) (types.AddressWithVersion, error)
  ReserveAddress(address types.Address) error
  
	ReserveAddressForContainer(containerID string, address types.Address) error
	ReleaseReservedAddress(address types.Address) error
	GetIPPoolsByNetworkName(name string) ([]types.Pool, error)
	IsAddressReserved(address *types.Address) (bool, error)
	AquireIfReserved(address *types.Address) (bool, error)
}

type NetworkAgent interface {
  NotifyEndpointCreated(string)
	NotifyEndpointRemoved(string)
}

type Agent interface {

  HasFixedIPContainersByEndpoint(string) bool
  
	InitContainerInfoRecord(containerInfo types.ContainerInfo) error
	ReleaseContainerAddresses(containerID string) error
	ReleaseContainerAddressesByIPPools(containerID string, pools []types.Pool) error
}

type Pool interface {

}

type Endpoint struct {
  NetworkID string
  EndpointID string
  ContainerID string
}

type Container struct {
  ContainerID string
  Networks []struct {
    NetworkID string
    EndpointID string
    Address struct {
      PoolID string
      Value string
      Fixed bool
    }
  }
}

type NetworkStatus interface {
  ListEndpoints(string) []Endpoint
  GetEndpoint(string, string) (Endpoint, bool)
  AddEndpoint(Endpoint)
  RemoveEndpoint(Endpoint)
}

// Node .
type NodeManager interface {
  GetContainers()
}

```


# legency code

```golang

func Legency() {
  if container, endpointSettings, err = driver.findDockerContainerByEndpointID(request.EndpointID); err != nil {
		return err
	}

	if pool, err = driver.calNetDriver.FindPoolByNetworkID(endpointSettings.NetworkID); err != nil {
		return err
	}

	if containerHasFixedIPLabel(container) {
		if err = driver.ipam.ReserveAddressForContainer(
			container.ID,
			types.Address{
				PoolID:  pool.Name,
				Address: endpointSettings.IPAddress,
			},
		); err != nil {
			// we move on when reserve is failed
			log.Errorln(err)
		}
	}
}


func (driver *networkDriver) findDockerContainerByEndpointID(endpointID string) (dockerTypes.Container, *dockerNetworkTypes.EndpointSettings, error) {
	containers, err := driver.dockerCli.ContainerList(context.Background(), dockerTypes.ContainerListOptions{})
	if err != nil {
		log.Errorf("dockerCli ContainerList Error, %v", err)
		return dockerTypes.Container{}, nil, err
	}
	for _, container := range containers {
		for _, network := range container.NetworkSettings.Networks {
			if endpointID == network.EndpointID {
				return container, network, nil
			}
		}
	}
	return dockerTypes.Container{}, nil, errors.Errorf("find no container with endpintID = %s", endpointID)
}


func L1() {
		reserved, err := i.IsAddressReserved(
		&types.Address{
			PoolID:  request.PoolID,
			Address: request.Address,
		},
	)
	if err != nil {
		logger.Errorf("Get reserved ip status error, ip: %v", request.Address)
		return err
	}

	if reserved {
		logger.Infof("Ip is reserved, will not release to pool, ip: %v\n", request.Address)
		return nil
	}
}
```