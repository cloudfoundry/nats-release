package nats

//go:generate counterfeiter -generate

//go:generate counterfeiter -o ../fakes/nats.go --fake-name NatsConn . NatsConn
type NatsConn interface {
	ConnectedServerVersion() string
}
