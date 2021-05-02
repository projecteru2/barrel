package resources

import (
	"regexp"

	"github.com/projecteru2/barrel/vessel"
)

func NewResourceManager(regexps []*regexp.Regexp, helper vessel.Helper) vessel.ResourceManager {
	return &manager{
		mountResourceRegex: regexps,
	}
}

type manager struct {
	mountResourceRegex []*regexp.Regexp
	helper             vessel.Helper
}

func (m *manager) MountResource(path string) (vessel.Resource, bool) {
	for _, reg := range m.mountResourceRegex {
		if reg.Match([]byte(path)) {
			return &mountResource{
				path: path,
			}, true
		}
	}
	return nil, false
}

func (m *manager) ContainerIPResource(containerID string) vessel.Resource {
	return &ipResource{
		containerID:  containerID,
		vesselHelper: m.helper,
	}
}
