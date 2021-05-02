package container

type Container struct {
	ID     string `json:"Id"`
	Config struct {
		Image  string
		Labels map[string]string
	}
	Mounts []struct {
		Name        string
		Source      string
		Destination string
	}
}
