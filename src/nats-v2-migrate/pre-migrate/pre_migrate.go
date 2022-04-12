package pre_migrate

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
)

type Config struct {
	NATSMachines      []string `json:"nats_machines"`
	V1BPMConfigPath   string   `json:"v1_bpm_config_path"`
	NATSBPMConfigPath string   `json:"nats_bpm_config_path"`
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

	natsClient := NatsClient
	preMigrator := NewPreMigrator(config.NATSMachines, natsClient, config.V1BPMConfigPath, config.NATSBPMConfigPath)
	preMigrator.PrepareForMigration()
}
