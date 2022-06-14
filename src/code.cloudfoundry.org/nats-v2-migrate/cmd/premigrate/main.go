package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/nats-v2-migrate/natsinfo"
)

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	var cfg config.Config
	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(configBytes, &cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling config file: %v\n", err)
		os.Exit(1)
	}

	logger, _ := lagerflags.NewFromConfig("nats-premigrate", lagerflags.LagerConfig{LogLevel: lagerflags.INFO, TimeFormat: lagerflags.FormatRFC3339})
	logger.Info("Starting migrate")

	if len(cfg.NATSPeers) == 0 {
		logger.Info("Single instance NATs cluster. Deploying as V2")
		return
	}
	for _, natsMachineUrl := range cfg.NATSPeers {
		majorVersion, err := natsinfo.GetMajorVersion(natsMachineUrl)
		if err != nil {
			if _, ok := err.(*natsinfo.ErrConnectingToNATS); ok {
				logger.Error(fmt.Sprintf("Ignoring machine %s due to connection error", natsMachineUrl), err)
				continue
			}
			logger.Error("Error getting nats version", err)
			os.Exit(1)
		}
		if majorVersion < 2 {
			logger.Info(fmt.Sprintf("Instance %s is on version %d", natsMachineUrl, majorVersion))

			err = replaceBPMConfig(cfg.NATSBPMv1ConfigPath, cfg.NATSBPMConfigPath)
			if err != nil {
				logger.Error("Error replacing bpm config", err)
				os.Exit(1)
			}
			break
		} else {
			logger.Info(fmt.Sprintf("Instance %s is on version %d", natsMachineUrl, majorVersion))
		}
	}
}

func replaceBPMConfig(sourcePath, destinationPath string) error {
	bytesRead, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("Error reading source file: %v", err)
	}

	err = ioutil.WriteFile(destinationPath, bytesRead, 0644)
	if err != nil {
		return fmt.Errorf("Error writing destination file: %v", err)
	}

	return nil
}
