package main

import (
	"flag"
	"fmt"
	"os"

	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"

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

	preMigrator := premigrate.NewPreMigrator(c, logger)
	err = preMigrator.PrepareForMigration()

	if err != nil {
		logger.Error("Premigrate failure: ", err)
		os.Exit(1)
	}
}
