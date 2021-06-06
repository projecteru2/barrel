package filesystem

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func (s FSStore) netnsPath(ident string) string {
	return filepath.Join(s.root, ident)
}

func (s FSStore) tenantPath(ip string) string {
	return filepath.Join(s.root, ip+"-tenant")
}

func (s FSStore) ownerPath(ip string) string {
	return filepath.Join(s.root, ip+"-owner")
}

func (s FSStore) flockPath(ip string) string {
	return filepath.Join(s.root, ip+"-flock")
}

func assertFileNotExists(pathfiles ...string) (err error) {
	for _, pathfile := range pathfiles {
		if _, err := os.Stat(pathfile); err == nil {
			return errors.Errorf("file remains existing: %s", pathfile)
		}
	}
	return
}
