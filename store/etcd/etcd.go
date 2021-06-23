package etcd

import (
	"context"
	"sync"

	"github.com/coreos/etcd/clientv3"
	"github.com/juju/errors"

	"github.com/projecteru2/barrel/store"
)

const (
	cmpVersion = "version"
	cmpValue   = "value"
)

var (
	errKeyIsBlank = errors.New("Key shouldn't be blank")
	errNoOps      = errors.New("No ops")
)

type etcdStore struct {
	cli clientv3.KV
}

// NewEtcdStore .
func NewEtcdStore(cli *clientv3.Client) store.Store {
	return &etcdStore{cli}
}

// Get .
func (e *etcdStore) Get(ctx context.Context, codec store.Codec) error {
	var (
		resp *clientv3.GetResponse
		err  error
	)
	if resp, err = e.cli.Get(ctx, codec.Key()); err != nil {
		return err
	}
	if len(resp.Kvs) == 0 {
		return store.ErrKVNotExists
	}
	kv := resp.Kvs[0]
	codec.SetVersion(kv.Version)
	return codec.Decode(string(kv.Value))
}

// Put save a key value
func (e *etcdStore) Put(ctx context.Context, codec store.Codec) error {
	var (
		key  = codec.Key()
		val  string
		err  error
		resp *clientv3.PutResponse
	)
	if key == "" {
		return errKeyIsBlank
	}
	if val, err = codec.Encode(); err != nil {
		return err
	}
	if resp, err = e.cli.Put(ctx, key, val, clientv3.WithPrevKV()); err != nil {
		return err
	}
	if resp.PrevKv != nil {
		codec.SetVersion(resp.PrevKv.Version + 1)
	} else {
		codec.SetVersion(0)
	}
	return nil
}

// Delete delete key
// returns true on delete count > 0
func (e *etcdStore) Delete(ctx context.Context, codec store.Codec) error {
	var (
		key  = codec.Key()
		resp *clientv3.DeleteResponse
		err  error
	)
	if key == "" {
		return errKeyIsBlank
	}
	if resp, err = e.cli.Delete(ctx, key, clientv3.WithPrevKV()); err != nil {
		return err
	}
	if len(resp.PrevKvs) == 0 {
		return store.ErrKVNotExists
	}
	return nil
}

// GetAndDelete delete key, and return value
// returns true on delete count > 0
func (e *etcdStore) GetAndDelete(ctx context.Context, codec store.Codec) error {
	var (
		key  = codec.Key()
		resp *clientv3.DeleteResponse
		err  error
	)
	if key == "" {
		return errKeyIsBlank
	}
	if resp, err = e.cli.Delete(ctx, key, clientv3.WithPrevKV()); err != nil {
		return err
	}
	if len(resp.PrevKvs) == 0 {
		return store.ErrKVNotExists
	}
	codec.SetVersion(0)
	return codec.Decode(string(resp.PrevKvs[0].Value))
}

// Update .
func (e *etcdStore) UpdateElseGet(ctx context.Context, codec store.Codec) (bool, error) {
	var (
		value string
		err   error
		resp  *clientv3.TxnResponse
	)
	if value, err = codec.Encode(); err != nil {
		return false, err
	}
	key := codec.Key()
	prevVersion := codec.Version()

	if resp, err = e.cli.Txn(
		ctx,
	).If(
		clientv3.Compare(clientv3.Version(key), "=", prevVersion),
	).Then(
		clientv3.OpPut(key, value, clientv3.WithPrevKV()),
	).Else(
		clientv3.OpGet(key),
	).Commit(); err != nil {
		return false, err
	}
	if len(resp.Responses) != 1 {
		return resp.Succeeded, store.ErrUnexpectedTxnResp
	}

	response := resp.Responses[0]
	if resp.Succeeded {
		r := response.GetResponsePut()
		if r == nil {
			return true, store.ErrUnexpectedTxnResp
		}
		if r.PrevKv == nil {
			return true, store.ErrKVNotExists
		}
		codec.SetVersion(r.PrevKv.Version + 1)
		return true, nil
	}

	r := response.GetResponseRange()
	if r == nil {
		return false, store.ErrUnexpectedTxnResp
	}
	if r.Count == 0 {
		return false, store.ErrKVNotExists
	}
	kv := r.Kvs[0]
	codec.SetVersion(kv.Version)
	return false, codec.Decode(string(kv.Value))
}

// Update .
func (e *etcdStore) Update(ctx context.Context, codec store.UpdateCodec) (bool, error) {
	for {
		var (
			succeeded bool
			err       error
		)
		if succeeded, err = e.UpdateElseGet(ctx, codec); err != nil {
			return succeeded, err
		}
		if succeeded {
			return true, nil
		}
		if !codec.Retry() {
			return false, nil
		}
	}
}

// PutMulti .
func (e *etcdStore) PutMulti(ctx context.Context, codecs ...store.Codec) error {
	data := make(map[string]string)
	for _, encoder := range codecs {
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
func (e *etcdStore) batchPut(
	ctx context.Context,
	data map[string]string,
	limit map[string]map[string]string,
	opts ...clientv3.OpOption,
) (*clientv3.TxnResponse, error) {

	var ops, failOps []clientv3.Op
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

func (e *etcdStore) doBatchOp(
	ctx context.Context,
	conds []clientv3.Cmp,
	ops, failOps []clientv3.Op,
) (*clientv3.TxnResponse, error) {
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
		resp, err := e.cli.Txn(
			ctx,
		).If(
			conds...,
		).Then(
			ops...,
		).Else(
			failOps...,
		).Commit()
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
