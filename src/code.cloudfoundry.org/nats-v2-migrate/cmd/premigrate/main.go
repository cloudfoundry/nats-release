package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"

	bpm_rewriter "code.cloudfoundry.org/nats-v2-migrate/bpm-rewriter"
	nats "code.cloudfoundry.org/nats-v2-migrate/nats-interface"
	"code.cloudfoundry.org/nats-v2-migrate/premigrate"
	natsClient "github.com/nats-io/nats.go"
)

type Config struct {
	NATSMachines      []string `json:"nats_machines"`
	NatsUser          string   `json:"nats_user"`
	NatsPassword      string   `json:"nats_password"`
	NatsPort          int      `json:"nats_port"`
	V1BPMConfigPath   string   `json:"nats_v1_bpm_config_path"`
	NATSBPMConfigPath string   `json:"nats_bpm_config_path"`
	CertFile          string   `json:"nats_cert_path"`
	KeyFile           string   `json:"nats_key_path"`
	CaFile            string   `json:"nats_ca_path"`
}

type NatsServerInfo struct {
	Version string `json:"version"`
	Host    string `host:"host"`
}

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	var config Config
	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		fmt.Printf("Error reading config file: %v\n", err)
		return
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		fmt.Printf("Error unmarshalling config file: %v\n", err)
		return
	}

	logConfig := lagerflags.LagerConfig{
		LogLevel:      string(lagerflags.INFO),
		RedactSecrets: false,
		TimeFormat:    lagerflags.FormatRFC3339,
	}

	logger, _ := lagerflags.NewFromConfig(fmt.Sprintf("nats-v2-migrate"), logConfig)
	logger.Info(fmt.Sprintf("Starting premigration. Nats instances: %s", config.NATSMachines))
	tlsConfig, err := makeTLSConfig(config, logger)
	if err != nil {
		logger.Error("Error making TLS Config", err)
		return
	}

	var natsConns []nats.NatsConn

	for _, url := range config.NATSMachines {

		logger.Info(fmt.Sprintf("Connecting to url %s", url))

		tlsConfig.ServerName = url
		natsConn, err := natsClient.Connect(fmt.Sprintf("%s:%s@%s:%d", config.NatsUser, config.NatsPassword, url, config.NatsPort), natsClient.Secure(tlsConfig))
		if err != nil {
			logger.Error("Error connecting to nats server:", err)
			continue
		}
		natsConns = append(natsConns, natsConn)
	}

	rewriter := bpm_rewriter.BPMRewriter{}

	preMigrator := premigrate.NewPreMigrator(natsConns, &rewriter, config.V1BPMConfigPath, config.NATSBPMConfigPath, logger)
	err = preMigrator.PrepareForMigration()

	if err != nil {
		logger.Error("Premigrate failure: ", err)
	}
}

func makeTLSConfig(config Config, logger lager.Logger) (*tls.Config, error) {
	certFile := config.CertFile
	keyFile := config.KeyFile

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	caFile := config.CaFile

	caCerts, err := ioutil.ReadFile(caFile)
	if err != nil {
		return nil, err
	}

	caPool := x509.NewCertPool()

	if ok := caPool.AppendCertsFromPEM(caCerts); !ok {
		return nil, errors.New("No ca certs appended. Must supply valid CA cert.")

	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS12,
	}
	return tlsConfig, nil
}
