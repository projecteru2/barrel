package vessel

import (
	"context"
	"sync"
	"time"

	dockerTypes "github.com/docker/docker/api/types"

	"github.com/projecteru2/barrel/docker"
	"github.com/projecteru2/barrel/service"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
)

// CNMAgent .
type CNMAgent interface {
	NotifyEndpointCreated(networkID string, endpointID string)
	NotifyEndpointRemoved(networkID string, endpointID string)
}

// AgentConfig .
type AgentConfig struct {
	HostName     string
	MinInterval  time.Duration
	PollInterval time.Duration
	PollTimeout  time.Duration
}

type networkAgentImpl struct {
	utils.LoggerFactory
	hostname        string
	pollers         pollers
	allocator       CalicoIPAllocator
	dockerClient    docker.Client
	chErr           chan error
	mutex           sync.Mutex
	pollInterval    time.Duration
	minPollInterval time.Duration
	pollTimeout     time.Duration
	closed          utils.AtomicBool
	notifier        notifier
	containerVessel ContainerVessel
}

// NewAgent .
func NewAgent(vess Vessel, config AgentConfig) interface {
	CNMAgent
	service.Service
} {
	return &networkAgentImpl{
		LoggerFactory:   utils.NewObjectLogger("networkAgentImpl"),
		pollers:         newPollers(),
		allocator:       vess.CalicoIPAllocator(),
		containerVessel: vess.ContainerVessel(),
		minPollInterval: config.MinInterval,
		pollInterval:    config.PollInterval,
		pollTimeout:     config.PollTimeout,
		notifier: notifier{
			LoggerFactory: utils.NewObjectLogger("notifier"),
		},
	}
}

func (impl *networkAgentImpl) Serve(ctx context.Context) (service.Disposable, error) {
	logger := impl.Logger("Serve")
	impl.chErr = make(chan error)
	impl.serve()

	select {
	case err := <-impl.chErr:
		logger.Warnf("End with error, cause=%v", err)
		return impl, err
	case <-ctx.Done():
		logger.Info("Done")
		return impl, nil
	}
}

func (impl *networkAgentImpl) serve() {
	logger := impl.Logger("serve")
	go func() {
		logger.Info("starting")
		for {
			logger.Info("polling")
			if err := impl.poll(); err != nil {
				logger.Errorf("polling encountered error, cause=%v", err)
				impl.chErr <- err
				logger.Info("polling end")
				return
			}
			if impl.closed.Get() {
				logger.Info("interval closed, waiting for polling signal")
				ch := make(chan int)
				impl.notifier.wait(ch)
				impl.notifier.cancel(ch)
				close(ch)
				<-ch
				continue
			}
			impl.next()
		}
	}()
}

func (impl *networkAgentImpl) next() {
	logger := impl.Logger("next")
	logger.Info("waiting for next polling signal")
	if impl.pollers.size() > 0 {
		<-time.After(impl.minPollInterval)
		logger.Info("min inverval timeout signal arrived")
		return
	}

	logger.Info("make ch")
	ch := make(chan int)
	impl.notifier.wait(ch)
	defer impl.notifier.cancel(ch)
	defer close(ch)

	select {
	case <-time.After(impl.pollInterval):
		logger.Info("timeout signal arrived")
	case <-ch:
		logger.Info("poll signal arrived")
	}
}

func (impl *networkAgentImpl) Dispose(ctx context.Context) error {
	logger := impl.Logger("Dispose")
	logger.Info("Disposeing")
	impl.closed.Set(true)
	return nil
}

func (impl *networkAgentImpl) NotifyEndpointCreated(networkID string, endpointID string) {
	// and poller and send notify signal
	logger := impl.Logger("NotifyEndpointCreated")
	impl.pollers.newPoller(endpointUpdatePoller{
		networkID:  networkID,
		endpointID: endpointID,
		add:        true,
		timeout:    time.Now().Add(impl.pollTimeout),
	})
	logger.Info("Send polling signal")
	impl.notifier.send(0)
	logger.Info("End")
}

