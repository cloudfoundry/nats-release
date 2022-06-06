package integration

import (
	"encoding/json"
	"fmt"
	"io"
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
	cfg       config.Config
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

func StartMockMonit(cfg config.Config) {
	mockMonit := `#!/bin/sh 
	echo $1 >> /tmp/monit-output.txt
	echo " " >> /tmp/monit-output.txt
	echo $2 >> /tmp/monit-output.txt`

	monitScript, err := os.Create("/tmp/monit.sh")
	err = os.Chmod("/tmp/monit.sh", 0777)
	Expect(err).NotTo(HaveOccurred())
	_, err = os.Create("/tmp/monit-output.txt")
	Expect(err).NotTo(HaveOccurred())
	_, err = io.WriteString(monitScript, mockMonit)
	Expect(err).NotTo(HaveOccurred())
	monitScript.Close()
}

var _ = Describe("MigrationServer", func() {
	AfterEach(func() {
		session.Kill()
		os.Remove(cfgFile.Name())
	})

	Describe("/info", func() {
		Context("when the server is the bootstrap instance", func() {
			BeforeEach(func() {
				cfg = config.Config{
					Bootstrap:       true,
					NATSMigratePort: 4242,
				}
				StartServer(cfg)
			})

			It("returns 'bootstrap': true", func() {
				resp, err := http.Get(fmt.Sprintf("http://%s/info", address))

				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))
				respString, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(respString).To(MatchJSON(`{ "bootstrap": true}`))
			})
		})

		Context("when the server is not the bootstrap instance", func() {
			BeforeEach(func() {
				cfg = config.Config{
					Bootstrap:       false,
					NATSMigratePort: 4242,
				}
				StartServer(cfg)
			})

			It("returns 'bootstrap': false", func() {
				resp, err := http.Get(fmt.Sprintf("http://%s/info", address))
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))
				respString, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(respString).To(MatchJSON(`{ "bootstrap": false}`))
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

				cfg = config.Config{
					Bootstrap:           true,
					NATSMigratePort:     4242,
					Address:             "127.0.0.1",
					NATSPort:            4224,
					NATSBPMConfigPath:   bpmFile.Name(),
					NATSBPMv2ConfigPath: bpmv2File.Name(),
					MonitPath:           "/tmp/monit.sh",
				}
				StartServer(cfg)
				StartMockMonit(cfg)
			})

			FIt("should replace the BPM config with v2", func() {
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

				content, err := ioutil.ReadFile("/tmp/monit-output.txt")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).NotTo(ContainSubstring("nats-tls"))

				resp, err := http.Post(fmt.Sprintf("http://%s/migrate", address), "application/json", nil)
				Expect(err).ToNot(HaveOccurred())

				// Nats 1 should be no more
				// Nats 2 should be running

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

				// test that the restart happened and that the nats-tls is running nats-server
				// and we are no longer running gnatsd

				content, err = ioutil.ReadFile("/tmp/monit-output.txt")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("nats-tls"))

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
