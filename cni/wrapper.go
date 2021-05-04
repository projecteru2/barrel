package cni

type Wrapper struct {
	CNIBin     string
	Env        []string
	StdinBytes []byte
}

func (w Wrapper) Run() error {
	return nil
}

func (w Wrapper) RequireFixedIP() bool {
	return false
}
