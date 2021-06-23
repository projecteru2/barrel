package etcd

import (
	"testing"

	"github.com/coreos/etcd/clientv3"
	"github.com/coreos/etcd/integration"
)

// EmbedEtcd .
type EmbedEtcd struct {
	cluster *integration.ClusterV3
}

// NewEmbedEtcd .
func NewEmbedEtcd(t *testing.T) *EmbedEtcd {
	Cluster := integration.NewClusterV3(t, &integration.ClusterConfig{Size: 1})
	t.Cleanup(func() {
		Cluster.Terminate(t)
	})
	return &EmbedEtcd{cluster: Cluster}
}

// Client .
func (e *EmbedEtcd) Client() *clientv3.Client {
	return e.cluster.RandClient()
}
