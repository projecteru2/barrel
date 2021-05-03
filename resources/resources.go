package resources

import (
	"os"
	"sort"
	"strings"

	log "github.com/sirupsen/logrus"
)

var resPathPrefixes []string

// Init .
func Init(pathPrefixes []string) {
	resPathPrefixes = pathPrefixes
}

// RecycleResources .
func RecycleMounts(paths []string) {
	paths = minify(paths)

	for _, path := range paths {
		if !matchPrefix(path) {
			log.WithField("path", path).Info("Path not matching resource path prefixes")
			continue
		}
		if err := os.RemoveAll(path); err != nil {
			log.WithError(err).WithField("path", path).Error("Remove mount path error")
			continue
		}
		log.WithField("path", path).Info("Remove mount path success")
	}
}

func matchPrefix(path string) bool {
	for _, prefix := range resPathPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

func minify(paths []string) []string {
	if len(paths) == 0 {
		return paths
	}
	var result []string
	sort.Strings(paths)
	curr := paths[0]
	result = append(result, curr)
	for _, p := range paths[1:] {
		if strings.HasPrefix(p, curr) {
			continue
		}
		curr = p
		result = append(result, curr)
	}
	return result
}
