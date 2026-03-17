package integration

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"

	"code.cloudfoundry.org/cf-networking-helpers/certauthority"
	"code.cloudfoundry.org/cf-networking-helpers/portauthority"
	"code.cloudfoundry.org/nats-v2-wrapper/config"
)

var (
	cfgFile                                                      *os.File
	cfg                                                          config.Config
	address                                                      string
	session                                                      *gexec.Session
	certDepoDir                                                  string
	client                                                       http.Client
	outputFile, natsV2File                                      string
	natsPort, natsWrapperPort, natsRunnerPort1, natsRunnerPort2 uint16
)

func GenerateCerts(cfg *config.Config) {
	var err error
	certDepoDir, err = os.MkdirTemp("", "cert-depot-dir")
	Expect(err).NotTo(HaveOccurred())

	ca, err := certauthority.NewCertAuthority(certDepoDir, "nats-v2-wrapper-ca")
	Expect(err).NotTo(HaveOccurred())

	serverKeyFile, serverCertFile, err := ca.GenerateSelfSignedCertAndKey("server", []string{}, false)
	Expect(err).NotTo(HaveOccurred())

	_, serverCAFile := ca.CAAndKey()
	cfg.NATSV2WrapperServerCAFile = serverCAFile
	cfg.NATSV2WrapperServerCertFile = serverCertFile
	cfg.NATSV2WrapperServerKeyFile = serverKeyFile
}

func StartServer(cfg config.Config) {
	StartServerWithoutWaiting(cfg)
	address = fmt.Sprintf("127.0.0.1:%d", natsWrapperPort)
	serverIsAvailable := func() error {
		err := VerifyTCPConnection(address)
		if err != nil {
			fmt.Print(err.Error())
		}
		return err
	}
	Eventually(serverIsAvailable, "60s").Should(Succeed())
}

func StartServerWithoutWaiting(cfg config.Config) {
	var err error
	cfgFile, err = os.CreateTemp("", "wrapper-config.json")
	Expect(err).NotTo(HaveOccurred())

	cfgJSON, err := json.Marshal(cfg)
	Expect(err).NotTo(HaveOccurred())
	_, err = cfgFile.Write(cfgJSON)
	Expect(err).NotTo(HaveOccurred())

	serverBin, err := gexec.Build("code.cloudfoundry.org/nats-v2-wrapper/nats-wrapper", "-buildvcs=false")
	Expect(err).NotTo(HaveOccurred())

	startCmd := exec.Command(serverBin, "-config-file", cfgFile.Name())
	session, err = gexec.Start(startCmd, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())
}

func CreateTLSClient(cfg config.Config) http.Client {
	cert, err := tls.LoadX509KeyPair(cfg.NATSV2WrapperServerCertFile, cfg.NATSV2WrapperServerKeyFile)
	if err != nil {
		log.Fatalf("Error creating x509 keypair from client cert file %s and client key file", err.Error())
	}

	caCert, err := os.ReadFile(cfg.NATSV2WrapperServerCAFile)
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

func CreateMockNATS(natsPath string, version string, outputFile string) {
	mockNATSScript := `#!/bin/sh 
    echo "` + version + `" >` + outputFile + `
	sleep 60`

	natsFile, err := os.Create(natsPath)
	Expect(err).NotTo(HaveOccurred())

	err = os.Chmod(natsPath, 0777)
	Expect(err).NotTo(HaveOccurred())

	_, err = io.WriteString(natsFile, mockNATSScript)
	Expect(err).NotTo(HaveOccurred())

	natsFile.Close()
}

var _ = Describe("NATS Wrapper", func() {
	BeforeEach(func() {
		node := GinkgoParallelProcess()
		startPort := 1000 * node
		portRange := 950
		endPort := startPort + portRange

		allocator, err := portauthority.New(startPort, endPort)
		Expect(err).NotTo(HaveOccurred())
		natsPort, err = allocator.ClaimPorts(4)
		Expect(err).NotTo(HaveOccurred())
		natsRestartPort = natsPort + 1
		natsRunnerPort1 = natsPort + 2
		natsRunnerPort2 = natsPort + 3

		file, err := os.CreateTemp("", "output-file-")
		Expect(err).NotTo(HaveOccurred())
		Expect(os.Remove(file.Name())).To(Succeed())
		outputFile = file.Name()

		file, err = os.CreateTemp("", "nats-v2-sh-")
		Expect(err).NotTo(HaveOccurred())
		natsV2File = file.Name()

		CreateMockNATS(natsV2File, "v2", outputFile)
	})

	AfterEach(func() {
		if session != nil {
			session.Kill()
		}
		if cfgFile != nil {
			os.Remove(cfgFile.Name())
		}
		os.Remove(outputFile)
		os.Remove(natsV2File)
		if certDepoDir != "" {
			os.RemoveAll(certDepoDir)
		}
	})

	Describe("starting nats server", func() {
		BeforeEach(func() {
			cfg = config.Config{
				NATSV2WrapperPort: int(natsWrapperPort),
				NATSV2BinPath:     natsV2File,
			}
			GenerateCerts(&cfg)
			StartServer(cfg)
			client = CreateTLSClient(cfg)
		})

		It("starts as v2", func() {
			content, err := os.ReadFile(outputFile)
			Expect(err).ToNot(HaveOccurred())
			Expect(string(content)).To(ContainSubstring("v2"))
		})
	})

	Describe("/shutdown-gracefully", func() {
		BeforeEach(func() {
			cfg = config.Config{
				NATSV2WrapperPort: int(natsWrapperPort),
				NATSV2BinPath:     natsV2File,
			}
			GenerateCerts(&cfg)
			StartServer(cfg)
			client = CreateTLSClient(cfg)
		})

		It("sends a signal to NATS to shutdown", func() {
			resp, err := client.Post(fmt.Sprintf("https://%s/shutdown-gracefully", address), "application/json", nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(resp.StatusCode).To(Equal(200))
			Eventually(session).Should(gbytes.Say("graceful-shutdown-triggered"))
			Eventually(session).Should(gbytes.Say("signalled-nats"))
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
