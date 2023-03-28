package integration

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"

	"code.cloudfoundry.org/inigo/helpers/certauthority"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/nats-v2-migrate/integration/helpers"
	"code.cloudfoundry.org/nats-v2-migrate/natsinfo"
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
	cfg.NATSMigrateServerCertFile = serverCertFile
	cfg.NATSMigrateServerKeyFile = serverKeyFile
}

func StartServer(cfg config.Config) {
	StartServerWithoutWaiting(cfg)
	address = "127.0.0.1:4242"
	serverIsAvailable := func() error {
		err := VerifyTCPConnection(address)
		if err != nil {
			fmt.Printf(err.Error())
		}
		return err
	}
	Eventually(serverIsAvailable, "60s").Should(Succeed())
}

func StartServerWithoutWaiting(cfg config.Config) {
	cfgFile, err = ioutil.TempFile("", "migrate-config.json")
	Expect(err).NotTo(HaveOccurred())

	cfgJSON, err := json.Marshal(cfg)
	_, err = cfgFile.Write(cfgJSON)
	Expect(err).NotTo(HaveOccurred())

	serverBin, err := gexec.Build("code.cloudfoundry.org/nats-v2-migrate/nats-wrapper", "-buildvcs=false")
	Expect(err).NotTo(HaveOccurred())

	startCmd := exec.Command(serverBin, "-config-file", cfgFile.Name())
	session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
}

func CreateTLSClient(cfg config.Config) http.Client {
	cert, err := tls.LoadX509KeyPair(cfg.NATSMigrateServerCertFile, cfg.NATSMigrateServerKeyFile)
	if err != nil {
		log.Fatalf("Error creating x509 keypair from client cert file %s and client key file", err.Error())
	}

	caCert, err := ioutil.ReadFile(cfg.NATSMigrateServerCAFile)
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

var _ = Describe("NATS Wrapper", func() {
	BeforeEach(func() {
		CreateMockNATS("/tmp/nats-v1.sh", "v1")
		CreateMockNATS("/tmp/nats-v2.sh", "v2")
	})

	AfterEach(func() {
		session.Kill()
		os.Remove(cfgFile.Name())
		os.Remove("/tmp/nats.output")
	})

	Describe("starting nats server", func() {
		Context("when there is only one nats-server", func() {
			BeforeEach(func() {
				cfg = config.Config{
					NATSInstances:   []string{"127.0.0.1:4222"},
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

			It("starts as v2", func() {
				content, err := ioutil.ReadFile("/tmp/nats.output")
				Expect(err).ToNot(HaveOccurred())
				Expect(string(content)).To(ContainSubstring("v2"))
				Expect(string(content)).NotTo(ContainSubstring("v1"))
			})
		})

		Context("when there are multiple nats-servers", func() {
			var natsRunner1, natsRunner2 *helpers.NATSRunner

			BeforeEach(func() {
				natsRunner1 = helpers.NewNATSRunner(int(4225))
				natsRunner2 = helpers.NewNATSRunner(int(4226))

				cfg = config.Config{
					Address:         "127.0.0.1",
					Bootstrap:       true,
					NATSMigratePort: 4242,
					NATSPort:        4222,
					NATSInstances: []string{
						"127.0.0.1:4222",
						natsRunner1.Addr(),
						natsRunner2.Addr(),
					},
					NATSMigrateServers: []string{
						"127.0.0.1:4242",
						natsRunner1.URL(),
						natsRunner2.URL(),
					},
					NATSV1BinPath:  "/tmp/nats-v1.sh",
					NATSV2BinPath:  "/tmp/nats-v2.sh",
					NATSConfigPath: "/tmp/nats-config.json",
				}
				GenerateCerts(&cfg)
				client = CreateTLSClient(cfg)
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

					StartServer(cfg)
				})

				It("starts as v2", func() {
					content, err := ioutil.ReadFile("/tmp/nats.output")
					Expect(err).ToNot(HaveOccurred())
					Expect(string(content)).To(ContainSubstring("v2"))
					Expect(string(content)).NotTo(ContainSubstring("v1"))
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

					StartServer(cfg)
				})

				It("starts as v1", func() {
					content, err := ioutil.ReadFile("/tmp/nats.output")
					Expect(err).ToNot(HaveOccurred())
					Expect(string(content)).To(ContainSubstring("v1"))
					Expect(string(content)).NotTo(ContainSubstring("v2"))
				})
			})

			Context("when it fails to connect to one nats server within the timeout", func() {
				BeforeEach(func() {
					natsRunner2.Start()
					version, err := natsinfo.GetMajorVersion(natsRunner2.Addr())
					Expect(err).NotTo(HaveOccurred())
					Expect(version).To(Equal(2))

					StartServerWithoutWaiting(cfg)
				})

				It("retries for the timeout period", func() {
					Consistently(func() bool {
						_, err := os.Stat("/tmp/nats.output")
						Expect(err).To(HaveOccurred())
						return errors.Is(err, os.ErrNotExist)
					}).WithTimeout(5 * time.Second).Should(BeTrue())

					natsRunner1.StartV1()
					version, err := natsinfo.GetMajorVersion(natsRunner1.Addr())
					Expect(err).NotTo(HaveOccurred())
					Expect(version).To(Equal(1))
					Eventually(func() string {
						content, err := ioutil.ReadFile("/tmp/nats.output")
						if err != nil {
							return ""
						}

						return string(content)
					}).WithTimeout(5 * time.Second).Should(ContainSubstring("v1"))
				})
			})

			Context("when it fails to connect to one nats server and times out", func() {
				BeforeEach(func() {
					natsRunner2.Start()
					version, err := natsinfo.GetMajorVersion(natsRunner2.Addr())
					Expect(err).NotTo(HaveOccurred())
					Expect(version).To(Equal(2))

					StartServerWithoutWaiting(cfg)
				})

				It("starts as v2", func() {
					Eventually(func() string {
						content, err := ioutil.ReadFile("/tmp/nats.output")
						if err != nil {
							return ""
						}

						return string(content)
					}).WithTimeout(12 * time.Second).Should(ContainSubstring("v2"))
				})
			})
		})
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
		var natsRunner1 *helpers.NATSRunner

		BeforeEach(func() {
			natsRunner1 = helpers.NewNATSRunner(int(4225))
			natsRunner1.StartV1()

			cfg = config.Config{
				Bootstrap:       true,
				NATSMigratePort: 4242,
				Address:         "127.0.0.1",
				NATSPort:        4222,
				NATSInstances: []string{
					"127.0.0.1:4222",
					natsRunner1.Addr(),
				},
				NATSMigrateServers: []string{
					"127.0.0.1:4242",
					natsRunner1.URL(),
				},
				NATSV1BinPath:  "/tmp/nats-v1.sh",
				NATSV2BinPath:  "/tmp/nats-v2.sh",
				NATSConfigPath: "/tmp/nats-config.json",
			}
			GenerateCerts(&cfg)
			StartServer(cfg)
			client = CreateTLSClient(cfg)
		})

		AfterEach(func() {
			if natsRunner1 != nil {
				natsRunner1.Stop()
			}
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
