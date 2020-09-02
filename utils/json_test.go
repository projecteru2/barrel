package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const jsonObjectString = `{
	"null": null,
	"string": "string",
	"int": 42,
	"float": 2.718,
	"true": true,
	"false": false,
	"array": [ {} ],
	"object": {
		"object": {}
	} 
}`

func TestJsonObject(t *testing.T) {
	var (
		jsonString = jsonObjectString
		jsonObject Object
		jsonNode   Any
		err        error
		isObject   bool
	)
	jsonObject, err = UnmarshalObject([]byte(jsonString))
	assert.Nil(t, err, "UnmarshalObject should be success")
	assertObjectCase(t, jsonObject)

	jsonNode, err = Unmarshal([]byte(jsonString))
	assert.Nil(t, err, "Unmarshal should be success")
	jsonObject, isObject = jsonNode.ObjectValue()
	assert.True(t, isObject, "should be object")
	assertObjectCase(t, jsonObject)
}

func TestEditObject(t *testing.T) {
	object := constructObject()
	bytes, err := Marshal(object.Any())
	assert.Nil(t, err, "Marshal should cause no error")

	jsonObject, err := UnmarshalObject(bytes)
	assert.Nil(t, err, "UnmarshalObject should be success")
	assertObjectCase(t, jsonObject)
}

func constructObject() Object {
	object := NewObjectNode()
	object.Set("null", NewNullNode())
	object.Set("string", NewStringNode("string"))
	object.Set("int", NewIntNode(42))
	object.Set("float", NewFloatNode(2.718))
	object.Set("true", NewBoolNode(true))
	object.Set("false", NewBoolNode(false))
	object.Set("array", childArray().Any())
	object.Set("object", childObject().Any())
	return object
}

func childObject() Object {
	childObj := NewObjectNode()
	childObj.Set("object", NewObjectNode().Any())
	return childObj
}

func childArray() Array {
	array := NewArrayNode()
	array.Add(NewObjectNode().Any())
	return array
}

func TestJsonArray(t *testing.T) {
	var (
		jsonString = "[" + jsonObjectString + `, 
			null, 
			"string", 
			42,
			2.718,
			true,
			false,
			[{}],
			{"object": {}}
		]`
		jsonArray Array
		err       error
	)
	jsonArray, err = UnmarshalArray([]byte(jsonString))
	assert.Nil(t, err, "UnmarshalObject should be success")
	assertArrayCase(t, jsonArray)
}

func TestEditArray(t *testing.T) {
	array := NewArrayNode()
	array.Add(constructObject().Any())
	array.Add(NewNullNode())
	array.Add(NewStringNode("string"))
	array.Add(NewIntNode(42))
	array.Add(NewFloatNode(2.718))
	array.Add(NewBoolNode(true))
	array.Add(NewBoolNode(false))
	array.Add(childArray().Any())
	array.Add(childObject().Any())

	bytes, err := Marshal(array.Any())
	assert.Nil(t, err, "Marshal should cause no error")

	jsonArray, err := UnmarshalArray(bytes)
	assert.Nil(t, err, "UnmarshalArray should be success")
	assertArrayCase(t, jsonArray)
}

func assertArrayCase(t *testing.T, jsonArray Array) {
	objectNode, isObject := asserttedGetFromArray(t, jsonArray, 0).ObjectValue()
	assert.True(t, isObject, "should be object")
	assertObjectCase(t, objectNode)
	assertIsNull(t, asserttedGetFromArray(t, jsonArray, 1))
	assertIsString(t, asserttedGetFromArray(t, jsonArray, 2))
	assertIsInt(t, asserttedGetFromArray(t, jsonArray, 3))
	assertIsFloat(t, asserttedGetFromArray(t, jsonArray, 4))
	assertIsTrue(t, asserttedGetFromArray(t, jsonArray, 5))
	assertIsFalse(t, asserttedGetFromArray(t, jsonArray, 6))
	assertIsArray(t, asserttedGetFromArray(t, jsonArray, 7))
	assertIsObject(t, asserttedGetFromArray(t, jsonArray, 8))
}

func assertObjectCase(t *testing.T, object Object) {
	assertIsNull(t, asserttedGetFromObject(t, object, "null"))
	assertIsString(t, asserttedGetFromObject(t, object, "string"))
	assertIsInt(t, asserttedGetFromObject(t, object, "int"))
	assertIsFloat(t, asserttedGetFromObject(t, object, "float"))
	assertIsTrue(t, asserttedGetFromObject(t, object, "true"))
	assertIsFalse(t, asserttedGetFromObject(t, object, "false"))
	assertIsArray(t, asserttedGetFromObject(t, object, "array"))
	assertIsObject(t, asserttedGetFromObject(t, object, "object"))
}

func asserttedGetFromObject(t *testing.T, object Object, name string) Any {
	var (
		any Any
		ok  bool
	)
	any, ok = object.Get(name)
	assert.True(t, ok, `field["%s"] should be presented`, name)
	return any
}

func asserttedGetFromArray(t *testing.T, array Array, index int) Any {
	size := array.Size()
	assert.True(t, index < size, `index[%v] should be presented`, index)
	return array.Get(index)
}

func assertIsNull(t *testing.T, any Any) {
	assert.True(t, any.Null(), "node should be Null")
}

func assertIsString(t *testing.T, any Any) {
	value, ok := any.StringValue()
	assert.True(t, ok, "node should be string")
	assert.Equal(t, "string", value, "node value should be equal to %s, but is %s", "string", value)
}

func assertIsInt(t *testing.T, any Any) {
	value, ok := any.IntValue()
	assert.True(t, ok, "node should be int")
	assert.Equal(t, int64(42), value, "node value should be equal to %v, but is %v", 42, value)
}

func assertIsFloat(t *testing.T, any Any) {
	value, ok := any.FloatValue()
	assert.True(t, ok, "node should be float")
	assert.Equal(t, float64(2.718), value, "node value should be equal to %v, but is %v", 2.718, value)
}

func assertIsTrue(t *testing.T, any Any) {
	value, ok := any.BoolValue()
	assert.True(t, ok, "node should be bool")
	assert.Equal(t, true, value, "node value should be equal to %v, but is %v", true, value)
}

func assertIsFalse(t *testing.T, any Any) {
	value, ok := any.BoolValue()
	assert.True(t, ok, "node should be bool")
	assert.Equal(t, false, value, "node value should be equal to %v, but is %v", false, value)
}

func assertIsArray(t *testing.T, any Any) {
	value, ok := any.ArrayValue()
	assert.True(t, ok, "node should be array")
	assert.Equal(t, 1, value.Size(), "size of array should be equal to %v, but is %v", 1, value.Size())
}

func assertIsObject(t *testing.T, any Any) {
	value, ok := any.ObjectValue()
	assert.True(t, ok, "node should be object")
	assert.True(t, value.Has("object"), "object node should have key %s", "object")
}
