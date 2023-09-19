package main

import (
	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/lager/v3/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/nats-v2-migrate/natsinfo"
)

type MigrateServerResponse struct {
	Bootstrap bool `json:"bootstrap"`
}

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	var cfg config.Config
	cfg, err := config.NewConfig(*configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	logger, _ := lagerflags.NewFromConfig("nats-fail-v1", lagerflags.LagerConfig{LogLevel: lagerflags.INFO, TimeFormat: lagerflags.FormatRFC3339})
	logger.Info("Starting confirmation that NATS instances are on V2")

	if !cfg.Bootstrap {
		logger.Info("Skipping because instance is not canary")
		return
	}

	majorVersion, err := natsinfo.GetMajorVersion(fmt.Sprintf("%s:%d", cfg.Address, cfg.NATSPort))
	if err != nil {
		logger.Error("Failed to connect to local NATS server", err, nil)
		os.Exit(1)
	}
	logger.Info(fmt.Sprintf("Local nats server version: %d", majorVersion))
	if majorVersion < 2 {
		logger.Info("Local NATS server is on v1; exiting with error", nil)
		os.Exit(1)
	}
	logger.Info("Finished NATS v2 confirmation")
}
