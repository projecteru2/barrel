package etcd

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	barrelEtcd "github.com/projecteru2/barrel/etcd"
	"github.com/projecteru2/barrel/store"
	"github.com/projecteru2/barrel/types"
	"github.com/projecteru2/barrel/vessel/codecs"
)

func TestEtcdOperation(t *testing.T) {

	server := barrelEtcd.NewEmbedEtcd(t)
	cli := server.Client()
	stor := NewEtcdStore(cli)

	var (
		ipInfo      = types.IPInfo{Address: "127.0.0.1", PoolID: "test-pool"}
		ipInfoCodec = &codecs.IPInfoCodec{IPInfo: &ipInfo}
	)

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(6)*time.Second)
	defer cancel()
	err := stor.Put(ctx, ipInfoCodec)

	assert.NoError(t, err, "Put object error")

	err = stor.Get(ctx, ipInfoCodec)
	assert.Equal(t, err, store.ErrKVNotExists)

	t.Logf("Version = %v", ipInfoCodec.Version())
}
