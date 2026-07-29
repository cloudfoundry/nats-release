package main

import (
	"encoding/json"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("loadConfig", func() {
	var cfgFile *os.File

	BeforeEach(func() {
		var err error
		cfgFile, err = os.CreateTemp("", "healthcheck-config-*.json")
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		os.Remove(cfgFile.Name())
	})

	It("loads all fields from the config file", func() {
		cfg := Config{
			Address:           "127.0.0.1",
			Port:              "4224",
			User:              "alice",
			Password:          "secret",
			ServerCA:          "/tmp/ca.pem",
			ServerHostname:    "nats.example.com",
			ClientCertificate: "/tmp/cert.pem",
			ClientPrivateKey:  "/tmp/key.pem",
		}
		Expect(json.NewEncoder(cfgFile).Encode(cfg)).To(Succeed())
		cfgFile.Close()

		got, err := loadConfig(cfgFile.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(got.Address).To(Equal(cfg.Address))
		Expect(got.Password).To(Equal(cfg.Password))
		Expect(got.ServerHostname).To(Equal(cfg.ServerHostname))
	})

	It("returns an error for a missing file", func() {
		_, err := loadConfig("/nonexistent/path.json")
		Expect(err).To(HaveOccurred())
	})

	It("loads config without optional user/password fields", func() {
		cfg := Config{
			Address:           "127.0.0.1",
			Port:              "4224",
			ServerCA:          "/tmp/ca.pem",
			ServerHostname:    "nats.example.com",
			ClientCertificate: "/tmp/cert.pem",
			ClientPrivateKey:  "/tmp/key.pem",
		}
		Expect(json.NewEncoder(cfgFile).Encode(cfg)).To(Succeed())
		cfgFile.Close()

		got, err := loadConfig(cfgFile.Name())
		Expect(err).NotTo(HaveOccurred())
		Expect(got.User).To(BeEmpty())
		Expect(got.Password).To(BeEmpty())
	})
})
