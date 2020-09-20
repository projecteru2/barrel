package docker

// func newMockHandler() ContainerCreateHandler {
// 	return ContainerCreateHandler{}
// }

// func TestCasesWillPassingCheckFixedIP(t *testing.T) {
// 	var fixedIPEnables = []string{
// 		// cases should enable fixed ip
// 		`{
// 			"Labels": { "fixed-ip": true },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": 1 },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": -1 },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": null },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": null },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": "true" },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": "1" },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		// it's a meaning less value, but we recognize it as true, avoiding return error
// 		`{
// 			"Labels": { "fixed-ip": "adbldfe" },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 	}
// 	hander := newMockHandler()
// 	for _, json := range fixedIPEnables {
// 		if obj, err := utils.UnmarshalObject([]byte(json)); err != nil {
// 			t.Errorf("[TestCasesWillPassingCheckFixedIP] UnmarshalObject error, %v", err)
// 		} else if passed, err := hander.checkFixedIPLabel(obj); err != nil {
// 			t.Errorf("[TestCasesWillPassingCheckFixedIP] checkFixedIPLabel error, %v", err)
// 		} else if !passed {
// 			t.Errorf("[TestCasesWillPassingCheckFixedIP] the given case should passing checkFixedIPLabel: %s", json)
// 		}
// 	}
// }

// func TestCasesWillNotPassingCheckFixedIP(t *testing.T) {
// 	var fixedIPDisabledCases = []string{
// 		// cases below is in the network but doesn't request fixed ip
// 		`{
// 			"Labels": { "fixed-ip": false },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": "false" },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": 0 },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": "0" },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		`{
// 			"Labels": { },
// 			"HostConfig": { "NetworkMode": "clouddev" }
// 		}`,
// 		// case below request fixed ip but doesn't has a valid network mode
// 		`{ "Labels": { "fixed-ip": true } }`,
// 		`{ "Labels": { "fixed-ip": 1 } }`,
// 		`{ "Labels": { "fixed-ip": -1 } }`,
// 		`{ "Labels": { "fixed-ip": null } }`,
// 		`{ "Labels": { "fixed-ip": null } }`,
// 		`{ "Labels": { "fixed-ip": "true" } }`,
// 		`{ "Labels": { "fixed-ip": "1" } }`,
// 		`{ "Labels": { "fixed-ip": "adbldfe" } }`,
// 		// case below is not in out network
// 		`{
// 			"Labels": { "fixed-ip": true },
// 			"HostConfig": { "NetworkMode": "cloud" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": 1 },
// 			"HostConfig": { "NetworkMode": "cloud" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": -1 },
// 			"HostConfig": { "NetworkMode": "cloud" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": null },
// 			"HostConfig": { "NetworkMode": "cloud" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": null },
// 			"HostConfig": { "NetworkMode": "cloud" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": "true" },
// 			"HostConfig": { "NetworkMode": "cloud" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": "1" },
// 			"HostConfig": { "NetworkMode": "cloud" }
// 		}`,
// 		`{
// 			"Labels": { "fixed-ip": "adbldfe" },
// 			"HostConfig": { "NetworkMode": "cloud" }
// 		}`,
// 	}

// 	hander := newMockHandler()
// 	for _, json := range fixedIPDisabledCases {
// 		if obj, err := utils.UnmarshalObject([]byte(json)); err != nil {
// 			t.Errorf("[TestCasesWillNotPassingCheckFixedIP] UnmarshalObject error, %v", err)
// 		} else if passed, err := hander.checkFixedIPLabel(obj); err != nil {
// 			t.Error(err)
// 		} else if passed {
// 			t.Errorf("the given case should not passing checkFixedIPLabel: %s", json)
// 		}
// 	}
// }
