package codecs

import (
	"testing"

	"github.com/projecteru2/barrel/types"
	"github.com/stretchr/testify/assert"
)

func TestCodec(t *testing.T) {
	codec := &IPInfoCodec{
		IPInfo: &types.IPInfo{
			PoolID:  "pool-1",
			Address: "127.0.0.1",
			Attrs: &types.IPAttributes{
				Borrowers: []types.Container{
					{
						ID:       "container-01",
						HostName: "host-01",
					},
				},
			},
		},
	}
	json, err := codec.Encode()
	assert.NoError(t, err)
	decoder := &IPInfoCodec{IPInfo: &types.IPInfo{}}
	assert.NoError(t, decoder.Decode(json))
	assert.NotNil(t, decoder.IPInfo.Attrs)
	assert.NotNil(t, decoder.IPInfo.Attrs.Borrowers)
	assert.Equal(t, 1, len(decoder.IPInfo.Attrs.Borrowers))
	assert.Equal(t, "container-01", decoder.IPInfo.Attrs.Borrowers[0].ID)
	assert.Equal(t, "host-01", decoder.IPInfo.Attrs.Borrowers[0].HostName)
}
