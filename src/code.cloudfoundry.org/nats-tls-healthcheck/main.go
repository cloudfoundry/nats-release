package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"code.cloudfoundry.org/tlsconfig"
	"github.com/nats-io/nats.go"
)

// Config holds all healthcheck configuration loaded from a JSON file.
type Config struct {
	Address           string `json:"address"`
	Port              string `json:"port"`
	User              string `json:"user,omitempty"`
	Password          string `json:"password,omitempty"`
	ServerCA          string `json:"server_ca"`
	ServerHostname    string `json:"server_hostname"`
	ClientCertificate string `json:"client_certificate"`
	ClientPrivateKey  string `json:"client_private_key"`
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing config file: %w", err)
	}
	return cfg, nil
}

func main() {
	configFile := flag.String("config-file", "", "path to JSON config file")
	flag.Parse()

	if *configFile == "" {
		log.Fatal("--config-file is required")
	}

	cfg, err := loadConfig(*configFile)
	if err != nil {
		log.Fatalf("failed to load config: %s", err)
	}

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(cfg.ClientCertificate, cfg.ClientPrivateKey),
	).Client(
		tlsconfig.WithAuthorityFromFile(cfg.ServerCA),
	)
	if err != nil {
		log.Fatalf("failed to build tls configuration: %s\n", err)
	}
	tlsConfig.ServerName = cfg.ServerHostname

	connectionOptions := []nats.Option{
		nats.Secure(tlsConfig),
		nats.NoReconnect(),
	}

	if cfg.User != "" && cfg.Password != "" {
		connectionOptions = append(connectionOptions, nats.UserInfo(cfg.User, cfg.Password))
	}

	for {
		connection, err := nats.Connect(
			fmt.Sprintf("nats://%s:%s", cfg.Address, cfg.Port),
			connectionOptions...,
		)
		if err != nil {
			log.Fatalf("failed to connect to NATS server: %s", err)
		}
		connection.Close()

		time.Sleep(10 * time.Second)
	}
}
