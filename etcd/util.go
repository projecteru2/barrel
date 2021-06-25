package etcd

import (
	"context"
	"strings"
	"time"

	"github.com/coreos/etcd/clientv3"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
)

const (
	clientTimeout    = 10 * time.Second
	keepaliveTime    = 30 * time.Second
	keepaliveTimeout = 10 * time.Second
)

// NewClient .
func NewClient(config *apiconfig.CalicoAPIConfig) (*clientv3.Client, error) {
	endpoints := strings.Split(config.Spec.EtcdConfig.EtcdEndpoints, ",")
	return clientv3.New(clientv3.Config{
		Endpoints:            endpoints,
		DialTimeout:          clientTimeout,
		DialKeepAliveTime:    keepaliveTime,
		DialKeepAliveTimeout: keepaliveTimeout,
		Context:              context.Background(),
	})
}
