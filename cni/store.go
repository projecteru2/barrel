package cni

import "context"

func (w Wrapper) GetNetEndpoint(ctx context.Context, ID, IPv4 string) (netEP *NetEndpoint, err error) {
	return
}

func (w Wrapper) CreateNetEndpoint(ctx context.Context, netEP NetEndpoint) (err error) {
	return
}

func (w Wrapper) UpdateNetEndpoint(ctx context.Context, netEP NetEndpoint) (err error) {
	return
}
