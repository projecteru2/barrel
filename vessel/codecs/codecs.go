package codecs

import (
	"encoding/json"
	"fmt"

	"github.com/projecteru2/barrel/types"
)

// IPInfoCodec .
type IPInfoCodec struct {
	IPInfo  *types.IPInfo
	version int64
}

// Key .
func (codec *IPInfoCodec) Key() string {
	if codec.IPInfo.Address == "" {
		return ""
	}
	if codec.IPInfo.PoolID == "" {
		return fmt.Sprintf("/barrel/addresses/%s", codec.IPInfo.Address)
	}
	return fmt.Sprintf("/barrel/pools/%s/addresses/%s", codec.IPInfo.PoolID, codec.IPInfo.Address)
}

// Encode .
func (codec *IPInfoCodec) Encode() (string, error) {
	return marshal(codec.IPInfo)
}

// SetVersion .
func (codec *IPInfoCodec) SetVersion(version int64) {
	codec.version = version
}

// Version .
func (codec *IPInfoCodec) Version() int64 {
	return codec.version
}

// Decode .
func (codec *IPInfoCodec) Decode(input string) error {
	return json.Unmarshal([]byte(input), codec.IPInfo)
}

// ContainerInfoCodec .
type ContainerInfoCodec struct {
	Info    *types.ContainerInfo
	version int64
}

// Key .
func (codec ContainerInfoCodec) Key() string {
	if codec.Info.ID == "" || codec.Info.HostName == "" {
		return ""
	}
	return fmt.Sprintf("/barrel/hosts/%s/containers/%s", codec.Info.HostName, codec.Info.ID)
}

// Encode .
func (codec ContainerInfoCodec) Encode() (string, error) {
	return marshal(codec.Info)
}

// SetVersion .
func (codec *ContainerInfoCodec) SetVersion(version int64) {
	codec.version = version
}

// Version .
func (codec *ContainerInfoCodec) Version() int64 {
	return codec.version
}

// Decode .
func (codec ContainerInfoCodec) Decode(input string) error {
	return json.Unmarshal([]byte(input), codec.Info)
}

func marshal(src interface{}) (string, error) {
	bytes, err := json.Marshal(src)
	return string(bytes), err
}

// IPInfoMultiGetCodec .
type IPInfoMultiGetCodec struct {
	PrefixKey string
	Codecs    []*IPInfoCodec
	Errors    []error
}

// Prefix .
func (codec *IPInfoMultiGetCodec) Prefix() string {
	return codec.PrefixKey
}

// Decode .
func (codec *IPInfoMultiGetCodec) Decode(val string, ver int64) {
	c := &IPInfoCodec{}
	if err := c.Decode(val); err != nil {
		codec.Errors = append(codec.Errors, err)
		return
	}
	c.SetVersion(ver)
	codec.Codecs = append(codec.Codecs, c)
}
