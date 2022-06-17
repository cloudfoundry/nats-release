package config

import (
	"encoding/json"
	"io/ioutil"

	"code.cloudfoundry.org/lager/lagerflags"
)

type Config struct {
	Address                         string   `json:"address"`
	Bootstrap                       bool     `json:"bootstrap"`
	NATSPeers                       []string `json:"nats_peers"`
	NATSPort                        int      `json:"nats_port"`
	NATSMigratePort                 int      `json:"nats_migrate_port"`
	NATSMigrateServers              []string `json:"nats_migrate_servers"`
	NATSMigrateServerCAFile         string   `json:"nats_migrate_server_ca_file"`
	NATSMigrateServerClientCertFile string   `json:"nats_migrate_server_client_cert_file"`
	NATSMigrateServerClientKeyFile  string   `json:"nats_migrate_server_client_key_file"`
	NATSBPMConfigPath               string   `json:"nats_bpm_config_path"`
	NATSBPMv1ConfigPath             string   `json:"nats_bpm_v1_config_path"`
	NATSBPMv2ConfigPath             string   `json:"nats_bpm_v2_config_path"`
	MonitPath                       string   `json:"monit_path"`
	lagerflags.LagerConfig
}

func NewConfig(configPath string) (Config, error) {
	var cfg Config
	configBytes, err := ioutil.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	err = json.Unmarshal(configBytes, &cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}
