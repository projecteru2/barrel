package store

import (
	"context"

	"github.com/juju/errors"
)

var (
	// ErrKVNotExists .
	ErrKVNotExists = errors.New("KV not exists")

	// ErrUnexpectedTxnResp .
	ErrUnexpectedTxnResp = errors.New("unexpected txn resp")
)

// IsNotExists .
func IsNotExists(err error) bool {
	return err == ErrKVNotExists
}

// ErrButOtherThenKVUnexistsErr .
func ErrButOtherThenKVUnexistsErr(err error) bool {
	return err != nil && !IsNotExists(err)
}

// Codec .
type Codec interface {
	Key() string
	Encode() (string, error)
	Decode(string) error
	Version() int64
	SetVersion(int64)
}

// UpdateCodec .
type UpdateCodec interface {
	Codec
	Retry() bool
}

// Store .
type Store interface {
	Get(ctx context.Context, codec Codec) error
	Put(ctx context.Context, codec Codec) error
	Delete(ctx context.Context, codec Codec) error
	GetAndDelete(ctx context.Context, codec Codec) error
	UpdateElseGet(ctx context.Context, codec Codec) (bool, error)
	Update(ctx context.Context, codec UpdateCodec) (bool, error)
	PutMulti(ctx context.Context, codec ...Codec) error
}
