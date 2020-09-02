package utils

import "strconv"

//
func asText(any Any) string {
	var (
		bytes []byte
		err   error
	)
	if bytes, err = Marshal(any); err != nil {
		return ""
	}
	return string(bytes)
}

// ********
// arrayNode
type arrayNode struct {
	array []Any
}

func createArrayNode() Array {
	return &arrayNode{array: make([]Any, 0)}
}

func newArrayNode(src []interface{}) (Array, error) {
	size := len(src)
	array := make([]Any, size)
	for i := 0; i < size; i++ {
		var err error
		if array[i], err = newNode(src[i]); err != nil {
			return nil, err
		}
	}
	return &arrayNode{array}, nil
}

func (node *arrayNode) Null() bool {
	return false
}

func (node *arrayNode) StringValue() (string, bool) {
	return "", false
}

func (node *arrayNode) String() string {
	return asText(node)
}

func (node *arrayNode) IntValue() (int64, bool) {
	return 0, false
}

func (node *arrayNode) FloatValue() (float64, bool) {
	return 0, false
}

func (node *arrayNode) BoolValue() (bool, bool) {
	return false, false
}

func (node *arrayNode) ArrayValue() (Array, bool) {
	return node, true
}

func (node *arrayNode) ObjectValue() (Object, bool) {
	return nil, false
}

func (node *arrayNode) RawValue() interface{} {
	size := len(node.array)
	array := make([]interface{}, size)
	for i := 0; i < size; i++ {
		array[i] = node.array[i].RawValue()
	}
	return array
}

func (node *arrayNode) Any() Any {
	return node
}

func (node *arrayNode) Get(index int) Any {
	return node.array[index]
}

func (node *arrayNode) Add(any ...Any) {
	node.array = append(node.array, any...)
}

func (node *arrayNode) Set(index int, value Any) {
	node.ensureSize(index)
	node.array[index] = value
}

func (node *arrayNode) ensureSize(index int) {
	if index >= len(node.array) {
		capacity := cap(node.array)
		newLength := index + 1
		if index < capacity {
			node.array = node.array[:newLength]
		} else {
			arr := node.array
			node.array = make([]Any, newLength, computeNewCapacity(capacity, newLength))
			copy(node.array, arr)
		}
	}
}

func computeNewCapacity(old int, required int) int {
	if old >= 1024 {
		newCap := int(float64(old) * 1.5)
		if newCap >= required {
			return newCap
		}
		return old/2 + required
	}
	if old*2 >= required {
		return old * 2
	}
	return old + required
}

func (node *arrayNode) Size() int {
	return len(node.array)
}

// ********
// boolNode
type boolNode struct {
	value bool
}

func newBoolNode(value bool) Any {
	return &boolNode{value}
}

// always return false
func (node *boolNode) Null() bool {
	return false
}

func (node *boolNode) StringValue() (string, bool) {
	return "", false
}

func (node *boolNode) String() string {
	return strconv.FormatBool(node.value)
}

func (node *boolNode) IntValue() (int64, bool) {
	return 0, false
}

func (node *boolNode) FloatValue() (float64, bool) {
	return 0, false
}

func (node *boolNode) BoolValue() (bool, bool) {
	return node.value, true
}

func (node *boolNode) ArrayValue() (Array, bool) {
	return nil, false
}

func (node *boolNode) ObjectValue() (Object, bool) {
	return nil, false
}

func (node *boolNode) RawValue() interface{} {
	return node.value
}

// ********
// Null .
type nullNode struct{}

var null = nullNode{}

func newNullNode() Any {
	return &null
}

// always return false
func (node nullNode) Null() bool {
	return true
}

func (node nullNode) StringValue() (string, bool) {
	return "", false
}

func (node nullNode) String() string {
	return "null"
}

func (node nullNode) IntValue() (int64, bool) {
	return 0, false
}

func (node nullNode) FloatValue() (float64, bool) {
	return 0, false
}

func (node nullNode) BoolValue() (bool, bool) {
	return false, false
}

func (node nullNode) ArrayValue() (Array, bool) {
	return nil, false
}

func (node nullNode) ObjectValue() (Object, bool) {
	return nil, false
}

