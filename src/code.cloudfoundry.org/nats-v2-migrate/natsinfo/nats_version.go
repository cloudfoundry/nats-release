package natsinfo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	NATSConnectionTimeout       = 60 * time.Second
	NATSConnectionRetryInterval = 1 * time.Second
)

type NatsServerInfo struct {
	Version string `json:"version"`
}

type ErrConnectingToNATS struct {
	err error
}

func (e *ErrConnectingToNATS) Error() string {
	return fmt.Sprintf("Error connecting: %v", e.err)
}

func GetMajorVersion(natsMachineUrl string) (int, error) {
	conn, err := connectWithRetry(natsMachineUrl)
	if err != nil {
		return 0, &ErrConnectingToNATS{err}
	}
	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return 0, fmt.Errorf("Error reading", err)
	}

	serverJSON := strings.TrimPrefix(status, "INFO ")
	var natsServerInfo NatsServerInfo
	err = json.Unmarshal([]byte(serverJSON), &natsServerInfo)
	if err != nil {
		return 0, fmt.Errorf("Error unmarshalling json", err)
	}

	semanticVersions := strings.Split(natsServerInfo.Version, ".")
	if len(semanticVersions) < 3 {
		return 0, fmt.Errorf("Version is not normal semantic version", err)
	}

	majorVersion, err := strconv.Atoi(semanticVersions[0])
	if err != nil {
		return 0, fmt.Errorf("Error parsing semantic version", err)
	}

	return majorVersion, nil
}

func connectWithRetry(natsMachineUrl string) (conn net.Conn, err error) {
	attempts := int(NATSConnectionTimeout / NATSConnectionRetryInterval)
	for i := 0; i < attempts; i++ {
		fmt.Printf("Attempting local nats server")

		conn, err = net.Dial("tcp", natsMachineUrl)
		if err == nil {
			fmt.Printf("No error, exiting")
			return conn, nil
		}
		fmt.Printf("Error")
		fmt.Printf("%s", natsMachineUrl)
		time.Sleep(NATSConnectionRetryInterval)
	}
	return nil, err
}
