package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"code.cloudfoundry.org/tlsconfig"

	nats "github.com/nats-io/go-nats"
)

// Simple healthcheck app that verifies a connection can be
// made to the locally-running NATS server every ten seconds

func main() {
	address := flag.String("address", "", "")
	port := flag.String("port", "", "")
	user := flag.String("user", "", "")
	password := flag.String("password", "", "")
	serverCAPath := flag.String("server-ca", "", "")
	serverHostname := flag.String("server-hostname", "", "")
	clientCertificatePath := flag.String("client-certificate", "", "")
	clientKeyPath := flag.String("client-private-key", "", "")

	flag.Parse()

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(*clientCertificatePath, *clientKeyPath),
	).Client(
		tlsconfig.WithAuthorityFromFile(*serverCAPath),
	)
	if err != nil {
		log.Fatalf("failed to build tls configuration: %s\n", err)
	}
	tlsConfig.ServerName = *serverHostname

	connectionOptions := []nats.Option{
		nats.Secure(tlsConfig),
		nats.NoReconnect(),
	}

	if *user != "" && *password != "" {
		connectionOptions = append(connectionOptions, nats.UserInfo(*user, *password))
	}

	for {
		connection, err := nats.Connect(
			fmt.Sprintf("nats://%s:%s", *address, *port),
			connectionOptions...,
		)
		if err != nil {
			log.Fatalf("failed to connect to NATS server: %s", err)
		}
		connection.Close()

		time.Sleep(10 * time.Second)
	}
}
