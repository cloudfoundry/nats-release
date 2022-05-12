package config

import (
	"encoding/json"
	"io/ioutil"

	"code.cloudfoundry.org/lager/lagerflags"
)

type Config struct {
	NATSMigrateServers              []string `json:"nats_migrate_servers"`
	NATSMigrateServerCAFile         string   `json:"nats_migrate_server_ca_file"`
	NATSMigrateServerClientCertFile string   `json:"nats_migrate_server_client_cert_file"`
	NATSMigrateServerClientKeyFile  string   `json:"nats_migrate_server_client_key_file"`
	LocalNATSAddr                   string   `json:"local_nats_addr"`
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
