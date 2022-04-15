package pre_migrate

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	bpm_rewriter "code.cloudfoundry.org/nats-v2-migrate/bpm-rewriter"
	"code.cloudfoundry.org/nats-v2-migrate/nats"
	natsClient "github.com/nats-io/nats.go"
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

	var natsConns []nats.NatsConn
	for _, url := range config.NATSMachines {
		natsConn, err := natsClient.Connect(url)
		if err != nil {
			fmt.Sprintf("Error connecting to nats sever: %s ", err)
		}
		natsConns = append(natsConns, natsConn)
	}

	rewriter := bpm_rewriter.BPMRewriter{}

	preMigrator := NewPreMigrator(natsConns, &rewriter, config.V1BPMConfigPath, config.NATSBPMConfigPath)
	preMigrator.PrepareForMigration()
}
