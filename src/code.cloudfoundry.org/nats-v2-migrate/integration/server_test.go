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
	err     error
	cfgFile *os.File
	address string
)

func StartServer(cfg config.Config) *gexec.Session {
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

	session, err := gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	address = "http://127.0.0.1:4242"
	// serverIsAvailable := func() error {
	// 	return VerifyTCPConnection(address)
	// }
	// Eventually(serverIsAvailable).Should(Succeed())
	return session
}

var _ = Describe("MigrationServer", func() {
	FDescribe("/info", func() {
		Context("when the server is the bootstrap instance", func() {
			BeforeEach(func() {
				cfg := config.Config{
					NATSBPMv2ConfigPath: "/path/to/bpm.v2.yml",
					NATSBPMConfigPath:   "/path/to/bpm.yml",
					Bootstrap:           "true",
				}
				StartServer(cfg)
			})

			It("returns 'bootstrap': true", func() {
				resp, err := http.Get(fmt.Sprintf("%s/info", address))
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))
				respString, err := ioutil.ReadAll(resp.Body)
				Expect(respString).To(MatchJSON(`{ "bootstrap": "true"}`))
			})
		})

		// Context("when the server is not the bootstrap instance", func() {

		// })

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
