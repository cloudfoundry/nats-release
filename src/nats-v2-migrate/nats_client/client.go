package nats_client

//go:generate counterfeiter -generate
import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
)

//counterfeiter:generate -o fakes/client.go --fake-name NatsClient . NatsClient
type NatsClient interface {
	GetInfo(url string) (NatsInfo, error)
}

type NatsInfo struct {
	ServerId      string
	NatsVersion   string `json:"version"`
	GolangVersion string `json:"go"`
	Host          string
	Port          int32
}

func GetInfo(url string) (*NatsInfo, error) {
	conn, err := net.Dial("tcp", url)
	if err != nil {
		return nil, fmt.Errorf("Error connecting: %v", err)
	}
	status, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		return nil, fmt.Errorf("Error reading: %v", err)
	}

	serverJSON := strings.TrimPrefix(status, "INFO ")
	var natsInfo NatsInfo
	err = json.Unmarshal([]byte(serverJSON), &natsInfo)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshalling json: %v", err)
	}

	return &natsInfo, nil
}
