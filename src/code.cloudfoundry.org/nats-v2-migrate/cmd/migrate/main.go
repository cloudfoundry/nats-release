package main

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/nats-v2-migrate/natsinfo"
)

type MigrateServerResponse struct {
	Bootstrap bool `json:"bootstrap"`
}

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	var cfg config.Config
	cfg, err := config.NewConfig(*configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	logger, _ := lagerflags.NewFromConfig("nats-migrate", cfg.LagerConfig)
	logger.Info("Starting migrate")

	if len(cfg.NATSMigrateServers) == 0 {
		logger.Info("Single instance NATs cluster. Skipping migration.")
		return
	}

	majorVersion, err := natsinfo.GetMajorVersion(cfg.LocalNATSAddr)
	if err != nil {
		logger.Error("Failed to connect to local NATS server", err, lager.Data{"addr": cfg.LocalNATSAddr})
		os.Exit(1)
	}

	if majorVersion == 2 {
		logger.Info("Local NATS instance has already been migrated to v2. Skipping migration.")
		return
	}

	natsMigrateServerClient, err := newNATSMigrateServerClient(cfg.NATSMigrateServerCAFile, cfg.NATSMigrateServerClientCertFile, cfg.NATSMigrateServerClientKeyFile)
	if err != nil {
		logger.Error("Failed to create NATS migrate server client", err)
		os.Exit(1)
	}

	var bootstrapMigrateServer string
	for _, natsMigrateServer := range cfg.NATSMigrateServers {
		// TODO: maybe retry connecting?
		resp, err := natsMigrateServerClient.Get(natsMigrateServer + "/info")
		if err != nil {
			logger.Info(
				"Failed to connect to NATS migrate server. Assuming it does not have a new version of nats yet.",
				lager.Data{"url": natsMigrateServer, "error": err},
			)
			return
		}
		defer resp.Body.Close()
		var migrateServerResponse MigrateServerResponse
		err = json.NewDecoder(resp.Body).Decode(&migrateServerResponse)
		if err != nil {
			logger.Info(
				"Failed to parse response from NATS migrate server. Assuming it does not have a new version of nats yet.",
				lager.Data{"url": natsMigrateServer, "error": err, "resp": resp},
			)
			return
		}

		if migrateServerResponse.Bootstrap {
			bootstrapMigrateServer = natsMigrateServer
		}

		logger.Debug("Got response", lager.Data{"resp": migrateServerResponse, "url": natsMigrateServer})
	}
	if bootstrapMigrateServer == "" {
		logger.Error("Can't migrate", errors.New("No bootstrap migrate server found"))
		os.Exit(1)
	}

	logger.Info("Migrating bootstrap server", lager.Data{"url": bootstrapMigrateServer})
	// TODO: retry if post fails with err not unexpected status code
	resp, err := natsMigrateServerClient.Post(bootstrapMigrateServer+"/migrate", "application/json", bytes.NewReader([]byte{}))
	if err != nil {
		logger.Error("Failed to migrate bootstrap NATS server", err, lager.Data{"url": bootstrapMigrateServer})
		os.Exit(1)
	}
	if resp.StatusCode != http.StatusOK {
		logger.Error("Failed to migrate bootstrap NATS server", errors.New("Unexpected status code"), lager.Data{"url": bootstrapMigrateServer, "code": resp.StatusCode})
		os.Exit(1)
	}

	wg := sync.WaitGroup{}
	aggregateError := &AggregateError{}

	logger.Info("Migration of bootstrap server succeeded, migrating the rest")
	for _, natsMigrateServerUrl := range cfg.NATSMigrateServers {
		if natsMigrateServerUrl == bootstrapMigrateServer {
			continue
		}
		wg.Add(1)

		go func(serverUrl string) {
			defer wg.Done()
			logger.Info("Migrating server", lager.Data{"url": serverUrl})

			// TODO: retry if post fails with err not unexpected status code
			resp, err := natsMigrateServerClient.Post(serverUrl+"/migrate", "application/json", bytes.NewReader([]byte{}))

			if err != nil {
				err := fmt.Errorf("Failed to migrate NATS server %s: %v", serverUrl, err)
				aggregateError.Append(err)
				return
			}

			if resp.StatusCode != http.StatusOK {
				err := fmt.Errorf("Failed to migrate NATS server %s: Unexpected status code %d", serverUrl, resp.StatusCode)
				aggregateError.Append(err)
			}
		}(natsMigrateServerUrl)
	}

	wg.Wait()
	if aggregateError != nil {
		logger.Error("err", aggregateError)
		os.Exit(1)
	}
	logger.Info("Finished migration")
}

type AggregateError struct {
	errors []error
	mu     sync.Mutex
}

func (es *AggregateError) Append(err error) {
	es.mu.Lock()

	es.errors = append(es.errors, err)
	es.mu.Unlock()
}

func (es *AggregateError) Error() string {
	var errstrings []string
	es.mu.Lock()
	for _, e := range es.errors {
		errstrings = append(errstrings, e.Error())
	}
	es.mu.Unlock()
	return strings.Join(errstrings, ", ")
}

func newNATSMigrateServerClient(caCertFile, clientCertFile, clientKeyFile string) (*http.Client, error) {
	caCert, err := ioutil.ReadFile(caCertFile)
	if err != nil {
		return nil, err
	}
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	cert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:      caCertPool,
				Certificates: []tls.Certificate{cert},
			},
		},
	}
	return client, nil
}

func makeRequest(url string, ch chan<- string) {
	start := time.Now()
	resp, _ := http.Get(url)
	secs := time.Since(start).Seconds()
	body, _ := ioutil.ReadAll(resp.Body)
	ch <- fmt.Sprintf("%.2f elapsed with response length: %d %s", secs, len(body), url)
}
