package filesystem

import "path/filepath"

func (s FSStore) NetnsPath(ident string) string {
	return filepath.Join(s.root, ident)
}

func (s FSStore) TenantPath(ip string) string {
	return filepath.Join(s.root, ip+"-tenant")
}

func (s FSStore) OwnerPath(ip string) string {
	return filepath.Join(s.root, ip+"-owner")
}

func (s FSStore) FlockPath(ip string) string {
	return filepath.Join(s.root, ip+"-flock")
}
