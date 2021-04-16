package resources

import (
	"os"
	"strings"

	"github.com/projecteru2/barrel/utils"
)

var resPaths []string

// Init .
func Init(paths []string) {
	resPaths = paths
}

// RecycleResources .
func RecycleResources(logger utils.Logger, source string) error {
	for _, path := range resPaths {
		if strings.HasPrefix(source, path) {
			logger.Infof("remove source, path = %s", source)
			return os.RemoveAll(source)
		}
	}
	return nil
}
