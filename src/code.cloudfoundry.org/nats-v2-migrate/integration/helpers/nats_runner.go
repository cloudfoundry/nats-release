package helpers

import (
	"fmt"
	"os/exec"
	"strconv"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/onsi/gomega/gexec"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

type NATSRunner struct {
	port        int
	natsSession *gexec.Session
}

func NewNATSRunner(port int) *NATSRunner {
	return &NATSRunner{
		port: port,
	}
}

func (runner *NATSRunner) Stop() {
	runner.KillWithFire()
}

func (runner *NATSRunner) URL() string {
	return fmt.Sprintf("nats://127.0.0.1:%d", runner.port)
}

func (runner *NATSRunner) Addr() string {
	return fmt.Sprintf("127.0.0.1:%d", runner.port)
}

func (runner *NATSRunner) KillWithFire() {
	if runner.natsSession != nil {
		runner.natsSession.Kill().Wait(5 * time.Second)
		runner.natsSession = nil
	}
}

func (runner *NATSRunner) StartV1() {
	runner.Start("v1")
}

func (runner *NATSRunner) Start(version ...string) {
	if runner.natsSession != nil {
		panic("starting an already started NATS runner!!!")
	}

	var cmd *exec.Cmd

	if version != nil && version[0] == "v1" {
		gnatsdBin, err := gexec.Build("github.com/nats-io/gnatsd", "-buildvcs=false")
		Expect(err).NotTo(HaveOccurred())
		cmd = exec.Command(gnatsdBin, "-p", strconv.Itoa(runner.port))
	} else {
		natsServerBin, err := gexec.Build("github.com/nats-io/nats-server/v2", "-buildvcs=false")
		Expect(err).NotTo(HaveOccurred())
		cmd = exec.Command(natsServerBin, "-p", strconv.Itoa(runner.port))
	}

	sess, err := gexec.Start(cmd,
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[34m[nats-server]\x1b[0m ", GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[34m[nats-server]\x1b[0m ", GinkgoWriter))
	Expect(err).NotTo(HaveOccurred())

	runner.natsSession = sess

	Eventually(func() error {
		_, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", runner.port))
		return err
	}, 5, 0.1).ShouldNot(HaveOccurred())
}
