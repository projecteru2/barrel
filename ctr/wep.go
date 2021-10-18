package ctr

import (
	"context"
	"net"

	"github.com/juju/errors"

	v3 "github.com/projectcalico/libcalico-go/lib/apis/v3"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
	"github.com/projectcalico/libcalico-go/lib/options"
	log "github.com/sirupsen/logrus"

	"github.com/projecteru2/barrel/types"
)

// ListWorkloadEndpoints .
func (c *Ctr) ListWorkloadEndpoints(ctx context.Context, namespace string, poolname string) ([]v3.WorkloadEndpoint, error) {
	ipPool, err := c.calico.IPPools().Get(ctx, poolname, options.GetOptions{})
	if err != nil {
		return nil, err
	}

	weps, err := c.calico.WorkloadEndpoints().List(ctx, options.ListOptions{
		Namespace: namespace,
	})
	if err != nil {
		return nil, err
	}

	var poolWeps []v3.WorkloadEndpoint
	for _, wep := range weps.Items {
		if belongsToPool(wep, ipPool) {
			poolWeps = append(poolWeps, wep)
		}
	}
	return poolWeps, nil
}

// RecycleWorkloadEndpoint .
func (c *Ctr) RecycleWorkloadEndpoint(ctx context.Context, wepName string, poolname string) error {
	wepList, err := c.calico.WorkloadEndpoints().List(ctx, options.ListOptions{Name: wepName})
	if err != nil {
		return err
	}
	var workloadEndpoint *v3.WorkloadEndpoint
	for _, wep := range wepList.Items {
		if wep.Name == wepName {
			copyWep := wep
			workloadEndpoint = &copyWep
		}
	}
	if workloadEndpoint == nil {
		return errors.New("WorkloadEndpoint not found")
	}

	if _, err := c.calico.WorkloadEndpoints().Delete(
		ctx,
		workloadEndpoint.Namespace,
		workloadEndpoint.Name,
		options.DeleteOptions{},
	); err != nil {
		return err
	}
	log.Info("Recycle IP addresses bound with WorkloadEndpoint")
	for _, ipCidr := range workloadEndpoint.Spec.IPNetworks {
		if ip, _, err := net.ParseCIDR(ipCidr); err != nil {
			log.WithError(err).Errorf("Parse cidr %s error, failed to recycle ip", ipCidr)
		} else if err := c.UnassignFixedIP(ctx, types.IP{
			PoolID:  poolname,
			Address: ip.String(),
		}, false); err != nil {
			log.WithError(err).Errorf("Failed to recycle ip %s", ipCidr)
		}
	}
	return nil
}

func belongsToPool(wep v3.WorkloadEndpoint, pool *v3.IPPool) bool {
	_, poolIPNet, err := cnet.ParseCIDROrIP(pool.Spec.CIDR)
	if err != nil {
		return false
	}
	for _, addr := range wep.Spec.IPNetworks {
		ip := cnet.ParseIP(addr)

		if ip == nil {
			continue
		}

		if poolIPNet.Contains(ip.IP) {
			return true
		}
	}
	return false
}