func (impl *networkAgentImpl) NotifyEndpointRemoved(networkID string, endpointID string) {
	logger := impl.Logger("NotifyEndpointRemoved")
	// if there are polling events we have to wait polling done
	impl.mutex.Lock()
	defer impl.mutex.Unlock()
	// we will remove pollers if there are any existes
	impl.pollers.newPoller(endpointUpdatePoller{
		networkID:  networkID,
		endpointID: endpointID,
		add:        false,
		timeout:    time.Now().Add(impl.pollTimeout),
	})
	logger.Info("Send polling signal")
	impl.notifier.send(0)
	logger.Info("End")
	// remove container
}

func (impl *networkAgentImpl) poll() error {
	logger := impl.Logger("poll")
	logger.Info("start")
	impl.mutex.Lock()
	defer impl.mutex.Unlock()

	if impl.pollers.size() == 0 {
		logger.Info("poller size == 0, return")
		return nil
	}

	var (
		dockerContainers []dockerTypes.Container
		containerInfos   []types.ContainerInfo
		err              error
	)
	if dockerContainers, err = impl.dockerClient.ContainerList(
		context.Background(),
		dockerTypes.ContainerListOptions{},
	); err != nil {
		logger.Errorf("ContainerList Error, %v", err)
		return err
	}

	if containerInfos, err = impl.containerVessel.ListContainers(); err != nil {
		logger.Errorf("ListContainers Error, %v", err)
		return err
	}

	matchEvent := newMatchEvent(impl.hostname, impl.allocator, dockerContainers, containerInfos)
	logger.Info("match pollers")
	impl.pollers.match(matchEvent)
	r := matchEvent.result()
	if len(r) > 0 {
		impl.commit(r)
	}
	logger.Info("end")
	return nil
}

func (impl *networkAgentImpl) commit(pollResults []pollResult) {
	logger := impl.Logger("commit")
	logger.Infof("start, matched results = %v", pollResults)
	for _, result := range pollResults {
		if result.add {
			if err := impl.containerVessel.UpdateContainer(context.Background(), result.containerInfo); err != nil {
				logger.Errorf("UpdateContainer error, cause=%v", err)
			}
			continue
		}
		if err := impl.containerVessel.DeleteContainer(context.Background(), result.containerInfo); err != nil {
			logger.Errorf("DeleteContainer error, cause=%v", err)
		}
	}
	logger.Info("end")
}

type pollResult struct {
	add           bool
	containerInfo types.ContainerInfo
}

type endpointUpdatePoller struct {
	networkID  string
	endpointID string
	add        bool
	timeout    time.Time
}

func (poller endpointUpdatePoller) isTimeout(now time.Time) bool {
	return poller.timeout.Before(now)
}

type pollers struct {
	utils.LoggerFactory
	mutex   sync.Mutex
	pSize   int
	pollers map[string]map[string]endpointUpdatePoller
}

func newPollers() pollers {
	return pollers{
		LoggerFactory: utils.ObjectLogger{ObjectName: "vessel/pollers"},
		pollers:       make(map[string]map[string]endpointUpdatePoller),
	}
}

func (pollers *pollers) newPoller(poller endpointUpdatePoller) {
	pollers.mutex.Lock()
	defer pollers.mutex.Unlock()

	var (
		m    map[string]endpointUpdatePoller
		prev endpointUpdatePoller
		ok   bool
	)
	if m, ok = pollers.pollers[poller.networkID]; !ok {
		m = make(map[string]endpointUpdatePoller)
		pollers.pollers[poller.networkID] = m
	}
	if prev, ok = m[poller.endpointID]; !ok {
		m[poller.endpointID] = poller
		pollers.pSize++
		return
	}
	if prev.add && !poller.add {
		delete(m, poller.endpointID)
		pollers.pSize--
	}
}

