package integration

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"code.cloudfoundry.org/inigo/helpers/certauthority"
	"code.cloudfoundry.org/nats-v2-migrate/config"
)

var (
	err         error
	cfgFile     *os.File
	cfg         config.Config
	address     string
	session     *gexec.Session
	bpmFile     *os.File
	bpmv2File   *os.File
	certDepoDir string
	client      http.Client
)

func GenerateCerts(cfg *config.Config) {
	certDepoDir, err = ioutil.TempDir("", "")
	Expect(err).NotTo(HaveOccurred())

	ca, err := certauthority.NewCertAuthority(certDepoDir, "nats-v2-migrate-ca")
	Expect(err).NotTo(HaveOccurred())

	serverKeyFile, serverCertFile, err := ca.GenerateSelfSignedCertAndKey("server", []string{}, false)
	Expect(err).NotTo(HaveOccurred())

	_, serverCAFile := ca.CAAndKey()
	cfg.NATSMigrateServerCAFile = serverCAFile
	cfg.NATSMigrateServerClientCertFile = serverCertFile
	cfg.NATSMigrateServerClientKeyFile = serverKeyFile
}

func StartServer(cfg config.Config) {
	cfgFile, err = ioutil.TempFile("", "migrate-config.json")
	Expect(err).NotTo(HaveOccurred())

	cfgJSON, err := json.Marshal(cfg)
	_, err = cfgFile.Write(cfgJSON)
	Expect(err).NotTo(HaveOccurred())

	serverBin, err := gexec.Build("code.cloudfoundry.org/nats-v2-migrate/nats-wrapper")
	Expect(err).NotTo(HaveOccurred())

	startCmd := exec.Command(serverBin, "-config-file", cfgFile.Name())
	session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	address = "127.0.0.1:4242"
	serverIsAvailable := func() error {
		err := VerifyTCPConnection(address)
		if err != nil {
			fmt.Printf(err.Error())
		}
		return err
	}
	Eventually(serverIsAvailable, "10s").Should(Succeed())
}

func CreateTLSClient(cfg config.Config) http.Client {
	cert, err := tls.LoadX509KeyPair(cfg.NATSMigrateServerClientCertFile, cfg.NATSMigrateServerClientKeyFile)
	if err != nil {
		log.Fatalf("Error creating x509 keypair from client cert file %s and client key file", err.Error())
	}

	caCert, err := ioutil.ReadFile(cfg.NATSMigrateServerClientCertFile)
	if err != nil {
		log.Fatalf("Error opening cert file, Error: %s", err)
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
			RootCAs:      caCertPool,
		},
	}

	return http.Client{Transport: t, Timeout: 15 * time.Second}
}

func CreateMockNATS(natsPath string, version string) {
	mockNATSScript := `#!/bin/sh 
    echo "` + version + `" > /tmp/nats.output
    sleep 60`

	natsFile, err := os.Create(natsPath)
	err = os.Chmod(natsPath, 0777)
	Expect(err).NotTo(HaveOccurred())
	_, err = io.WriteString(natsFile, mockNATSScript)
	Expect(err).NotTo(HaveOccurred())
	natsFile.Close()
}

var _ = FDescribe("NATS Wrapper", func() {
	BeforeEach(func() {
		CreateMockNATS("/tmp/nats-v1.sh", "v1")
		CreateMockNATS("/tmp/nats-v2.sh", "v2")
	})

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
					NATSV1BinPath:   "/tmp/nats-v1.sh",
					NATSV2BinPath:   "/tmp/nats-v2.sh",
					NATSConfigPath:  "/tmp/nats-config.json",
				}
				GenerateCerts(&cfg)
				StartServer(cfg)
				client = CreateTLSClient(cfg)
			})

			It("returns 'bootstrap': true", func() {
				address = "https://127.0.0.1:4242"
				resp, err := client.Get(fmt.Sprintf("%s/info", address))

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
					NATSV1BinPath:   "/tmp/nats-v1.sh",
					NATSV2BinPath:   "/tmp/nats-v2.sh",
					NATSConfigPath:  "/tmp/nats-config.json",
				}
				GenerateCerts(&cfg)
				StartServer(cfg)
				client = CreateTLSClient(cfg)
			})

			It("returns 'bootstrap': false", func() {
				resp, err := client.Get(fmt.Sprintf("https://%s/info", address))
				Expect(err).ToNot(HaveOccurred())

				Expect(resp.StatusCode).To(Equal(200))
				respString, err := ioutil.ReadAll(resp.Body)
				Expect(err).ToNot(HaveOccurred())
				Expect(respString).To(MatchJSON(`{ "bootstrap": false}`))
			})
		})
	})

	Describe("/migrate", func() {
		BeforeEach(func() {
			cfg = config.Config{
				Bootstrap:       true,
				NATSMigratePort: 4242,
				Address:         "127.0.0.1",
				NATSPort:        4224,
				NATSV1BinPath:   "/tmp/nats-v1.sh",
				NATSV2BinPath:   "/tmp/nats-v2.sh",
				NATSConfigPath:  "/tmp/nats-config.json",
			}
			GenerateCerts(&cfg)
			StartServer(cfg)
			client = CreateTLSClient(cfg)
		})

		Context("when the server should have migrated", func() {
			It("should stop v1 and start v2", func() {
				// before migration
				content, err := ioutil.ReadFile("/tmp/nats.output")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("v1"))
				Expect(string(content)).NotTo(ContainSubstring("v2"))

				resp, err := client.Post(fmt.Sprintf("https://%s/migrate", address), "application/json", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))

				// after migration
				Eventually(func() string {
					content, err = ioutil.ReadFile("/tmp/nats.output")
					Expect(err).ToNot(HaveOccurred())
					return string(content)
				}).Should(ContainSubstring("v2"))
				content, err = ioutil.ReadFile("/tmp/nats.output")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).NotTo(ContainSubstring("v1"))
			})
		})
		Context("when the server has already been migrated", func() {
			It("should succeed the first time and get a 409 the second time", func() {
				resp, err := client.Post(fmt.Sprintf("https://%s/migrate", address), "application/json", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(200))

				resp, err = client.Post(fmt.Sprintf("https://%s/migrate", address), "application/json", nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(resp.StatusCode).To(Equal(409))
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
