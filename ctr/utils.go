package ctr

import (
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"

	"github.com/juju/errors"

	"github.com/projectcalico/libcalico-go/lib/backend/model"
	cnet "github.com/projectcalico/libcalico-go/lib/net"
)

var (
	initFuncs = make(map[uintptr]*InitUnit)
	// ErrCircularDependencies .
	ErrCircularDependencies = errors.New("circular dependencies")
	// ErrInited .
	ErrInited = errors.New("already inited")
)

// BlockHasAffinity .
func BlockHasAffinity(block *model.AllocationBlock, hostname string) bool {
	blockAffinity := block.Affinity
	return blockAffinity != nil && *blockAffinity == fmt.Sprintf("host:%s", hostname)
}

// BlockIsEmpty .
func BlockIsEmpty(block *model.AllocationBlock) bool {
	for _, val := range block.Allocations {
		if val != nil {
			return false
		}
	}
	return true
}

// InitHost .
func InitHost(dest *string) error {
	if *dest == "" {
		host, err := os.Hostname()
		if err != nil {
			return err
		}
		*dest = host
	}
	return nil
}

// InitUnit .
type InitUnit struct {
	ptr      uintptr
	requires []UnitDependency
	init     InitFunc
	inited   bool
}

// InitFunc .
type InitFunc func() error

// UnitDependency .
type UnitDependency func() *InitUnit

// InitUnits .
type InitUnits struct {
	units []*InitUnit
}

func (units *InitUnits) append(unit *InitUnit) {
	units.units = append(units.units, unit)
}

// Declare .
func (units *InitUnits) Declare(init InitFunc) *InitUnit {
	ptr := reflect.ValueOf(init).Pointer()
	unit := initFuncs[ptr]
	if unit == nil {
		unit = &InitUnit{
			ptr:  ptr,
			init: init,
		}
		initFuncs[ptr] = unit
	}
	units.append(unit)
	return unit
}

// Require .
func (units *InitUnits) Require(init InitFunc, deps ...UnitDependency) *InitUnit {
	ptr := reflect.ValueOf(init).Pointer()
	unit := initFuncs[ptr]
	if unit == nil {
		unit = &InitUnit{
			ptr:      ptr,
			requires: deps,
			init:     init,
		}
		initFuncs[ptr] = unit
	}
	units.append(unit)
	return unit
}

func (units *InitUnits) init() error {
	for _, unit := range units.units {
		if err := initUnit(unit, nil); err != nil {
			return err
		}
	}
	return nil
}

func initUnit(unit *InitUnit, ptrs []uintptr) (err error) {
	if unit.inited {
		return nil
	}
	defer func() {
		if err == nil {
			unit.inited = true
		}
	}()
	for _, ptr := range ptrs {
		if unit.ptr == ptr {
			return ErrCircularDependencies
		}
	}
	if len(unit.requires) > 0 {
		for _, dep := range unit.requires {
			if err := initUnit(dep(), append(ptrs, unit.ptr)); err != nil {
				return err
			}
		}
	}
	return unit.init()
}

// FormatBlock .
func FormatBlock(block *model.AllocationBlock) (result *FormattedBlock) {
	var (
		allocated   []cnet.IPNet
		unallocated []cnet.IPNet
	)
	_, mask, _ := cnet.ParseCIDR(block.CIDR.String())
	for idx, val := range block.Allocations {
		ipNet := *mask
		ipNet.IP = cnet.IncrementIP(cnet.IP{IP: block.CIDR.IP}, big.NewInt(int64(idx))).IP
		if val != nil {
			allocated = append(allocated, ipNet)
			continue
		}
		unallocated = append(unallocated, ipNet)
	}

	return &FormattedBlock{
		CIDR:           block.CIDR,
		Affinity:       block.Affinity,
		StrictAffinity: block.StrictAffinity,
		Allocated:      allocated,
		Unallocated:    unallocated,
		Attributes:     block.Attributes,
		Deleted:        block.Deleted,
	}
}

// FormattedBlock .
type FormattedBlock struct {
	CIDR           cnet.IPNet                  `json:"cidr"`
	Affinity       *string                     `json:"affinity"`
	StrictAffinity bool                        `json:"strictAffinity"`
	Allocated      []cnet.IPNet                `json:"allocated"`
	Unallocated    []cnet.IPNet                `json:"unallocated"`
	Attributes     []model.AllocationAttribute `json:"attributes"`
	Deleted        bool                        `json:"deleted"`
}

// Fprintlnf .
func Fprintlnf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stdout, format+"\n", args...)
}

// Ferrorlnf .
func Ferrorlnf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(os.Stderr, format+"\n", args...)
}

// Fprintln .
func Fprintln(val string) error {
	_, err := fmt.Fprint(os.Stdout, val+"\n")
	return err
}

// Ferrorln .
func Ferrorln(val string) error {
	_, err := fmt.Fprintf(os.Stderr, val+"\n")
	return err
}

// AllocationBlockHost we can't use Host method of AllocationBlock
func AllocationBlockHost(b *model.AllocationBlock) string {
	if b.Affinity != nil && strings.HasPrefix(*b.Affinity, "host:") {
		return strings.TrimPrefix(*b.Affinity, "host:")
	}
	return ""
}