func (pollers *pollers) size() int {
	pollers.mutex.Lock()
	defer pollers.mutex.Unlock()

	return pollers.pSize
}

func (pollers *pollers) match(
	event matchEvent,
) {
	pollers.mutex.Lock()
	defer pollers.mutex.Unlock()

	now := time.Now()
	for _, m := range pollers.pollers {
		for endpointID, poller := range m {
			var (
				ok  bool
				err error
			)
			if ok, err = event.match(poller); err != nil {
				continue
			} else if ok || poller.isTimeout(now) {
				delete(m, endpointID)
				pollers.pSize--
				continue
			}
		}
	}
}

type matchEvent struct {
	utils.ObjectLogger
	allocator          CalicoIPAllocator
	hostname           string
	matched            map[string]pollResult
	containerMap       map[string]map[string]dockerTypes.Container
	vesselContainerMap map[string]map[string]types.ContainerInfo
}

func newMatchEvent(
	hostname string,
	allocator CalicoIPAllocator,
	containers []dockerTypes.Container,
	containerInfos []types.ContainerInfo,
) matchEvent {
	containerMap := make(map[string]map[string]dockerTypes.Container)
	vesselContainerMap := make(map[string]map[string]types.ContainerInfo)

	for _, container := range containers {
		for _, network := range container.NetworkSettings.Networks {
			m, ok := containerMap[network.NetworkID]
			if !ok {
				m = make(map[string]dockerTypes.Container)
				containerMap[network.NetworkID] = m
			}
			m[network.EndpointID] = container
		}
	}
	for _, containerInfo := range containerInfos {
		for _, network := range containerInfo.Networks {
			m, ok := vesselContainerMap[network.NetworkID]
			if !ok {
				m = make(map[string]types.ContainerInfo)
				vesselContainerMap[network.NetworkID] = m
			}
			m[network.EndpointID] = containerInfo
		}
	}
	return matchEvent{
		ObjectLogger:       utils.ObjectLogger{ObjectName: "matchEvent"},
		hostname:           hostname,
		allocator:          allocator,
		matched:            make(map[string]pollResult),
		containerMap:       containerMap,
		vesselContainerMap: vesselContainerMap,
	}
}

func (event matchEvent) result() []pollResult {
	var result []pollResult
	for _, item := range event.matched {
		result = append(result, item)
	}
	return result
}

func (event matchEvent) match(poller endpointUpdatePoller) (bool, error) {
	if poller.add {
		return event.matchAddPoller(poller)
	}
	return event.matchRemovePoller(poller)
}

func (event matchEvent) matchRemovePoller(poller endpointUpdatePoller) (bool, error) {
	logger := event.Logger("matchRemovePoller")
	var (
		m  map[string]dockerTypes.Container
		ok bool
	)
	logger.Info("start")
	if m, ok = event.containerMap[poller.networkID]; !ok {
		logger.Info("network not matched")
		event.makeRemovePollResult(poller)
		return true, nil
	}
	if _, ok = m[poller.endpointID]; !ok {
		logger.Info("endpoint not matched")
		event.makeRemovePollResult(poller)
		return true, nil
	}
	logger.Info("matched")
	return false, nil
}

func (event matchEvent) matchAddPoller(poller endpointUpdatePoller) (bool, error) {
	logger := event.Logger("matchAddPoller")
	var (
		err error
		m   map[string]dockerTypes.Container
		c   dockerTypes.Container
		ok  bool
		r   pollResult
	)
	logger.Info("start")
	if m, ok = event.containerMap[poller.networkID]; !ok {
		return false, nil
	}
	if c, ok = m[poller.endpointID]; !ok {
		return false, nil
	}
	if _, ok = event.matched[c.ID]; !ok {
		if r, err = event.makeAddPollResult(c); err != nil {
			return false, err
		}
		event.matched[c.ID] = r
	}
	return true, nil
}

