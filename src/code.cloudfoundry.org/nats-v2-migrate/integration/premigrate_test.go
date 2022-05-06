package integration

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
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
		gexec.Build("github.com/nats-io/gnatsd")
		cmd = exec.Command("gnatsd", "-p", strconv.Itoa(runner.port))
	} else {
		gexec.Build("github.com/nats-io/nats-server")
		cmd = exec.Command("nats-server", "-p", strconv.Itoa(runner.port))
	}

	sess, err := gexec.Start(cmd,
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[34m[nats-server]\x1b[0m ", ginkgo.GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[34m[nats-server]\x1b[0m ", ginkgo.GinkgoWriter))
	Expect(err).NotTo(HaveOccurred())

	runner.natsSession = sess

	Expect(err).NotTo(HaveOccurred())

	Eventually(func() error {
		_, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", runner.port))
		return err
	}, 5, 0.1).ShouldNot(HaveOccurred())

}

var _ = Describe("Premigrate", func() {

	var (
		// cfg                             *Config
		tmpdir     string
		natsPort   uint16
		natsRunner *NATSRunner
	)

	BeforeEach(func() {

		// cfgFile = filepath.Join(tmpdir, "config.yml")

		natsPort = 4224
		natsRunner = NewNATSRunner(int(natsPort))
	})

	AfterEach(func() {
		if natsRunner != nil {
			natsRunner.Stop()
		}

		os.RemoveAll(tmpdir)
	})

	FContext("With nats-server running v2", func() {

		BeforeEach(func() {

			natsRunner.Start()
		})

		It("confirms a nats server is running", func() {
			conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner.port))
			Expect(err).ToNot(HaveOccurred())

			version := conn.ConnectedServerVersion()
			Expect(version).To(Equal("2.8.2"))

		})
		// It("Keeps v2 bpm config in place", func() {
		// 	premigrateCmd := exec.Command("premigrate", "-c", cfgFile)
		// 	Start(premigrateCmd, GinkgoWriter, GinkgoWriter)

	})
	FContext("With nats-server running v1", func() {

		BeforeEach(func() {
			natsRunner.StartV1()
		})

		It("confirms a nats server is running", func() {
			conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner.port))
			Expect(err).ToNot(HaveOccurred())

			version := conn.ConnectedServerVersion()
			Expect(version).To(Equal("1.4.1"))

		})
		// It("Keeps v2 bpm config in place", func() {
		// 	premigrateCmd := exec.Command("premigrate", "-c", cfgFile)
		// 	Start(premigrateCmd, GinkgoWriter, GinkgoWriter)

	})
	Context("Running the premigrate script", func() {

		BeforeEach(func() {
			natsRunner.StartV1()

			gexec.Build("/home/pivotal/workspace/nats-release/src/code.cloudfoundry.org/nats-v2-migrate/premigrate")
			cmd = exec.Cmd("premigrate", "-c", "/fix/path")
			sess, err := gexec.Start(cmd,
				gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[34m[nats-server]\x1b[0m ", ginkgo.GinkgoWriter),
				gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[34m[nats-server]\x1b[0m ", ginkgo.GinkgoWriter))
			Expect(err).NotTo(HaveOccurred())
		})

		It("is able to talk to the nats server", func() {
			conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner.port))
			Expect(err).ToNot(HaveOccurred())

			version := conn.ConnectedServerVersion()
			Expect(version).To(Equal("3.0"))

		})
		// It("Keeps v2 bpm config in place", func() {
		// 	premigrateCmd := exec.Command("premigrate", "-c", cfgFile)
		// 	Start(premigrateCmd, GinkgoWriter, GinkgoWriter)

	})
})
