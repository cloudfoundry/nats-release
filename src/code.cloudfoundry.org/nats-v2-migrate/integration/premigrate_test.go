package integration

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/nats-v2-migrate/integration/helpers"
	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

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
		natsRunner          *helpers.NATSRunner
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
		natsRunner = helpers.NewNATSRunner(int(natsPort))
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
				conn, err := nats.Connect(natsRunner.URL())
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
		var natsRunner2, natsRunner3 *helpers.NATSRunner

		BeforeEach(func() {
			natsRunner2 = helpers.NewNATSRunner(int(4225))
			natsRunner3 = helpers.NewNATSRunner(int(4226))
			cfg.NATSMachines = []string{
				natsRunner2.Addr(),
				natsRunner3.Addr(),
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
				conn, err := nats.Connect(natsRunner.URL())
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner2.Start()
				conn, err = nats.Connect(natsRunner2.URL())
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner3.Start()
				conn, err = nats.Connect(natsRunner3.URL())
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
				conn, err := nats.Connect(natsRunner.URL())
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner2.StartV1()
				conn, err = nats.Connect(natsRunner2.URL())
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("1.4.1"))

				natsRunner3.Start()
				conn, err = nats.Connect(natsRunner3.URL())
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
				conn, err := nats.Connect(natsRunner.URL())
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner3.Start()
				conn, err = nats.Connect(natsRunner3.URL())
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
			})

			It("retries for the timeout period", func() {
				premigrateCmd := exec.Command(premmigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Consistently(sess).WithTimeout(5 * time.Second).ShouldNot(gexec.Exit())

				natsRunner2.StartV1()
				conn, err := nats.Connect(natsRunner2.URL())
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("1.4.1"))
				Eventually(sess).WithTimeout(10 * time.Second).Should(gexec.Exit(0))

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v1-bpm-config")))
			})
		})

		Context("when it fails to connect to one nats server and times out", func() {
			BeforeEach(func() {
				natsRunner.Start()
				conn, err := nats.Connect(natsRunner.URL())
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))

				natsRunner3.Start()
				conn, err = nats.Connect(natsRunner3.URL())
				Expect(err).ToNot(HaveOccurred())
				version = conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
			})

			It("keeps v2 bpm config", func() {
				premigrateCmd := exec.Command(premmigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).WithTimeout(12 * time.Second).Should(gexec.Exit(0))

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v2-bpm-config")))
			})
		})
	})
})
