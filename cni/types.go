package cni

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Result struct {
	stdout    []byte
	stderr    []byte
	netnsPath string

	parsed *ParsedResult
}

type ParsedResult struct {
	IPs []struct {
		Address string `json:"address"`
	} `json:"ips"`
	Code int `json:"code"`
}

func newResult(stdout, stderr []byte, netnsPath string) (*Result, error) {
	result := &Result{
		stdout:    stdout,
		stderr:    stderr,
		netnsPath: netnsPath,
	}
	return result, json.Unmarshal(result.stdout, result.parsed)
}

func (r Result) Successful() bool {
	return r.parsed.Code == 0 && len(r.parsed.IPs) > 0
}

func (r Result) IPv4() string {
	if len(r.parsed.IPs) > 0 {
		return r.parsed.IPs[0].Address
	}
	return ""
}

func (r Result) MetaNetnsPath() string {
	return r.netnsPath
}

func (r Result) Forward() (err error) {
	wg := sync.WaitGroup{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err = io.Copy(os.Stdout, bytes.NewReader(r.stdout)); err != nil {
			log.Errorf("failed to forward stdout: %+v", err)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err = io.Copy(os.Stderr, bytes.NewReader(r.stderr)); err != nil {
			log.Errorf("failed to forward stderr: %+v", err)
		}
	}()

	wg.Done()
	return
}

type NetEndpoint struct {
	workloadIDs        []string `json:"workload_ids"`
	occupiedWorkloadID string   `json:"occupied_workload_id"`
	ipv4               string   `json:"ipv4"`
	netnsPath          string   `json:"netns_path"`
	successfulStdout   []byte   `json:"successful_result"`
}

func (e NetEndpoint) ContainsWorkloadID(workloadID string) bool {
	for _, id := range e.workloadIDs {
		if id == workloadID {
			return true
		}
	}
	return false
}

func (e NetEndpoint) CNIResult() (Result, error) {
	result, err := newResult(e.successfulStdout, nil, e.netnsPath)
	return *result, err
}
