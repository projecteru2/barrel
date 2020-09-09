package types

import "time"

// DockerConfig .
type DockerConfig struct {
	DockerdSocketPath string
	DialTimeout       time.Duration
	Driver            string
	DockerGid         int64
	Hosts             []string
	CertFile          string
	KeyFile           string
}
