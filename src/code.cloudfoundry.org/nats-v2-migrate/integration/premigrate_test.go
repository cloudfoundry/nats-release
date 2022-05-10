package integration

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
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
		gnatsdBin, err := gexec.Build("github.com/nats-io/gnatsd")
		Expect(err).NotTo(HaveOccurred())
		cmd = exec.Command(gnatsdBin, "-p", strconv.Itoa(runner.port))
	} else {
		natsServerBin, err := gexec.Build("github.com/nats-io/nats-server/v2")
		Expect(err).NotTo(HaveOccurred())
		cmd = exec.Command(natsServerBin, "-p", strconv.Itoa(runner.port))
	}

	sess, err := gexec.Start(cmd,
		gexec.NewPrefixedWriter("\x1b[32m[o]\x1b[34m[nats-server]\x1b[0m ", ginkgo.GinkgoWriter),
		gexec.NewPrefixedWriter("\x1b[91m[e]\x1b[34m[nats-server]\x1b[0m ", ginkgo.GinkgoWriter))
	Expect(err).NotTo(HaveOccurred())

	runner.natsSession = sess

	Eventually(func() error {
		_, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", runner.port))
		return err
	}, 5, 0.1).ShouldNot(HaveOccurred())
}

var _ = Describe("Premigrate", func() {
	type PremigrateConfig struct {
		NATSMachines        []string `json:"nats_machines"`
		NATSV1BPMConfigPath string   `json:"nats_v1_bpm_config_path"`
		NATSBPMConfigPath   string   `json:"nats_bpm_config_path"`
	}

	var (
		cfg                 PremigrateConfig
		configFile          *os.File
		natsV1BPMConfigFile *os.File
		natsBPMConfigFile   *os.File
		premmigrateBin      string
		natsPort            uint16
		natsRunner          *NATSRunner
	)

	BeforeEach(func() {
		var err error
		natsV1BPMConfigFile, err = ioutil.TempFile("", "nats_v1_bpm_config_path")
		Expect(err).NotTo(HaveOccurred())
		_, err = natsV1BPMConfigFile.Write([]byte("v1-bpm-config"))
		Expect(err).NotTo(HaveOccurred())

		natsBPMConfigFile, err = ioutil.TempFile("", "nats_bpm_config_path")
		Expect(err).NotTo(HaveOccurred())
		_, err = natsBPMConfigFile.Write([]byte("v2-bpm-config"))
		Expect(err).NotTo(HaveOccurred())

		cfg = PremigrateConfig{
			NATSMachines:        []string{},
			NATSV1BPMConfigPath: natsV1BPMConfigFile.Name(),
			NATSBPMConfigPath:   natsBPMConfigFile.Name(),
		}

		configFile, err = ioutil.TempFile("", "premigrate_config_1_server")
		Expect(err).NotTo(HaveOccurred())

		cfgJSON, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())

		_, err = configFile.Write(cfgJSON)
		Expect(err).NotTo(HaveOccurred())

		premmigrateBin, err = gexec.Build("code.cloudfoundry.org/nats-v2-migrate/premigrate")
		Expect(err).NotTo(HaveOccurred())

		natsPort = 4224
		natsRunner = NewNATSRunner(int(natsPort))
	})

	AfterEach(func() {
		os.Remove(configFile.Name())
		os.Remove(natsV1BPMConfigFile.Name())
		os.Remove(natsBPMConfigFile.Name())
	})

	Context("when there is only one nats-server", func() {
		Context("with nats-server running v2", func() {
			BeforeEach(func() {
				natsRunner.Start()
				conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner.port))
				Expect(err).ToNot(HaveOccurred())

				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
			})

			AfterEach(func() {
				if natsRunner != nil {
					natsRunner.Stop()
				}
			})

			It("keeps original bpm config in place", func() {
				premigrateCmd := exec.Command(premmigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(0))

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v2-bpm-config")))
			})
		})
	})

	Context("when there are multiple nats-servers", func() {
		var natsRunner2, natsRunner3 *NATSRunner

		BeforeEach(func() {
			natsRunner2 = NewNATSRunner(int(4225))
			natsRunner3 = NewNATSRunner(int(4226))
			cfg.NATSMachines = []string{
				fmt.Sprintf("127.0.0.1:%d", natsRunner2.port),
				fmt.Sprintf("127.0.0.1:%d", natsRunner3.port),
			}
			cfgJSON, err := json.Marshal(cfg)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(configFile.Name(), cfgJSON, fs.ModePerm)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if natsRunner != nil {
				natsRunner.Stop()
			}

			if natsRunner2 != nil {
				natsRunner2.Stop()
			}

			if natsRunner3 != nil {
				natsRunner3.Stop()
			}
		})

		Context("with all nats-servers running v2", func() {
			BeforeEach(func() {
				natsRunner.Start()
				conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner.port))
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner2.Start()
				conn, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner2.port))
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner3.Start()
				conn, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner3.port))
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
			})

			It("keeps original bpm config in place", func() {
				premigrateCmd := exec.Command(premmigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit(0))

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v2-bpm-config")))
			})
		})

		Context("with one nats-server running v1", func() {
			BeforeEach(func() {
				natsRunner.Start()
				conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner.port))
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner2.StartV1()
				conn, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner2.port))
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("1.4.1"))

				natsRunner3.Start()
				conn, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner3.port))
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
			})

			It("switches to v1 bpm config", func() {
				premigrateCmd := exec.Command(premmigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).Should(gexec.Exit())

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v1-bpm-config")))
			})
		})

		Context("when it fails to connect to one nats server within the timeout", func() {
			BeforeEach(func() {
				natsRunner.Start()
				conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner.port))
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner3.Start()
				conn, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner3.port))
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
			})

			It("retries", func() {
				premigrateCmd := exec.Command(premmigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Consistently(sess).WithTimeout(5 * time.Second).ShouldNot(gexec.Exit())

				natsRunner2.Start()
				conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner2.port))
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
				Eventually(sess).WithTimeout(10 * time.Second).Should(gexec.Exit(0))

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v2-bpm-config")))
			})
		})

		Context("when it fails to connect to one nats server and times out", func() {
			BeforeEach(func() {
				natsRunner.Start()
				conn, err := nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner.port))
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner3.Start()
				conn, err = nats.Connect(fmt.Sprintf("nats://127.0.0.1:%d", natsRunner3.port))
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
			})

			It("exits with failure", func() {
				premigrateCmd := exec.Command(premmigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).WithTimeout(12 * time.Second).Should(gexec.Exit(1))
			})
		})
	})
})
