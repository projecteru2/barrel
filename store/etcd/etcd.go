package etcd

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/projecteru2/barrel/store"

	"github.com/coreos/etcd/clientv3"
	"github.com/projectcalico/libcalico-go/lib/apiconfig"
)

const (
	cmpVersion = "version"
	cmpValue   = "value"

	clientTimeout    = 10 * time.Second
	keepaliveTime    = 30 * time.Second
	keepaliveTimeout = 10 * time.Second
)

var (
	errKeyIsBlank = errors.New("Key shouldn't be blank")
	errNoOps      = errors.New("No ops")
)

// Etcd .
type Etcd struct {
	cliv3 *clientv3.Client
}

// NewClient .
func NewClient(ctx context.Context, config *apiconfig.CalicoAPIConfig) (*Etcd, error) {
	endpoints := strings.Split(config.Spec.EtcdConfig.EtcdEndpoints, ",")
	cliv3, err := clientv3.New(clientv3.Config{
		Endpoints:            endpoints,
		DialTimeout:          clientTimeout,
		DialKeepAliveTime:    keepaliveTime,
		DialKeepAliveTimeout: keepaliveTimeout,
		Context:              ctx,
	})
	if err != nil {
		return nil, err
	}
	return &Etcd{cliv3}, nil
}

// Get .
func (e *Etcd) Get(ctx context.Context, decoder store.Decoder) (bool, error) {
	var (
		resp *clientv3.GetResponse
		err  error
	)
	if resp, err = e.cliv3.Get(ctx, decoder.Key()); err != nil {
		return false, err
	}
	if len(resp.Kvs) == 0 {
		return false, nil
	}
	kv := resp.Kvs[0]
	err = decoder.Decode(string(kv.Value))
	decoder.SetVersion(kv.Version)
	return err == nil, err
}

// Put save a key value
func (e *Etcd) Put(ctx context.Context, encoder store.Encoder) error {
	var (
		key = encoder.Key()
		val string
		err error
	)
	if key == "" {
		return errKeyIsBlank
	}
	if val, err = encoder.Encode(); err != nil {
		return err
	}
	_, err = e.cliv3.Put(ctx, key, val)
	return err
}

// Delete delete key
// returns true on delete count > 0
func (e *Etcd) Delete(ctx context.Context, encoder store.Encoder) (bool, error) {
	var (
		key  = encoder.Key()
		resp *clientv3.DeleteResponse
		err  error
	)
	if key == "" {
		return false, errKeyIsBlank
	}
	if resp, err = e.cliv3.Delete(ctx, key, clientv3.WithPrevKV()); err != nil {
		return false, err
	}
	return len(resp.PrevKvs) > 0, nil
}

// GetAndDelete delete key, and return value
// returns true on delete count > 0
func (e *Etcd) GetAndDelete(ctx context.Context, decoder store.Decoder) (bool, error) {
	var (
		key  = decoder.Key()
		resp *clientv3.DeleteResponse
		err  error
	)
	if key == "" {
		return false, errKeyIsBlank
	}
	if resp, err = e.cliv3.Delete(ctx, key, clientv3.WithPrevKV()); err != nil {
		return false, err
	}
	if len(resp.PrevKvs) == 0 {
		return false, nil
	}
	err = decoder.Decode(string(resp.PrevKvs[0].Value))
	return err == nil, err
}

// Update .
func (e *Etcd) Update(ctx context.Context, encoder store.Encoder) (bool, error) {
	var (
		value string
		err   error
		resp  *clientv3.TxnResponse
	)
	if value, err = encoder.Encode(); err != nil {
		return false, err
	}
	key := encoder.Key()
	prevVersion := encoder.Version()
	if resp, err = e.cliv3.Txn(
		ctx,
	).If(
		clientv3.Compare(clientv3.Version(key), "=", prevVersion),
	).Then(
		clientv3.OpPut(key, value),
	).Commit(); err != nil {
		return false, err
	}
	return resp.Succeeded, err
}

// PutMulti .
func (e *Etcd) PutMulti(ctx context.Context, encoders ...store.Encoder) error {
	data := make(map[string]string)
	for _, encoder := range encoders {
		var (
			key = encoder.Key()
			val string
			err error
		)
		if key == "" {
			return errKeyIsBlank
		}
		if val, err = encoder.Encode(); err != nil {
			return err
		}
		data[key] = val
	}
	_, err := e.batchPut(ctx, data, nil)
	return err
}

// BatchPut .
func (e *Etcd) batchPut(
	ctx context.Context,
	data map[string]string,
	limit map[string]map[string]string,
	opts ...clientv3.OpOption,
) (*clientv3.TxnResponse, error) {
	ops := []clientv3.Op{}
	failOps := []clientv3.Op{}
	conds := []clientv3.Cmp{}
	for key, val := range data {
		op := clientv3.OpPut(key, val, opts...)
		ops = append(ops, op)
		if v, ok := limit[key]; ok {
			for method, condition := range v {
				switch method {
				case cmpVersion:
					cond := clientv3.Compare(clientv3.Version(key), condition, 0)
					conds = append(conds, cond)
				case cmpValue:
					cond := clientv3.Compare(clientv3.Value(key), condition, val)
					failOps = append(failOps, clientv3.OpGet(key))
					conds = append(conds, cond)
				}
			}
		}
	}
	return e.doBatchOp(ctx, conds, ops, failOps)
}

func (e *Etcd) doBatchOp(ctx context.Context, conds []clientv3.Cmp, ops, failOps []clientv3.Op) (*clientv3.TxnResponse, error) {
	if len(ops) == 0 {
		return nil, errNoOps
	}

	const txnLimit = 125
	count := len(ops) / txnLimit // stupid etcd txn, default limit is 128
	tail := len(ops) % txnLimit
	length := count
	if tail != 0 {
		length++
	}

	resps := make([]*clientv3.TxnResponse, length)
	errs := make([]error, length)

	wg := sync.WaitGroup{}
	doOp := func(index int, ops []clientv3.Op) {
		defer wg.Done()
		txn := e.cliv3.Txn(ctx)
		if len(conds) != 0 {
			txn = txn.If(conds...)
		}
		resp, err := txn.Then(ops...).Else(failOps...).Commit()
		resps[index] = resp
		errs[index] = err
	}

	if tail != 0 {
		wg.Add(1)
		go doOp(length-1, ops[count*txnLimit:])
	}

	for i := 0; i < count; i++ {
		wg.Add(1)
		go doOp(i, ops[i*txnLimit:(i+1)*txnLimit])
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return nil, err
		}
	}

	if len(resps) == 0 {
		return &clientv3.TxnResponse{}, nil
	}

	resp := resps[0]
	for i := 1; i < len(resps); i++ {
		resp.Succeeded = resp.Succeeded && resps[i].Succeeded
		resp.Responses = append(resp.Responses, resps[i].Responses...)
	}
	return resp, nil
}
