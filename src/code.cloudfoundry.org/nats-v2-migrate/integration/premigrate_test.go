package integration

import (
	"encoding/json"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/nats-v2-migrate/integration/helpers"
	"code.cloudfoundry.org/nats-v2-migrate/natsinfo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Premigrate", func() {
	var (
		cfg                 config.Config
		configFile          *os.File
		natsV1BPMConfigFile *os.File
		natsBPMConfigFile   *os.File
		premigrateBin       string
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

		cfg = config.Config{
			NATSPeers:           []string{},
			NATSMigrateServers:  []string{},
			NATSBPMv1ConfigPath: natsV1BPMConfigFile.Name(),
			NATSBPMConfigPath:   natsBPMConfigFile.Name(),
		}

		configFile, err = ioutil.TempFile("", "premigrate_config_1_server")
		Expect(err).NotTo(HaveOccurred())

		cfgJSON, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())

		_, err = configFile.Write(cfgJSON)
		Expect(err).NotTo(HaveOccurred())

		premigrateBin, err = gexec.Build("code.cloudfoundry.org/nats-v2-migrate/cmd/premigrate")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(configFile.Name())
		os.Remove(natsV1BPMConfigFile.Name())
		os.Remove(natsBPMConfigFile.Name())
	})

	Context("when there is only one nats-server", func() {
		It("keeps original bpm config in place", func() {
			premigrateCmd := exec.Command(premigrateBin, "-config-file", configFile.Name())
			sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			Eventually(sess).Should(gexec.Exit(0))

			bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
			Expect(err).NotTo(HaveOccurred())
			Expect(bpmConfigContents).To(Equal([]byte("v2-bpm-config")))
		})
	})

	Context("when there are multiple nats-servers", func() {
		var natsRunner1, natsRunner2 *helpers.NATSRunner

		BeforeEach(func() {
			natsRunner1 = helpers.NewNATSRunner(int(4225))
			natsRunner2 = helpers.NewNATSRunner(int(4226))
			cfg.NATSPeers = []string{
				natsRunner1.Addr(),
				natsRunner2.Addr(),
			}
			cfg.NATSMigrateServers = []string{
				natsRunner1.URL(),
				natsRunner2.URL(),
			}
			cfgJSON, err := json.Marshal(cfg)
			Expect(err).NotTo(HaveOccurred())

			err = os.WriteFile(configFile.Name(), cfgJSON, fs.ModePerm)
			Expect(err).NotTo(HaveOccurred())
		})

		AfterEach(func() {
			if natsRunner1 != nil {
				natsRunner1.Stop()
			}

			if natsRunner2 != nil {
				natsRunner2.Stop()
			}
		})

		Context("with all nats-servers running v2", func() {
			BeforeEach(func() {
				natsRunner1.Start()
				version, err := natsinfo.GetMajorVersion(natsRunner1.Addr())
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(2))

				natsRunner2.Start()
				version, err = natsinfo.GetMajorVersion(natsRunner2.Addr())
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(2))
			})

			It("keeps original bpm config in place", func() {
				premigrateCmd := exec.Command(premigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).WithTimeout(61 * time.Second).Should(gexec.Exit(0))

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v2-bpm-config")))
			})
		})

		Context("with one nats-server running v1", func() {
			BeforeEach(func() {
				natsRunner1.StartV1()
				version, err := natsinfo.GetMajorVersion(natsRunner1.Addr())
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(1))

				natsRunner2.Start()
				version, err = natsinfo.GetMajorVersion(natsRunner2.Addr())
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(2))

			})

			It("switches to v1 bpm config", func() {
				premigrateCmd := exec.Command(premigrateBin, "-config-file", configFile.Name())
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
				natsRunner2.Start()
				version, err := natsinfo.GetMajorVersion(natsRunner2.Addr())
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(2))
			})

			It("retries for the timeout period", func() {
				premigrateCmd := exec.Command(premigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Consistently(sess).WithTimeout(5 * time.Second).ShouldNot(gexec.Exit())

				natsRunner1.StartV1()
				version, err := natsinfo.GetMajorVersion(natsRunner1.Addr())
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(1))
				Eventually(sess).WithTimeout(61 * time.Second).Should(gexec.Exit(0))

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v1-bpm-config")))
			})
		})

		Context("when it fails to connect to one nats server and times out", func() {
			BeforeEach(func() {
				natsRunner2.Start()
				version, err := natsinfo.GetMajorVersion(natsRunner2.Addr())
				Expect(err).NotTo(HaveOccurred())
				Expect(version).To(Equal(2))
			})

			It("keeps v2 bpm config", func() {
				premigrateCmd := exec.Command(premigrateBin, "-config-file", configFile.Name())
				sess, err := gexec.Start(premigrateCmd, GinkgoWriter, GinkgoWriter)
				Expect(err).NotTo(HaveOccurred())
				Eventually(sess).WithTimeout(61 * time.Second).Should(gexec.Exit(0))

				bpmConfigContents, err := os.ReadFile(natsBPMConfigFile.Name())
				Expect(err).NotTo(HaveOccurred())
				Expect(bpmConfigContents).To(Equal([]byte("v2-bpm-config")))
			})
		})
	})
})
