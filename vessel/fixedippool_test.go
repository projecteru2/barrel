package vessel

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/projecteru2/barrel/store"
	storeMocks "github.com/projecteru2/barrel/store/mocks"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/utils"
	"github.com/projecteru2/barrel/vessel/codec"
	"github.com/projecteru2/barrel/vessel/mocks"
)

func TestAllocFixedIPSuccess(t *testing.T) {
	calicoIPAllocator := mocks.CalicoIPAllocator{}
	stor := storeMocks.Store{}

	allocator := newFixedIPAllocatorImpl(&calicoIPAllocator, &stor)

	stor.On("Get", mock.Anything, mock.Anything).Return(false, nil)
	calicoIPAllocator.On("AllocIP", mock.Anything, mock.Anything).Return(nil)
	stor.On("Put", mock.Anything, mock.Anything).Return(nil)

	err := allocator.AllocFixedIP(context.Background(), types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	})
	assert.NoError(t, err)
}

func TestAllocAllocatedFixedIP(t *testing.T) {
	calicoIPAllocator := mocks.CalicoIPAllocator{}
	stor := storeMocks.Store{}

	allocator := newFixedIPAllocatorImpl(&calicoIPAllocator, &stor)

	stor.On("Get", mock.Anything, mock.Anything).Return(
		func(_ context.Context, decoder store.Decoder) bool {
			ipInfo := types.IPInfo{
				Address: "10.10.10.10",
				PoolID:  "poolID",
			}
			ipInfoCodec := &codec.IPInfoCodec{IPInfo: &ipInfo}
			ipInfoJSON, err := ipInfoCodec.Encode()
			assert.NoError(t, err)

			err = decoder.Decode(ipInfoJSON)
			assert.NoError(t, err)
			return true
		},
		nil,
	)

	err := allocator.AllocFixedIP(context.Background(), types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	})
	assert.NoError(t, err)
}

func TestAllocInuseFixedIP(t *testing.T) {
	calicoIPAllocator := mocks.CalicoIPAllocator{}
	stor := storeMocks.Store{}

	allocator := newFixedIPAllocatorImpl(&calicoIPAllocator, &stor)

	stor.On("Get", mock.Anything, mock.Anything).Return(
		func(_ context.Context, decoder store.Decoder) bool {
			ipInfo := types.IPInfo{
				Address: "10.10.10.10",
				PoolID:  "poolID",
				Status:  types.IPStatusInUse,
			}
			ipInfoCodec := &codec.IPInfoCodec{IPInfo: &ipInfo}
			ipInfoJSON, err := ipInfoCodec.Encode()
			assert.NoError(t, err)

			err = decoder.Decode(ipInfoJSON)
			assert.NoError(t, err)
			return true
		},
		nil,
	)

	err := allocator.AllocFixedIP(context.Background(), types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	})
	assert.Error(t, err)
}

func newFixedIPAllocatorImpl(calicoIPAllocator CalicoIPAllocator, stor store.Store) FixedIPAllocatorImpl {
	return FixedIPAllocatorImpl{
		CalicoIPAllocator: calicoIPAllocator,
		Store:             stor,
		LoggerFactory:     utils.NewObjectLogger("FixedIPAllocatorImpl"),
	}
}
