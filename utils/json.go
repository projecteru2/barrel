package utils

import (
	"encoding/json"
	"reflect"

	"github.com/juju/errors"
)

// Any .
type Any interface {
	Null() bool
	String() string
	StringValue() (string, bool)
	IntValue() (int64, bool)
	FloatValue() (float64, bool)
	BoolValue() (bool, bool)
	ArrayValue() (Array, bool)
	ObjectValue() (Object, bool)
	// return raw value
	RawValue() interface{}
}

// Array .
type Array interface {
	Any() Any
	Size() int
	// force nil check, and panics when index over range
	Get(int) Any
	Set(int, Any)
	Add(...Any)
}

// Object .
type Object interface {
	Any() Any
	Keys() []string
	Has(string) bool
	// force nil check
	Get(string) (Any, bool)
	Set(string, Any)
	Del(string)
}

// Unmarshal .
func Unmarshal(data []byte) (Any, error) {
	var src interface{}
	if err := json.Unmarshal(data, &src); err != nil {
		return nil, err
	}
	return newNode(src)
}

// UnmarshalObject .
func UnmarshalObject(data []byte) (Object, error) {
	var src interface{}
	if err := json.Unmarshal(data, &src); err != nil {
		return nil, err
	}
	if node, err := newNode(src); err != nil {
		return nil, err
	} else if object, ok := node.ObjectValue(); !ok {
		return nil, errors.Errorf("not object, value=%s", node.String())
	} else {
		return object, nil
	}
}

// UnmarshalArray .
func UnmarshalArray(data []byte) (Array, error) {
	var src interface{}
	if err := json.Unmarshal(data, &src); err != nil {
		return nil, err
	}
	if node, err := newNode(src); err != nil {
		return nil, err
	} else if array, ok := node.ArrayValue(); !ok {
		return nil, errors.Errorf("not array, value=%s", node.String())
	} else {
		return array, nil
	}
}

// Marshal .
func Marshal(obj Any) ([]byte, error) {
	return json.Marshal(obj.RawValue())
}

// NewNullNode .
func NewNullNode() Any {
	return newNullNode()
}

// NewObjectNode .
func NewObjectNode() Object {
	return createObjectNode()
}

// NewArrayNode .
func NewArrayNode() Array {
	return createArrayNode()
}

// NewStringNode .
func NewStringNode(str string) Any {
	return newStringNode(str)
}

// NewBoolNode .
func NewBoolNode(value bool) Any {
	return newBoolNode(value)
}

// NewIntNode .
func NewIntNode(i int64) Any {
	return newIntNode(i)
}

// NewFloatNode .
func NewFloatNode(f float64) Any {
	return newFloatNode(f)
}

func newNode(src interface{}) (Any, error) {
	if src == nil {
		return newNullNode(), nil
	}
	if str, ok := src.(string); ok {
		return newStringNode(str), nil
	}
	if f, ok := src.(float64); ok {
		return newNumberNode(f), nil
	}
	if array, ok := src.([]interface{}); ok {
		var (
			arr Array
			err error
		)
		if arr, err = newArrayNode(array); err != nil {
			return nil, err
		}
		return arr.Any(), nil
	}
	if objectMapping, ok := src.(map[string]interface{}); ok {
		var (
			object Object
			err    error
		)
		if object, err = newObjectNode(objectMapping); err != nil {
			return nil, err
		}
		return object.Any(), nil
	}
	if boolean, ok := src.(bool); ok {
		return newBoolNode(boolean), nil
	}
	return nil, errors.Errorf("Unsupport type conversion: %v", reflect.TypeOf(src))
}
