package os

import (
	"io/fs"
	"os"
)

// OS .
type OS interface {
	Stat(name string) (fs.FileInfo, error)
}

var defaultOS OS = goOS{}

// Mock will replace os implementation with given parameter
// after the test, reset by call the given function result
func Mock(os OS) func() {
	old := defaultOS
	defaultOS = os
	return func() {
		defaultOS = old
	}
}

type goOS struct{}

// Stat .
func (i goOS) Stat(name string) (fs.FileInfo, error) {
	return os.Stat(name)
}
