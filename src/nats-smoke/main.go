package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"strings"
	"time"

	"code.cloudfoundry.org/tlsconfig"
	nats "github.com/nats-io/go-nats"
)

const wantedMessageCount = 10

type config struct {
	NonTLS struct {
		Hosts    []string
		User     string
		Password string
		Port     int
	}
	TLS struct {
		Hosts       []string
		User        string
		Password    string
		Port        int
		Ca          string
		Certificate string
		PrivateKey  string
	}
}

type pubSubConnection struct {
	sub *nats.Conn
	pub *nats.Conn
}

func main() {
	rawConfig := flag.String("config", "", "")
	flag.Parse()

	readBytes, err := ioutil.ReadFile(*rawConfig)
	if err != nil {
		log.Fatalf("failed to load file: %v\n", err)
	}

	var c config
	err = json.NewDecoder(bytes.NewBuffer(readBytes)).Decode(&c)
	if err != nil {
		log.Fatalf("failed to decode json configuration: %v\n", err)
	}

	tlsConnection, err := tlsConnection(c)
	if err != nil {
		log.Fatalf("failed to connect to tls cluster: %v\n", err)
	}
	if tlsConnection != nil {
		defer tlsConnection.Close()
	}

	plaintextConnection, err := plaintextConnection(c)
	if err != nil {
		log.Fatalf("failed to connect to non-tls cluster: %v\n", err)
	}
	if plaintextConnection != nil {
		defer plaintextConnection.Close()
	}

	conns := createConnPermutations(plaintextConnection, tlsConnection)

ConnPermutations:
	for _, conn := range conns {

		actualMessageCount := 0
		msgChan := make(chan *nats.Msg, 64)

		sub, err := conn.sub.ChanSubscribe("test", msgChan)
		if err != nil {
			log.Fatalf("failed to subscribe to topic: %v\n", err)
		}

		// because we are dealing with async channels, we want to sleep to
		// make sure we subscribe before we start publishing
		time.Sleep(500 * time.Millisecond)

		for i := 0; i < wantedMessageCount; i++ {
			err = conn.pub.Publish("test", []byte(fmt.Sprintf("message %d", i)))
			if err != nil {
				log.Fatalf("failed to publish message: %v\n", err)
			}
		}

		timeout := time.After(15 * time.Second)
		tick := time.Tick(500 * time.Millisecond)

		for {
			select {
			case <-msgChan:
				actualMessageCount++
			case <-timeout:
				log.Printf("Subscriber conn using TLS: %v\n", conn.sub.TLSRequired())
				log.Printf("Publisher conn using TLS: %v\n", conn.pub.TLSRequired())
				log.Fatalf("FAILED - expected to receive %d messages but only received %d", wantedMessageCount, actualMessageCount)
			case <-tick:
				if wantedMessageCount == actualMessageCount {
					err = sub.Unsubscribe()

					if err != nil {
						log.Fatalf("failed to unsubscribe: %v\n", err)
					}

					close(msgChan)
					continue ConnPermutations
				}
			}
		}
	}
	log.Println("SUCCESS")
}

func tlsConnection(c config) (*nats.Conn, error) {
	if len(c.TLS.Hosts) == 0 {
		log.Println("Detected no TLS hosts")
		return nil, nil
	}

	var servers []string
	for _, host := range c.TLS.Hosts {
		servers = append(servers, fmt.Sprintf("nats://%s:%s@%s:%d", c.TLS.User, c.TLS.Password, host, c.TLS.Port))
	}

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(c.TLS.Certificate, c.TLS.PrivateKey),
	).Client(
		tlsconfig.WithAuthorityFromFile(c.TLS.Ca),
	)
	if err != nil {
		log.Fatalf("failed to build tls configuration: %v\n", err)
	}

	return nats.Connect(strings.Join(servers, ","), nats.Secure(tlsConfig))
}

func plaintextConnection(c config) (*nats.Conn, error) {
	if len(c.NonTLS.Hosts) == 0 {
		log.Println("Detected no non-TLS hosts")
		return nil, nil
	}

	var servers []string
	for _, host := range c.NonTLS.Hosts {
		servers = append(servers, fmt.Sprintf("nats://%s:%s@%s:%d", c.NonTLS.User, c.NonTLS.Password, host, c.NonTLS.Port))
	}

	return nats.Connect(strings.Join(servers, ","))
}

func createConnPermutations(plaintextConnection, tlsConnection *nats.Conn) []pubSubConnection {
	conns := make([]pubSubConnection, 0, 4)

	if plaintextConnection != nil {
		log.Println("Testing plaintext connections")
		conns = append(
			conns,
			pubSubConnection{
				pub: plaintextConnection,
				sub: plaintextConnection,
			},
		)
	}

	if tlsConnection != nil {
		log.Println("Testing TLS connections")
		conns = append(
			conns,
			pubSubConnection{
				pub: tlsConnection,
				sub: tlsConnection,
			},
		)
	}

	if plaintextConnection != nil && tlsConnection != nil {
		log.Println("Testing combinations of plaintext and TLS connections")
		conns = append(
			conns,
			pubSubConnection{
				pub: plaintextConnection,
				sub: tlsConnection,
			},
			pubSubConnection{
				pub: tlsConnection,
				sub: plaintextConnection,
			},
		)
	}

	return conns
}
