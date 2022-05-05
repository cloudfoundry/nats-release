package integration

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/nats-io/go-nats"
	"github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	. "github.com/onsi/gomega/gexec"
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

func (runner *NATSRunner) Start() {
	if runner.natsSession != nil {
		panic("starting an already started NATS runner!!!")
	}

	gexec.Build("github.com/nats-io/nats-server/v2")
	cmd := exec.Command("nats-server", "-p", strconv.Itoa(runner.port))

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
		cfgFile    string
		tmpdir     string
		natsPort   uint16
		natsRunner *NATSRunner
	)

	BeforeEach(func() {

		// cfgFile = filepath.Join(tmpdir, "config.yml")

		natsPort = 4224

		natsRunner = NewNATSRunner(int(natsPort))
		natsRunner.Start()
	})

	AfterEach(func() {
		if natsRunner != nil {
			natsRunner.Stop()
		}

		os.RemoveAll(tmpdir)
	})

	Context("With nats-server running v2", func() {

		It("Keeps v2 bpm config in place", func() {
			premigrateCmd := exec.Command("premigrate", "-c", cfgFile)
			Start(premigrateCmd, GinkgoWriter, GinkgoWriter)

			Expect(1).To(Equal(2))
		})
	})
})
