package codec

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMarshal(t *testing.T) {
	var bytes []byte
	assert.Equal(t, "", string(bytes), "this shouldn't happen")
}
