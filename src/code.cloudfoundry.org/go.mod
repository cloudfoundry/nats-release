module code.cloudfoundry.org

go 1.17

replace github.com/nats-io/gnatsd => github.com/nats-io/gnatsd v1.4.1

require (
	code.cloudfoundry.org/tlsconfig v0.0.0-20200131000646-bbe0f8da39b3
	github.com/nats-io/gnatsd v1.4.1
	github.com/nats-io/go-nats v1.4.0
)

require (
	github.com/golang/protobuf v1.5.2 // indirect
	github.com/nats-io/nuid v1.0.1 // indirect
	golang.org/x/crypto v0.0.0-20181127143415-eb0de9b17e85 // indirect
	golang.org/x/sys v0.0.0-20181128092732-4ed8d59d0b35 // indirect
)
