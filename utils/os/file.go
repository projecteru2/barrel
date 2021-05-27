package os

import (
	"os"
)

// FileExists .
func FileExists(path string) (bool, error) {
	if _, err := defaultOS.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}
