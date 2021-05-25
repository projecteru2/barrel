package filesystem

import "path/filepath"

func (s FSStore) NetnsPath(ident string) string {
	return filepath.Join(s.root, ident)
}

func (s FSStore) OccupiedPath(ip string) string {
	return filepath.Join(s.root, ip+"-occupied")
}

func (s FSStore) CreatedPath(ip string) string {
	return filepath.Join(s.root, ip+"-createdby")
}
