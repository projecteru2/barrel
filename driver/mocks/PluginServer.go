// Code generated by mockery v0.0.0-dev. DO NOT EDIT.

package mocks

import (
	ipam "github.com/docker/go-plugins-helpers/ipam"
	mock "github.com/stretchr/testify/mock"

	network "github.com/docker/go-plugins-helpers/network"
)

// PluginServer is an autogenerated mock type for the PluginServer type
type PluginServer struct {
	mock.Mock
}

// ServeIpam provides a mock function with given fields: _a0
func (_m *PluginServer) ServeIpam(_a0 ipam.Ipam) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(ipam.Ipam) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// ServeNetwork provides a mock function with given fields: _a0
func (_m *PluginServer) ServeNetwork(_a0 network.Driver) error {
	ret := _m.Called(_a0)

	var r0 error
	if rf, ok := ret.Get(0).(func(network.Driver) error); ok {
		r0 = rf(_a0)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}