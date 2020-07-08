module github.com/projecteru2/barrel

go 1.14

require (
	github.com/coreos/etcd v3.3.8+incompatible
	github.com/docker/go-connections v0.3.0
	github.com/pkg/errors v0.8.1
	github.com/projectcalico/libcalico-go v2.0.0-alpha1.0.20180615230155-efdf8fede805+incompatible
	github.com/projecteru2/minions v0.0.0-20200707104254-075470f5f35c
	github.com/sirupsen/logrus v1.4.2
	github.com/urfave/cli/v2 v2.2.0
)

replace github.com/projecteru2/minions => github.com/nyanpassu/minions v0.0.0-20200707104254-075470f5f35c
