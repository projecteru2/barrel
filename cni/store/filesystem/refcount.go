package filesystem

import (
	"os"

	"github.com/projecteru2/barrel/cni"
)

func (s FSStore) increaseRefcount(nep *cni.NetEndpoint) (err error) {
	file, err := os.OpenFile(s.RefcountPath(nep.IPv4), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 644)
	if err != nil {
		return
	}
	_, err = file.Write([]byte{'+'})
	return
}

func (s FSStore) decreaseRefcount(nep *cni.NetEndpoint) (err error) {
	file, err := os.OpenFile(s.RefcountPath(nep.IPv4), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 644)
	if err != nil {
		return
	}
	_, err = file.Write([]byte{'-'})
	return
}

func (s FSStore) GetNetEndpointRefcount(nep *cni.NetEndpoint) (rc int, err error) {
	file, err := os.OpenFile(s.RefcountPath(nep.IPv4), os.O_WRONLY|os.O_CREATE|os.O_APPEND, 644)
	if err != nil {
		return
	}
	// 我不信你能 borrow 1024 次
	// 反复 replace?
	bs := make([]byte, 1024)
	if _, err = file.Read(bs); err != nil {
		return
	}
	for _, b := range bs {
		switch b {
		case '+':
			rc++
		case '-':
			rc--
		default:
			break
		}
	}
	return
}
