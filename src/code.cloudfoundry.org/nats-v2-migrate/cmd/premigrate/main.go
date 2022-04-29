package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/cf-networking-helpers/mutualtls"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"

	bpm_rewriter "code.cloudfoundry.org/nats-v2-migrate/bpm-rewriter"
	"code.cloudfoundry.org/nats-v2-migrate/premigrate"
)

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
		os.Exit(1)
	}

	logger.Info(fmt.Sprintf("Starting premigration. Nats instances: %s", c.NATSMachines))

	var tlsConfig *tls.Config
	if c.CertFile != "" {
		tlsConfig, err = mutualtls.NewClientTLSConfig(c.CertFile, c.KeyFile, c.CaFile)
		if err != nil {
			logger.Error("Error creating TLS config for nats client", err)
			os.Exit(1)
		}
	}
	if len(c.NATSMachines) == 0 {
		logger.Info("Single-instance NATS cluster. Restarting as v2")
		return
	}

	if c.CertFile != "" {

		natsConns, err := premigrate.EnsureNatsConnections(c)
		if err != nil {
			logger.Error("Unable to connect to NATs peers to verify existing server version", err)
			os.Exit(1)
		}
	} else {

		natsConns, err := premigrate.EnsureNatsConnections(c, tlsConfig)
		if err != nil {
			logger.Error("Unable to connect to NATs peers to verify existing server version", err)
			os.Exit(1)
		}

	}
	natsConns, err := premigrate.EnsureNatsConnections(c, tlsConfig)
	if err != nil {
		logger.Error("Unable to connect to NATs peers to verify existing server version", err)
		os.Exit(1)
	}

	rewriter := bpm_rewriter.BPMRewriter{}

	preMigrator := premigrate.NewPreMigrator(natsConns, &rewriter, c, logger)
	err = preMigrator.PrepareForMigration()

	if err != nil {
		logger.Error("Premigrate failure: ", err)
		os.Exit(1)
	}
}
