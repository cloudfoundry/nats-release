package nats_interface

import (
	natsClient "github.com/nats-io/nats.go"
)

//go:generate counterfeiter -generate

//go:generate counterfeiter -o ../fakes/nats-client.go --fake-name NatsClient . NatsClient
type NatsClient interface {
	Connect() (natsClient.Conn, error)
}
