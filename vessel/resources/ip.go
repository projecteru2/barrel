package resources

import (
	"context"

	"github.com/projecteru2/barrel/vessel"
)

type ipResource struct {
	containerID  string
	vesselHelper vessel.Helper
}

func (r *ipResource) Recycle(ctx context.Context) error {
	return r.vesselHelper.ReleaseContainerAddresses(ctx, r.containerID)
}
