package resources

import (
	"context"

	"os"
)

type mountResource struct {
	path string
}

func (r *mountResource) Recycle(ctx context.Context) error {
	return os.RemoveAll(r.path)
}
