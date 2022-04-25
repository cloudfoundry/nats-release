package main

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"

	bpm_rewriter "code.cloudfoundry.org/nats-v2-migrate/bpm-rewriter"
	nats "code.cloudfoundry.org/nats-v2-migrate/nats-interface"
	"code.cloudfoundry.org/nats-v2-migrate/premigrate"
	natsClient "github.com/nats-io/nats.go"
)

// TODO: is this used?
type NatsServerInfo struct {
	Version string `json:"version"`
	Host    string `host:"host"`
}

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	logConfig := lagerflags.LagerConfig{
		LogLevel:      string(lagerflags.INFO),
		RedactSecrets: false,
		TimeFormat:    lagerflags.FormatRFC3339,
	}

	logger, _ := lagerflags.NewFromConfig(fmt.Sprintf("nats-v2-migrate"), logConfig)

	c, err := config.InitConfigFromFile(*configFilePath)
	if err != nil {
		logger.Error("Error reading config: ", err)
		return
	}

	logger.Info(fmt.Sprintf("Starting premigration. Nats instances: %s", c.NATSMachines))

	tlsConfig, err := makeTLSConfig(c, logger)
	if err != nil {
		logger.Error("Error making TLS Config", err)
		return
	}

	var natsConns []nats.NatsConn

	for _, url := range c.NATSMachines {

		logger.Info(fmt.Sprintf("Connecting to url %s", url))

		tlsConfig.ServerName = url
		natsConn, err := natsClient.Connect(fmt.Sprintf("%s:%s@%s:%d", c.NatsUser, c.NatsPassword, url, c.NatsPort), natsClient.Secure(tlsConfig))
		if err != nil {
			logger.Error("Error connecting to nats server:", err)
			continue
		}
		natsConns = append(natsConns, natsConn)
	}

	rewriter := bpm_rewriter.BPMRewriter{}

	preMigrator := premigrate.NewPreMigrator(natsConns, &rewriter, c, logger)
	err = preMigrator.PrepareForMigration()

	if err != nil {
		logger.Error("Premigrate failure: ", err)
	}
}

func makeTLSConfig(c *config.Config, logger lager.Logger) (*tls.Config, error) {
	certFile := c.CertFile
	keyFile := c.KeyFile

	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, err
	}

	caFile := c.CaFile

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
