package vessel

import (
	"context"
	"testing"
	"time"

	dockerTypes "github.com/docker/docker/api/types"
	dockerNetworkTypes "github.com/docker/docker/api/types/network"
	"github.com/juju/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	dockerMocks "github.com/projecteru2/barrel/docker/mocks"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel/mocks"
)

func TestPollEvent(t *testing.T) {
	dockerContainers := []dockerTypes.Container{
		{
			ID: "containerID",
			NetworkSettings: &dockerTypes.SummaryNetworkSettings{
				Networks: map[string]*dockerNetworkTypes.EndpointSettings{
					"networkName": &dockerNetworkTypes.EndpointSettings{
						NetworkID:  "networkID",
						EndpointID: "endpointID",
						IPAddress:  "10.10.10.10",
					},
				},
			},
		},
	}
	contaienrInfos := []types.ContainerInfo{}
	matchEvent := newMatchEvent(
		"localhost", mockAllocator(), dockerContainers, contaienrInfos)
	pollers := pollers{
		LoggerFactory: utils.ObjectLogger{ObjectName: "vessel/pollers"},
		pollers:       make(map[string]map[string]endpointUpdatePoller),
	}
	pollers.newPoller(endpointUpdatePoller{
		networkID:  "networkID",
		endpointID: "endpointID",
		add:        true,
		timeout:    time.Now().Add(time.Duration(10) * time.Second),
	})
	pollers.match(matchEvent)
	r := matchEvent.result()
	t.Log(r)
	assert.True(t, len(r) == 1)
}

func TestPollRemoveEvent(t *testing.T) {
	dockerContainers := []dockerTypes.Container{}
	contaienrInfos := []types.ContainerInfo{
		{
			ID:       "containerID",
			HostName: "localhost",
			Networks: []types.Network{
				{
					NetworkID:  "networkID",
					EndpointID: "endpointID",
					Address: types.IP{
						PoolID:  "poolID",
						Address: "10.10.10.10",
					},
				},
			},
		},
	}
	matchEvent := newMatchEvent(
		"localhost", mockAllocator(), dockerContainers, contaienrInfos)
	pollers := pollers{
		LoggerFactory: utils.ObjectLogger{ObjectName: "vessel/pollers"},
		pollers:       make(map[string]map[string]endpointUpdatePoller),
	}
	pollers.newPoller(endpointUpdatePoller{
		networkID:  "networkID",
		endpointID: "endpointID",
		add:        false,
		timeout:    time.Now().Add(time.Duration(10) * time.Second),
	})
	pollers.match(matchEvent)
	r := matchEvent.result()
	t.Log(r)
	assert.True(t, len(r) == 1)
}

