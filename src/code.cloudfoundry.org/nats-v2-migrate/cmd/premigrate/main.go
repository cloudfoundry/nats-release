package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"code.cloudfoundry.org/nats-v2-migrate/natsinfo"
)

type Config struct {
	NATSMachines        []string `json:"nats_machines"`
	NATSV1BPMConfigPath string   `json:"nats_v1_bpm_config_path"`
	NATSBPMConfigPath   string   `json:"nats_bpm_config_path"`
}

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	var config Config
	configBytes, err := ioutil.ReadFile(*configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling config file: %v\n", err)
		os.Exit(1)
	}

	if len(config.NATSMachines) == 0 {
		fmt.Fprintf(os.Stdout, "Single instance NATs cluster. Deploying as V2")
		return
	}
	for _, natsMachineUrl := range config.NATSMachines {
		majorVersion, err := natsinfo.GetMajorVersion(natsMachineUrl)
		if err != nil {
			if _, ok := err.(*natsinfo.ErrConnectingToNATS); ok {
				fmt.Fprintf(os.Stdout, "Ignoring machine %s due to connection error: %v\n", natsMachineUrl, err)
				continue
			}
			fmt.Fprintf(os.Stderr, "Error getting nats version: %v\n", err)
			os.Exit(1)
		}
		if majorVersion < 2 {
			err = replaceBPMConfig(config.NATSV1BPMConfigPath, config.NATSBPMConfigPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error replacing bpm config: %v\n", err)
				os.Exit(1)
			}
			break
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
