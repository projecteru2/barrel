package resources

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestJsonObject(t *testing.T) {
	pathes := []string{
		"/data/biz/cluster01/container01/data",
		"/data/biz/cluster01/container01/log",
		"/data/biz/cluster01/container01",
		"/data/biz/cluster02/container01/data",
		"/data/biz/cluster02/container01/log",
		"/data/biz/cluster02/container01",
	}
	newPathes := minify(pathes)
	assert.Equal(t, 2, len(newPathes))
	assert.Equal(t, "/data/biz/cluster01/container01", newPathes[0])
	assert.Equal(t, "/data/biz/cluster02/container01", newPathes[1])
}