func (node nullNode) RawValue() interface{} {
	return nil
}

// ********
// numberNode
type numberNode struct {
	floatValue float64
	intValue   int64
	isInt      bool
}

func newNumberNode(value float64) Any {
	intValue := int64(value)
	if value == float64(intValue) {
		return newIntNode(intValue)
	}
	return newFloatNode(value)
}

func newFloatNode(value float64) Any {
	return numberNode{
		floatValue: value,
		isInt:      false,
	}
}

func newIntNode(value int64) Any {
	return numberNode{
		intValue: value,
		isInt:    true,
	}
}

// always return false
func (node numberNode) Null() bool {
	return false
}

func (node numberNode) StringValue() (string, bool) {
	return "", false
}

func (node numberNode) String() string {
	if node.isInt {
		return strconv.FormatInt(node.intValue, 64)
	}
	return strconv.FormatFloat(node.floatValue, 'b', -1, 64)
}

func (node numberNode) IntValue() (int64, bool) {
	if node.isInt {
		return node.intValue, true
	}
	return 0, false
}

func (node numberNode) FloatValue() (float64, bool) {
	return node.floatValue, true
}

func (node numberNode) BoolValue() (bool, bool) {
	return false, false
}

func (node numberNode) ArrayValue() (Array, bool) {
	return nil, false
}

func (node numberNode) ObjectValue() (Object, bool) {
	return nil, false
}

func (node numberNode) RawValue() interface{} {
	if node.isInt {
		return node.intValue
	}
	return node.floatValue
}

// ********
// objectNode
type objectNode struct {
	objectMapping map[string]Any
}

func createObjectNode() Object {
	mapping := make(map[string]Any)
	return &objectNode{mapping}
}

func newObjectNode(src map[string]interface{}) (Object, error) {
	mapping := make(map[string]Any)
	for key, value := range src {
		var err error
		if mapping[key], err = newNode(value); err != nil {
			return nil, err
		}
	}
	return &objectNode{mapping}, nil
}

// always return false
func (node *objectNode) Null() bool {
	return false
}

func (node *objectNode) StringValue() (string, bool) {
	return "", false
}

func (node *objectNode) String() string {
	return asText(node)
}

func (node *objectNode) IntValue() (int64, bool) {
	return 0, false
}

func (node *objectNode) FloatValue() (float64, bool) {
	return 0, false
}

func (node *objectNode) BoolValue() (bool, bool) {
	return false, false
}

func (node *objectNode) ArrayValue() (Array, bool) {
	return nil, false
}

func (node *objectNode) ObjectValue() (Object, bool) {
	return node, true
}

func (node *objectNode) Any() Any {
	return node
}

func (node *objectNode) Keys() []string {
	var keys []string
	for key := range node.objectMapping {
		keys = append(keys, key)
	}
	return keys
}

func (node *objectNode) Has(key string) (has bool) {
	_, has = node.objectMapping[key]
	return
}

func (node *objectNode) Get(key string) (value Any, ok bool) {
	value, ok = node.objectMapping[key]
	return
}

func (node *objectNode) Set(key string, value Any) {
	node.objectMapping[key] = value
}

func (node *objectNode) Del(key string) {
	delete(node.objectMapping, key)
}

func (node *objectNode) RawValue() interface{} {
	mapping := make(map[string]interface{})
	for key, value := range node.objectMapping {
		mapping[key] = value.RawValue()
	}
	return mapping
}

// ********
// stringNode
type stringNode struct {
	value string
}

func newStringNode(value string) Any {
	return stringNode{value}
}

// always return false
func (node stringNode) Null() bool {
	return false
}

func (node stringNode) StringValue() (string, bool) {
	return node.value, true
}

func (node stringNode) String() string {
	return node.value
}

func (node stringNode) IntValue() (int64, bool) {
	return 0, false
}

func (node stringNode) FloatValue() (float64, bool) {
	return 0, false
}

func (node stringNode) BoolValue() (bool, bool) {
	return false, false
}

func (node stringNode) ArrayValue() (Array, bool) {
	return nil, false
}

func (node stringNode) ObjectValue() (Object, bool) {
	return nil, false
}

func (node stringNode) RawValue() interface{} {
	return node.value
}
