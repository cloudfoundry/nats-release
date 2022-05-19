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
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"code.cloudfoundry.org/nats-v2-migrate/config"
)

var (
	err       error
	cfgFile   *os.File
	address   string
	session   *gexec.Session
	bpmFile   *os.File
	bpmv2File *os.File
)

func StartServer(cfg config.Config) {
	cfgFile, err = ioutil.TempFile("", "migrate-config.json")
	Expect(err).NotTo(HaveOccurred())

	cfgJSON, err := json.Marshal(cfg)

	// Delete me
	// Expect(fmt.Sprintf(string(cfgJSON))).To(Equal("foo")) // does it record boostrap bool the right way
	// Expect(err).NotTo(HaveOccurred())

	_, err = cfgFile.Write(cfgJSON)
	Expect(err).NotTo(HaveOccurred())

	serverBin, err := gexec.Build("code.cloudfoundry.org/nats-v2-migrate/server")
	Expect(err).NotTo(HaveOccurred())

	startCmd := exec.Command(serverBin, "-config-file", cfgFile.Name())
	session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	address = "127.0.0.1:4242"
	serverIsAvailable := func() error {
		return VerifyTCPConnection(address)
	}
	Eventually(serverIsAvailable).Should(Succeed())
}

var _ = Describe("MigrationServer", func() {
	AfterEach(func() {
		session.Kill()
		os.Remove(cfgFile.Name())
	})

	Describe("/info", func() {
		Context("when the server is the bootstrap instance", func() {
			BeforeEach(func() {
				cfg := config.Config{
					Bootstrap: "true",
				}
				StartServer(cfg)
			})

			It("returns 'bootstrap': true", func() {
				resp, err := http.Get(fmt.Sprintf("http://%s/info", address))
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))
				respString, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(respString).To(MatchJSON(`{ "bootstrap": "true"}`))
			})
		})

		Context("when the server is not the bootstrap instance", func() {
			BeforeEach(func() {
				cfg := config.Config{
					Bootstrap: "false",
				}
				StartServer(cfg)
			})

			It("returns 'bootstrap': false", func() {
				resp, err := http.Get(fmt.Sprintf("http://%s/info", address))
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))
				respString, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(respString).To(MatchJSON(`{ "bootstrap": "false"}`))
			})
		})
	})

	Describe("/migrate", func() {
		Context("when the server should have migrated", func() {
			BeforeEach(func() {
				bpmFile, err = ioutil.TempFile("", "bpm.yml")
				bpmFile.Write([]byte("bpm.original"))
				bpmv2File, err = ioutil.TempFile("", "bpm.v2.yml")
				bpmv2File.Write([]byte("bpm.version2"))

				cfg := config.Config{
					Bootstrap:           "true",
					NATSBPMConfigPath:   bpmFile.Name(),
					NATSBPMv2ConfigPath: bpmv2File.Name(),
				}
				StartServer(cfg)
			})

			It("should replace the BPM config with v2", func() {
				// before migration
				originalContents, err := ioutil.ReadFile(bpmFile.Name())
				Expect(err).ToNot(HaveOccurred())
				version2Contents, err := ioutil.ReadFile(bpmv2File.Name())
				Expect(err).ToNot(HaveOccurred())

				original := string(originalContents)
				version2 := string(version2Contents)

				Expect(original).To(Equal("bpm.original"))
				Expect(version2).To(Equal("bpm.version2"))
				Expect(original).To(Not(Equal(version2)))

				resp, err := http.Post(fmt.Sprintf("http://%s/migrate", address), "application/json", nil)
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))

				// after migration
				originalContents, err = ioutil.ReadFile(bpmFile.Name())
				Expect(err).ToNot(HaveOccurred())
				version2Contents, err = ioutil.ReadFile(bpmv2File.Name())
				Expect(err).ToNot(HaveOccurred())

				original = string(originalContents)
				version2 = string(version2Contents)
				Expect(version2).To(Equal("bpm.version2"))
				Expect(original).To(Equal(version2))
			})
		})
	})
})

func VerifyTCPConnection(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}
