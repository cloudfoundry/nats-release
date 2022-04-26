package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"reflect"
)

type Config struct {
	NATSMachines      []string `json:"nats_machines"`
	NatsUser          string   `json:"nats_user"`
	NatsPassword      string   `json:"nats_password"`
	NatsPort          int      `json:"nats_port"`
	V1BPMConfigPath   string   `json:"nats_v1_bpm_config_path"`
	NATSBPMConfigPath string   `json:"nats_bpm_config_path"`
	CertFile          string   `json:"nats_cert_path"`
	KeyFile           string   `json:"nats_key_path"`
	CaFile            string   `json:"nats_ca_path"`
}

func InitConfigFromFile(path string) (*Config, error) {
	var config Config

	configBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error reading config file: %v\n", err))
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error unmarshalling config file: %v\n", err))
	}

	if reflect.DeepEqual(config, Config{}) {
		return nil, errors.New("Config file cannot be empty")
	}

	return &config, nil
}
