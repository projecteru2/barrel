package errors

import (
	"fmt"

	"github.com/juju/errors"
)

type ErrorBuilder interface {
	WithField(field string, value interface{}) ErrorBuilder
	Error(string) error
	Errorf(string, ...interface{}) error
}

func WithField(field string, value interface{}) ErrorBuilder {
	return &errBuilder{
		field: field,
		value: value,
	}
}

func New(msg string) error {
	return errors.New(msg)
}

func Errorf(format string, args ...interface{}) error {
	return errors.Errorf(format, args...)
}

func Wrap(err error, another error) error {
	return errors.Wrap(err, another)
}

type errBuilder struct {
	prev  *errBuilder
	field string
	value interface{}
}

func (b *errBuilder) WithField(field string, value interface{}) ErrorBuilder {
	newBuilder := &errBuilder{field: field, value: value, prev: b}
	return newBuilder
}

func (b *errBuilder) Error(msg string) error {
	return errors.Errorf("%s, %s", msg, b.fieldString())
}

func (b *errBuilder) Errorf(format string, args ...interface{}) error {
	return errors.Errorf("%s, %s", fmt.Sprintf(format, args...), b.fieldString())
}

func (b *errBuilder) fieldString() string {
	if b.prev != nil {
		return fmt.Sprintf("%s, %s=%v", b.prev.fieldString(), b.field, b.value)
	}
	return fmt.Sprintf("%s=%v", b.field, b.value)
}
