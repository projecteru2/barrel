package vessel

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	barrelEtcd "github.com/projecteru2/barrel/etcd"
	etcdStore "github.com/projecteru2/barrel/store/etcd"

	// storeMocks "github.com/projecteru2/barrel/store/mocks"
	"github.com/projecteru2/barrel/types"
	// "github.com/projecteru2/barrel/vessel/codecs"
	"github.com/projecteru2/barrel/vessel/mocks"
)

func TestAllocInuseFixedIP(t *testing.T) {
	stor := etcdStore.NewEtcdStore(barrelEtcd.NewEmbedEtcd(t).Client())

	calicoIPAllocator := mocks.CalicoIPAllocator{}
	allocator := fixedIPAllocator{
		CalicoIPAllocator: &calicoIPAllocator,
		store:             stor,
	}
	calicoIPAllocator.On("AllocIP", mock.Anything, mock.Anything).Return(nil)

	ip := types.IP{
		Address: "10.10.10.10",
		PoolID:  "poolID",
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(6)*time.Second)
	defer cancel()

	assert.NoError(t, allocator.AllocFixedIP(ctx, ip))
	assert.NoError(t, allocator.AssignFixedIP(ctx, ip))
	assert.Equal(t, types.ErrIPInUse, allocator.AllocFixedIP(ctx, ip))
}

func TestAllocFixedIPSuccess(t *testing.T) {
	stor := etcdStore.NewEtcdStore(barrelEtcd.NewEmbedEtcd(t).Client())

	calicoIPAllocator := mocks.CalicoIPAllocator{}
	allocator := fixedIPAllocator{
		CalicoIPAllocator: &calicoIPAllocator,
		store:             stor,
	}

	calicoIPAllocator.On("AllocIP", mock.Anything, mock.Anything).Return(nil)
	calicoIPAllocator.On("UnallocIP", mock.Anything, mock.Anything).Return(nil)

	ctx := context.Background()
	ip := types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	}
	assert.Equal(t, types.ErrFixedIPNotAllocated, allocator.AssignFixedIP(ctx, ip))

	assert.NoError(t, allocator.AllocFixedIP(ctx, ip))
	assert.NoError(t, allocator.AssignFixedIP(ctx, ip))
	assert.NoError(t, allocator.UnassignFixedIP(ctx, ip))
	assert.NoError(t, allocator.UnallocFixedIP(ctx, ip))

	assert.Equal(t, types.ErrFixedIPNotAllocated, allocator.AssignFixedIP(ctx, ip))
}

func TestAllocAllocatedFixedIP(t *testing.T) {
	stor := etcdStore.NewEtcdStore(barrelEtcd.NewEmbedEtcd(t).Client())

	calicoIPAllocator := mocks.CalicoIPAllocator{}
	calicoIPAllocator.On("AllocIP", mock.Anything, mock.Anything).Return(nil)

	allocator := fixedIPAllocator{
		CalicoIPAllocator: &calicoIPAllocator,
		store:             stor,
	}
	ip := types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	}
	ctx := context.Background()

	assert.NoError(t, allocator.AllocFixedIP(ctx, ip))
	assert.NoError(t, allocator.AllocFixedIP(ctx, ip))
}

func TestBorrowIPAndReturn(t *testing.T) {
	stor := etcdStore.NewEtcdStore(barrelEtcd.NewEmbedEtcd(t).Client())

	calicoIPAllocator := mocks.CalicoIPAllocator{}
	calicoIPAllocator.On("AllocIP", mock.Anything, mock.Anything).Return(nil)
	calicoIPAllocator.On("UnallocIP", mock.Anything, mock.Anything).Return(nil)

	allocator := fixedIPAllocator{
		CalicoIPAllocator: &calicoIPAllocator,
		store:             stor,
	}
	ip := types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	}
	ctx := context.Background()

	assert.NoError(t, allocator.AllocFixedIP(ctx, ip))
	container1 := types.Container{
		ID:       "diucat",
		HostName: "dev-1",
	}
	assert.NoError(t, allocator.BorrowFixedIP(ctx, ip, container1))
	assert.Equal(t, types.ErrFixedIPHasBorrower, allocator.UnallocFixedIP(ctx, ip))

	assert.NoError(t, allocator.ReturnFixedIP(ctx, ip, container1))
	assert.NoError(t, allocator.UnallocFixedIP(ctx, ip))
}

