package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	NATSConnectionTimeout       = 10 * time.Second
	NATSConnectionRetryInterval = 1 * time.Second
)

type Config struct {
	NATSMachines        []string `json:"nats_machines"`
	NATSV1BPMConfigPath string   `json:"nats_v1_bpm_config_path"`
	NATSBPMConfigPath   string   `json:"nats_bpm_config_path"`
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
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshalling config file: %v\n", err)
		os.Exit(1)
	}

	if len(config.NATSMachines) == 0 {
		fmt.Fprintf(os.Stderr, "Single instance NATs cluster. Deploying as V2")
		return
	}
	for _, natsMachineUrl := range config.NATSMachines {
		version, err := getNatsServerVersion(natsMachineUrl)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting nats version: %v\n", err)
			os.Exit(1)
		}
		semanticVersions := strings.Split(version, ".")
		if len(semanticVersions) < 3 {
			fmt.Fprintf(os.Stderr, "Version is not normal semantic version\n")
			os.Exit(1)
		}

		majorVersion, err := strconv.Atoi(semanticVersions[0])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing semantic version: %v\n", err)
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

func getNatsServerVersion(natsMachineUrl string) (string, error) {
	conn, err := connectWithRetry(natsMachineUrl)
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

func connectWithRetry(natsMachineUrl string) (conn net.Conn, err error) {
	attempts := int(NATSConnectionTimeout / NATSConnectionRetryInterval)
	for i := 0; i < attempts; i++ {
		conn, err = net.Dial("tcp", natsMachineUrl)
		if err == nil {
			return conn, nil
		}
		time.Sleep(NATSConnectionRetryInterval)
	}
	return nil, err
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