func TestAgent(t *testing.T) {
	dockerClient := dockerMocks.Client{}
	containerRemoved := utils.NewAtomicBool(false)
	dockerClient.On("ContainerList", mock.Anything, mock.Anything).Return(
		func(_ context.Context, opt dockerTypes.ContainerListOptions) []dockerTypes.Container {
			if containerRemoved.Get() {
				return []dockerTypes.Container{
					{
						ID: "anotherContainerID",
						NetworkSettings: &dockerTypes.SummaryNetworkSettings{
							Networks: map[string]*dockerNetworkTypes.EndpointSettings{
								"anotherNetworkName": &dockerNetworkTypes.EndpointSettings{
									NetworkID:  "anotherNetworkID",
									EndpointID: "anotherEndpointID",
									IPAddress:  "10.10.20.10",
								},
							},
						},
					},
				}
			}
			return []dockerTypes.Container{
				{
					ID: "containerID",
					NetworkSettings: &dockerTypes.SummaryNetworkSettings{
						Networks: map[string]*dockerNetworkTypes.EndpointSettings{
							"networkName": &dockerNetworkTypes.EndpointSettings{
								NetworkID:  "networkID",
								EndpointID: "endpointID",
								IPAddress:  "10.10.10.10",
							},
						},
					},
				},
				{
					ID: "anotherContainerID",
					NetworkSettings: &dockerTypes.SummaryNetworkSettings{
						Networks: map[string]*dockerNetworkTypes.EndpointSettings{
							"anotherNetworkName": &dockerNetworkTypes.EndpointSettings{
								NetworkID:  "anotherNetworkID",
								EndpointID: "anotherEndpointID",
								IPAddress:  "10.10.20.10",
							},
						},
					},
				},
			}
		},
		nil,
	)

	chContainerUpdated := make(chan time.Time)
	defer close(chContainerUpdated)
	chContainerRemoved := make(chan time.Time)
	defer close(chContainerRemoved)
	containerVessel := mocks.ContainerVessel{}

	infoMap := make(map[string]types.ContainerInfo)
	containerVessel.On("ListContainers").Run(func(mock.Arguments) {
	}).Return(func() (result []types.ContainerInfo) {
		for _, item := range infoMap {
			result = append(result, item)
		}
		t.Logf("ListContainers, %v", result)
		return
	}, nil)
	containerVessel.On("UpdateContainer", mock.Anything, mock.Anything).Return(func(
		_ context.Context, info types.ContainerInfo,
	) error {
		t.Logf("update container = %v", info)
		infoMap[info.ID] = info
		go func() {
			chContainerUpdated <- time.Now()
		}()
		return nil
	})
	containerVessel.On("DeleteContainer", mock.Anything, mock.Anything).Return(func(
		_ context.Context, info types.ContainerInfo,
	) error {
		t.Logf("delete container = %v", info)
		delete(infoMap, info.ID)
		go func() {
			chContainerRemoved <- time.Now()
		}()
		return nil
	})

	allocator := mockAllocator()

	agent := networkAgentImpl{
		LoggerFactory: utils.ObjectLogger{
			Log:        utils.NewTestLogger(t),
			ObjectName: "networkAgentImpl",
		},
		hostname:        "localhost",
		dockerClient:    &dockerClient,
		containerVessel: &containerVessel,
		pollers:         newPollers(),
		allocator:       allocator,
		notifier: notifier{
			LoggerFactory: utils.ObjectLogger{
				Log:        utils.NewTestLogger(t),
				ObjectName: "notifier",
			},
		},
		pollInterval:    time.Duration(30) * time.Second,
		minPollInterval: time.Duration(1) * time.Second,
		pollTimeout:     time.Duration(30) * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())

	co := utils.Async(func() {
		disposable, err := agent.Serve(ctx)
		assert.NoError(t, err)

		err = disposable.Dispose(context.Background())
		assert.NoError(t, err)
	})

	// We will send an update signal
	// This will be done asynchronously
	agent.NotifyEndpointCreated("networkID", "endpointID")
	t.Log("Wait Container Update")
	// We wait for container to be update
	<-chContainerUpdated

	agent.NotifyEndpointRemoved("networkID", "endpointID")
	containerRemoved.Set(true)
	// we will send an update signal
	// This will be done asynchronously
	t.Log("Wait Container Removed")
	<-chContainerRemoved

	t.Log("Cancel")
	// Now we cancel the service
	cancel()

	t.Log("Wait Co End")
	// Wait co end
	co.Await()
}

func mockAllocator() *mocks.CalicoIPAllocator {
	allocator := mocks.CalicoIPAllocator{}
	poolMap := map[string][]types.Pool{
		"networkName": []types.Pool{
			{
				CIDR:    "10.10.10.0/24",
				Name:    "poolID",
				Gateway: "",
			},
		},
		"anotherNetworkName": []types.Pool{
			{
				CIDR:    "10.10.20.0/24",
				Name:    "anotherPoolID",
				Gateway: "",
			},
		},
	}
	allocator.On("GetPoolsByNetworkName", mock.Anything, mock.Anything).Return(func(_ context.Context, name string) []types.Pool {
		return poolMap[name]
	}, func(_ context.Context, name string) error {
		if _, ok := poolMap[name]; !ok {
			return errors.New("no such network")
		}
		return nil
	})
	return &allocator
}