func TestBorrowIPTwiceForSingleContainer(t *testing.T) {
	stor := etcdStore.NewEtcdStore(barrelEtcd.NewEmbedEtcd(t).Client())

	calicoIPAllocator := mocks.CalicoIPAllocator{}
	calicoIPAllocator.On("AllocIP", mock.Anything, mock.Anything).Return(nil)
	calicoIPAllocator.On("UnallocIP", mock.Anything, mock.Anything).Return(nil)

	allocator := fixedIPAllocator{
		CalicoIPAllocator: &calicoIPAllocator,
		store:             stor,
	}
	ip := types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	}
	ctx := context.Background()

	assert.NoError(t, allocator.AllocFixedIP(ctx, ip))

	container1 := types.Container{
		ID:       "diucat",
		HostName: "dev-1",
	}

	assert.NoError(t, allocator.BorrowFixedIP(ctx, ip, container1))
	assert.NoError(t, allocator.BorrowFixedIP(ctx, ip, container1))
	assert.NoError(t, allocator.ReturnFixedIP(ctx, ip, container1))
	assert.NoError(t, allocator.UnallocFixedIP(ctx, ip))
}

func TestBorrowIPFromMultipleContainer(t *testing.T) {
	stor := etcdStore.NewEtcdStore(barrelEtcd.NewEmbedEtcd(t).Client())

	calicoIPAllocator := mocks.CalicoIPAllocator{}
	calicoIPAllocator.On("AllocIP", mock.Anything, mock.Anything).Return(nil)
	calicoIPAllocator.On("UnallocIP", mock.Anything, mock.Anything).Return(nil)

	allocator := fixedIPAllocator{
		CalicoIPAllocator: &calicoIPAllocator,
		store:             stor,
	}
	ip := types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	}
	ctx := context.Background()

	assert.NoError(t, allocator.AllocFixedIP(ctx, ip))

	container1 := types.Container{
		ID:       "diucat",
		HostName: "dev-1",
	}

	container2 := types.Container{
		ID:       "diucat",
		HostName: "dev-2",
	}
	assert.NoError(t, allocator.BorrowFixedIP(ctx, ip, container1))
	assert.NoError(t, allocator.BorrowFixedIP(ctx, ip, container2))
	assert.NoError(t, allocator.ReturnFixedIP(ctx, ip, container1))
	assert.Equal(t, types.ErrFixedIPHasBorrower, allocator.UnallocFixedIP(ctx, ip))
	assert.NoError(t, allocator.ReturnFixedIP(ctx, ip, container2))
	assert.NoError(t, allocator.UnallocFixedIP(ctx, ip))
}

func TestBorrowIPFromMultipleContainerAndReturnTwice(t *testing.T) {
	stor := etcdStore.NewEtcdStore(barrelEtcd.NewEmbedEtcd(t).Client())

	calicoIPAllocator := mocks.CalicoIPAllocator{}
	calicoIPAllocator.On("AllocIP", mock.Anything, mock.Anything).Return(nil)
	calicoIPAllocator.On("UnallocIP", mock.Anything, mock.Anything).Return(nil)

	allocator := fixedIPAllocator{
		CalicoIPAllocator: &calicoIPAllocator,
		store:             stor,
	}
	ip := types.IP{
		PoolID:  "poolID",
		Address: "10.10.10.10",
	}
	ctx := context.Background()

	assert.NoError(t, allocator.AllocFixedIP(ctx, ip))

	container1 := types.Container{
		ID:       "diucat",
		HostName: "dev-1",
	}

	container2 := types.Container{
		ID:       "diucat",
		HostName: "dev-2",
	}
	assert.NoError(t, allocator.BorrowFixedIP(ctx, ip, container1))
	assert.NoError(t, allocator.BorrowFixedIP(ctx, ip, container2))
	assert.NoError(t, allocator.ReturnFixedIP(ctx, ip, container1))
	// should not have error
	assert.NoError(t, allocator.ReturnFixedIP(ctx, ip, container1))
	assert.Equal(t, types.ErrFixedIPHasBorrower, allocator.UnallocFixedIP(ctx, ip))
	assert.NoError(t, allocator.ReturnFixedIP(ctx, ip, container2))
	assert.NoError(t, allocator.UnallocFixedIP(ctx, ip))
}
