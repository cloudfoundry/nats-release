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

	nats "github.com/nats-io/go-nats"
)

const wantedMessageCount = 10

type config struct {
	Hosts    []string
	User     string
	Password string
	Port     int
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

	var servers []string
	for _, host := range c.Hosts {
		servers = append(servers, fmt.Sprintf("nats://%s:%s@%s:%d", c.User, c.Password, host, c.Port))
	}

	nc, err := nats.Connect(strings.Join(servers, ","))
	if err != nil {
		log.Fatalf("can't connect: %v\n", err)
	}
	defer nc.Close()

	var actualMessageCount int
	_, err = nc.Subscribe("test", func(m *nats.Msg) {
		actualMessageCount++
	})

	for i := 0; i < wantedMessageCount; i++ {
		err = nc.Publish("test", []byte(fmt.Sprintf("message %d", i)))
		if err != nil {
			log.Fatalf("failed to publish message: %v\n", err)
		}
	}

	err = nc.Flush()
	if err != nil {
		log.Fatalf("failed to flush connection: %v\n", err)
	}

	if err = nc.LastError(); err != nil {
		log.Fatalf("last error: %v\n", err)
	}

	timeout := time.After(15 * time.Second)
	tick := time.Tick(500 * time.Millisecond)

	for {
		select {
		case <-timeout:
			log.Fatalf("expected to receive %d messages but only received %d", wantedMessageCount, actualMessageCount)
		case <-tick:
			if wantedMessageCount == actualMessageCount {
				return
			}
		}
	}
}
