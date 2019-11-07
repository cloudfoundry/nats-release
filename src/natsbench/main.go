package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"

	"code.cloudfoundry.org/tlsconfig"
	"github.com/nats-io/nats.go"

	"natsbench/natsbench"
)

var configPath = flag.String(
	"config",
	"",
	"path to configuration file",
)

func main() {
	flag.Parse()

	if *configPath == "" {
		log.Fatal("--config required")
	}

	c, err := loadConfig(*configPath)
	if err != nil {
		log.Fatalf("failed to load config file: %s", err)
	}

	noTLS := natsbench.NewBenchmark(
		"NATS (No TLS)",
		natsbench.WithAuth(c.NonTLS.User, c.NonTLS.Password),
		natsbench.WithHosts(c.NonTLS.Hosts...),
		natsbench.WithPort(c.NonTLS.Port),
	)
	noTLS.Run()

	if c.hasTLS() {
		tlsConf, err := tlsconfig.Build(
			tlsconfig.WithInternalServiceDefaults(),
			tlsconfig.WithIdentityFromFile(c.TLS.Certificate, c.TLS.PrivateKey),
		).Client(
			tlsconfig.WithAuthorityFromFile(c.TLS.Ca),
		)
		if err != nil {
			log.Fatal(err)
		}

		tls := natsbench.NewBenchmark(
			"NATS (TLS)",
			natsbench.WithAuth(c.TLS.User, c.TLS.Password),
			natsbench.WithHosts(c.TLS.Hosts...),
			natsbench.WithPort(c.TLS.Port),
			natsbench.WithNATSOptions(
				nats.Secure(tlsConf),
			),
		)
		tls.Run()
	}
}

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

func (c config) hasTLS() bool {
	return len(c.TLS.Hosts) > 0
}

func loadConfig(path string) (config, error) {
	bs, err := ioutil.ReadFile(path)
	if err != nil {
		return config{}, fmt.Errorf("failed to read file: %w", err)
	}

	var c config
	if err := json.Unmarshal(bs, &c); err != nil {
		return config{}, fmt.Errorf("failed to decode json: %w", err)
	}

	return c, nil
}
