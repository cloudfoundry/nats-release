module code.cloudfoundry.org/nats-release

go 1.16

replace github.com/nats-io/gnatsd => github.com/nats-io/gnatsd v1.4.1

require (
	code.cloudfoundry.org/tlsconfig v0.0.0-20200131000646-bbe0f8da39b3
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/nats-io/gnatsd v1.4.1
	github.com/nats-io/go-nats v1.4.0
	github.com/nats-io/nuid v1.0.1 // indirect
)
