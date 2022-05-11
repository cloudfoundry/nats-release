package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"

	. "github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = FDescribe("Migrate", func() {
	type MigrateConfig struct {
		NATSMigratePort int `json:"nats_migrate_port"`
	}

	var (
		cfg              MigrateConfig
		configFile       *os.File
		migrateBin       string
		migrateServerURL string
	)

	BeforeEach(func() {
		cfg = MigrateConfig{
			NATSMigratePort: 10000 + config.GinkgoConfig.ParallelNode,
		}

		var err error
		configFile, err = ioutil.TempFile("", "migrate_config")
		Expect(err).NotTo(HaveOccurred())

		cfgJSON, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())

		_, err = configFile.Write(cfgJSON)
		Expect(err).NotTo(HaveOccurred())

		migrateBin, err = gexec.Build("code.cloudfoundry.org/nats-v2-migrate/migrate")
		Expect(err).NotTo(HaveOccurred())

		migrateServerURL = fmt.Sprintf("http://127.0.0.1:%d", cfg.NATSMigratePort)
	})

	AfterEach(func() {
		os.Remove(configFile.Name())
	})

	Context("when there are no other nats machines", func() {
		var migrateSess *gexec.Session

		BeforeEach(func() {
			migrateCmd := exec.Command(migrateBin, "-config-file", configFile.Name())
			var err error
			migrateSess, err = gexec.Start(migrateCmd, GinkgoWriter, GinkgoWriter)
			Expect(err).NotTo(HaveOccurred())
			migrateServerAddress := fmt.Sprintf("127.0.0.1:%d", cfg.NATSMigratePort)
			Eventually(func() error {
				_, err := net.Dial("tcp", migrateServerAddress)
				return err
			}).Should(Succeed())
		})

		AfterEach(func() {
			migrateSess.Kill()
		})

		It("starts an API server", func() {
			resp, err := http.Get(migrateServerURL)
			Expect(err).NotTo(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
		})
	})
})
