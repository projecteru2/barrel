package docker

import (
	"context"

	dockerTypes "github.com/docker/docker/api/types"
)

// Client .
type Client interface {
	ContainerList(context.Context, dockerTypes.ContainerListOptions) ([]dockerTypes.Container, error)
}
