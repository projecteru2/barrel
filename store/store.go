package store

import "context"

// Encoder .
type Encoder interface {
	Key() string
	Encode() (string, error)
	Version() int64
}

// Decoder .
type Decoder interface {
	Key() string
	Decode(string) error
	SetVersion(int64)
}

// Codec .
type Codec interface {
	Encoder
	Decoder
}

// Store .
type Store interface {
	Get(ctx context.Context, decoder Decoder) (bool, error)
	Put(ctx context.Context, encoder Encoder) error
	Delete(ctx context.Context, encoder Encoder) (bool, error)
	GetAndDelete(ctx context.Context, decoder Decoder) (bool, error)
	Update(ctx context.Context, encoder Encoder) (bool, error)
	PutMulti(ctx context.Context, encoders ...Encoder) error
}
