package cni

import (
	"context"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"sync"
	"syscall"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
)

func (w Wrapper) runAdd(ctx context.Context) (err error) {
	netEP, err := w.GetNetEndpoint(ctx, w.GetWorkloadID(), w.GetRequestIPv4())
	if err != nil {
		return
	}
	if netEP != nil {
		return w.responseAddByEP(ctx, netEP)
	}

	return w.responseAddByCNI(ctx)
}

func (w Wrapper) responseAddByEP(ctx context.Context, netEP *NetEndpoint) (err error) {
	if err = syscall.Mount(netEP.netnsPath, w.GetNetnsPath(), "tmpfs", syscall.MS_BIND, ""); err != nil {
		return
	}

	netEP.occupiedWorkloadID = w.GetWorkloadID()
	if !netEP.ContainsWorkloadID(w.GetWorkloadID()) {
		netEP.workloadIDs = append(netEP.workloadIDs, w.GetWorkloadID())
		// TODO@zc: race condition!!!!
		if err = w.UpdateNetEndpoint(ctx, *netEP); err != nil {
			log.Errorf("failed to update net endpoint: %+v", err)
			return
		}
	}

	result, err := netEP.CNIResult()
	if err != nil {
		return
	}
	return result.Forward()
}

func (w Wrapper) responseAddByCNI(ctx context.Context) (err error) {
	cniResult, err := w.cniAdd()
	if err != nil {
		return err
	}

	if cniResult.Successful() {
		netEP := NetEndpoint{
			workloadIDs:        []string{w.GetWorkloadID()},
			occupiedWorkloadID: w.GetWorkloadID(),
			ipv4:               cniResult.IPv4(),
			netnsPath:          cniResult.MetaNetnsPath(),
			successfulStdout:   cniResult.stdout,
		}
		// TODO@zc: timeout config
		if err = w.CreateNetEndpoint(ctx, netEP); err != nil {
			// TODO@zc: how to deal with failure of recording endpoint?
			log.Errorf("failed to create barrel CNI endpoint: %+v", err)
		}
	}

	return cniResult.Forward()
}

func (w Wrapper) cniAdd() (result Result, err error) {
	var metaNetnsPath string
	cmd := exec.Command(w.bin)
	if cmd.Env, metaNetnsPath, err = w.generateCNIEnv(os.Environ()); err != nil {
		return
	}
	defer func() {
		// TODO@zc: check txn everywhere
		if err != nil {
			cmd := exec.Command("ip", "net", "d", metaNetnsPath)
			if e := cmd.Run(); e != nil {
				log.Errorf("failed to rollback and delete netns: %+v", err)
			}
		}
	}()

	cmd.Stdin = os.Stdin
	stdoutReader, err := cmd.StdoutPipe()
	if err != nil {
		return
	}
	stderrReader, err := cmd.StderrPipe()
	if err != nil {
		return
	}

	wg := sync.WaitGroup{}
	var stdout, stderr []byte

	wg.Add(1)
	go func() (e error) {
		defer wg.Done()
		if stdout, e = ioutil.ReadAll(stdoutReader); err != nil {
			// shouldn't reach here, a command line stdout doesn't have EOF???
			log.Errorf("failed to read cni stdout: %+v", e)
		}
		return
	}()

	wg.Add(1)
	go func() (e error) {
		defer wg.Done()
		if stderr, e = ioutil.ReadAll(stderrReader); err != nil {
			log.Errorf("failed to read cni stderr: %+v", e)
		}
		return
	}()

	wg.Wait()
	res, err := newResult(stdout, stderr, metaNetnsPath)
	return *res, err
}

func (w Wrapper) generateCNIEnv(environ []string) (env []string, metaNetnsPath string, err error) {
	metaNetnsPath = "/var/run/netns/_" + uuid.New().String()[:8]
	cmd := exec.Command("ip", "net", "a", metaNetnsPath)
	if err = cmd.Run(); err != nil {
		return
	}

	var netnsPath string
	env = append(env, "CNI_NETNS="+metaNetnsPath)
	for _, e := range environ {
		if strings.HasPrefix(e, "CNI_NETNS=") {
			netnsPath = strings.TrimPrefix(e, "CNI_NETNS=")
			continue
		}
		env = append(env, e)
	}

	err = syscall.Mount(metaNetnsPath, netnsPath, "tmpfs", syscall.MS_BIND, "")
	return
}
