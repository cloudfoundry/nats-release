package config

import (
	"encoding/json"
	"os"

	"code.cloudfoundry.org/lager/v3/lagerflags"
)

type Config struct {
	NATSV2WrapperPort           int    `json:"nats_v2_wrapper_port"`
	NATSV2WrapperServerCAFile   string `json:"nats_v2_wrapper_server_ca_file"`
	NATSV2WrapperServerCertFile string `json:"nats_v2_wrapper_server_cert_file"`
	NATSV2WrapperServerKeyFile  string `json:"nats_v2_wrapper_server_key_file"`
	NATSV2BinPath               string `json:"nats_v2_bin_path"`
	NATSConfigPath              string `json:"nats_config_path"`
	lagerflags.LagerConfig
}

func NewConfig(configPath string) (Config, error) {
	var cfg Config
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return Config{}, err
	}

	err = json.Unmarshal(configBytes, &cfg)
	if err != nil {
		return Config{}, err
	}

	return cfg, nil
}
