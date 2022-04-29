package config_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/nats-v2-migrate/config"
)

var _ = Describe("Config", func() {
	Describe("InitConfigFromFile", func() {
		It("Works for valid config", func() {
			expected := config.Config{
				NATSMachines:      []string{"772882e1-ea1e-4b0c-b010-5dfd41e680d1.nats.service.cf.internal", "c4e7c684-b986-48b8-8339-38989a6fed67.nats.service.cf.internal"},
				NatsUser:          "nats",
				NatsPassword:      "s72cHitFzjvFVtGfJXQVG91Q7X7Jpl",
				NatsPort:          4224,
				V1BPMConfigPath:   "/var/vcap/jobs/nats-tls/config/bpm.v1.yml",
				NATSBPMConfigPath: "/var/vcap/jobs/nats-tls/config/bpm.yml",
				CertFile:          "/var/vcap/jobs/nats-tls/config/client_tls/certificate.pem",
				KeyFile:           "/var/vcap/jobs/nats-tls/config/client_tls/private_key.pem",
				CaFile:            "/var/vcap/jobs/nats-tls/config/external_tls/ca.pem",
			}

			configPath := "./test_config.json"
			actual, err := config.InitConfigFromFile(configPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(&expected))
		})

		It("Works for no nats peers", func() {
			expected := config.Config{
				NATSMachines:      []string{},
				NatsUser:          "nats",
				NatsPassword:      "s72cHitFzjvFVtGfJXQVG91Q7X7Jpl",
				NatsPort:          4224,
				V1BPMConfigPath:   "/var/vcap/jobs/nats-tls/config/bpm.v1.yml",
				NATSBPMConfigPath: "/var/vcap/jobs/nats-tls/config/bpm.yml",
				CertFile:          "/var/vcap/jobs/nats-tls/config/client_tls/certificate.pem",
				KeyFile:           "/var/vcap/jobs/nats-tls/config/client_tls/private_key.pem",
				CaFile:            "/var/vcap/jobs/nats-tls/config/external_tls/ca.pem",
			}

			configPath := "./empty_peers_config.json"
			actual, err := config.InitConfigFromFile(configPath)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(&expected))
		})
		Context("when it fails to read the file", func() {
			It("returns an error", func() {
				configPath := "./does_not_exist.json"
				_, err := config.InitConfigFromFile(configPath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when unmarshaling fails", func() {
			It("returns an error", func() {
				configPath := "./invalid_json.json"
				_, err := config.InitConfigFromFile(configPath)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when the config file is empty", func() {
			It("returns an error", func() {
				configPath := "./empty_json.json"
				_, err := config.InitConfigFromFile(configPath)
				Expect(err).To(HaveOccurred())
			})
		})
	})
})
