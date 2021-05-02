package vessel

import (
	"context"
)

type Resource interface {
	Recycle(context.Context) error
}

type ResourceManager interface {
	MountResource(string) (Resource, bool)
	ContainerIPResource(string) Resource
}