func (event matchEvent) makeAddPollResult(c dockerTypes.Container) (pollResult, error) {
	logger := event.Logger("makeAddPollResult")
	var networks []types.Network
	// here we will try call docker api every iteration in order to avoid network change
	// and we should add a lock on calico plugin to lock on create network so we can cache network inspect result here
	for networkName, network := range c.NetworkSettings.Networks {
		var (
			pools   []types.Pool
			pool    types.Pool
			err     error
			address = network.IPAddress
		)
		if pools, err = event.allocator.GetPoolsByNetworkName(context.Background(), networkName); err != nil {
			return pollResult{}, err
		}
		size := len(pools)
		if size == 0 {
			continue
		}
		if size == 1 {
			pool = pools[0]
		}
		if size > 1 {
			for _, p := range pools {
				if ok, err := utils.Belongs(address, p.CIDR); err != nil {
					logger.Errorf("Test whether address in pool error, cause=%v", err)
					continue
				} else if ok {
					pool = p
				}
			}
		}

		networks = append(networks, types.Network{
			NetworkID:  network.NetworkID,
			EndpointID: network.EndpointID,
			Address: types.IP{
				PoolID:  pool.Name,
				Address: address,
			},
		})
	}
	return pollResult{
		add: true,
		containerInfo: types.ContainerInfo{
			ID:       c.ID,
			HostName: event.hostname,
			Networks: networks,
		},
	}, nil
}

func (event matchEvent) makeRemovePollResult(poller endpointUpdatePoller) {
	var (
		m  map[string]types.ContainerInfo
		c  types.ContainerInfo
		ok bool
	)
	if m, ok = event.vesselContainerMap[poller.networkID]; !ok {
		return
	}
	if c, ok = m[poller.endpointID]; !ok {
		return
	}
	if _, ok = event.matched[c.ID]; !ok {
		event.matched[c.ID] = pollResult{
			add:           false,
			containerInfo: c,
		}
	}
}

type notifier struct {
	utils.LoggerFactory
	mutex  sync.Mutex
	chs    []chan<- int
	hasSig bool
	sig    int
}

func (n *notifier) wait(ch chan<- int) {
	logger := n.Logger("wait")
	logger.Info("start")
	n.mutex.Lock()
	defer n.mutex.Unlock()
	defer logger.Info("end")

	if n.hasSig {
		n.hasSig = false
		go func() {
			ch <- n.sig
		}()
		return
	}

	for _, c := range n.chs {
		if c == ch {
			return
		}
	}

	logger.Info("append ch")
	n.chs = append(n.chs, ch)
}

func (n *notifier) remove(ch chan<- int) {
	size := len(n.chs)

	if size == 0 {
		return
	}
	if size == 1 {
		if n.chs[0] == ch {
			n.chs = n.chs[:0]
		}
		return
	}

	lastIdx := size - 1
	for idx, c := range n.chs {
		if c == ch {
			n.chs[idx] = n.chs[lastIdx]
			n.chs = n.chs[:lastIdx-1]
			return
		}
	}
}

func (n *notifier) cancel(ch chan<- int) {
	logger := n.Logger("cancel")
	logger.Info("start")
	n.mutex.Lock()
	defer n.mutex.Unlock()
	defer logger.Info("end")

	n.remove(ch)
}

func (n *notifier) send(sig int) {
	logger := n.Logger("send")
	logger.Info("start")

	n.mutex.Lock()
	defer n.mutex.Unlock()

	logger.Info("check chs")
	if len(n.chs) == 0 {
		logger.Info("no subscribers")
		n.hasSig = true
		n.sig = sig
		return
	}

	logger.Infof("subscriber size = %v", len(n.chs))
	for _, c := range n.chs {
		c <- sig
	}
	n.chs = n.chs[:0]
}
