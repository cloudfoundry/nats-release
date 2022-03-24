package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
)

type Config struct {
	NATSMachines      []string `json:"nats_machines"`
	V1BPMConfigPath   string   `json:"v1_bpm_config_path"`
	NATSBPMConfigPath string   `json:"nats_bpm_config_path"`
}

type NatsServerInfo struct {
	Version string `json:"version"`
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

	for _, natsMachineUrl := range config.NATSMachines {
		version, err := getNatsServerVersion(natsMachineUrl)
		if err != nil {
			fmt.Printf("Error getting nats version: %v\n", err)
			return
		}
		semanticVersions := strings.Split(version, ".")
		if len(semanticVersions) < 3 {
			fmt.Printf("Version is not normal semantic version\n")
			return
		}

		majorVersion, err := strconv.Atoi(semanticVersions[0])
		if err != nil {
			fmt.Printf("Error parsing semantic version: %v\n", err)
		}

		if majorVersion < 2 {
			err = replaceBPMConfig(config.V1BPMConfigPath, config.NATSBPMConfigPath)
			if err != nil {
				fmt.Printf("Error replacing bpm config: %v\n", err)
				return
			}
			break
		}
	}
}

func getNatsServerVersion(natsMachineUrl string) (string, error) {
	conn, err := net.Dial("tcp", natsMachineUrl)
	if err != nil {
		return "", fmt.Errorf("Error connecting: %v", err)
	}
	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("Error reading: %v", err)
	}

	serverJSON := strings.TrimPrefix(status, "INFO ")
	var natsServerInfo NatsServerInfo
	err = json.Unmarshal([]byte(serverJSON), &natsServerInfo)
	if err != nil {
		return "", fmt.Errorf("Error unmarshalling json: %v", err)
	}

	return natsServerInfo.Version, nil
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
