package fake

import (
	"github.com/projecteru2/barrel/driver"
	"github.com/projecteru2/barrel/driver/mocks"
)

// NewFakeCalicoPlugin .
func NewFakeCalicoPlugin() driver.AddressManager {
	m := &mocks.AddressManager{}
	// TODO add methods here
	// m.On("xxx", a, b).Retrun("bb")
	return m
}
