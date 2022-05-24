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

	logger, _ := lagerflags.NewFromConfig("nats-migrate", lagerflags.LagerConfig{LogLevel: lagerflags.INFO, TimeFormat: lagerflags.FormatRFC3339})
	logger.Info("Starting migrate")

	if len(cfg.NATSPeers) == 0 {
		logger.Info("Single instance NATs cluster. Skipping migration.")
		return
	}

	majorVersion, err := natsinfo.GetMajorVersion(fmt.Sprintf("127.0.0.1:%s", cfg.NATSPort))
	if err != nil {
		logger.Error("Failed to connect to local NATS server", err, nil)
		os.Exit(1)
	}
	logger.Info(fmt.Sprintf("Local nats server version: %d", majorVersion))

	if majorVersion == 2 {
		logger.Info("Local NATS instance has already been migrated to v2. Skipping migration.")
		return
	}

	natsMigrateServerClient, err := newNATSMigrateServerClient(cfg.NATSMigrateServerCAFile, cfg.NATSMigrateServerClientCertFile, cfg.NATSMigrateServerClientKeyFile)
	if err != nil {
		logger.Error("Failed to create NATS migrate server client", err)
		os.Exit(1)
	}

	// TODO: set retryCount in config?
	retryCount := 3
	var bootstrapMigrateServer string

	logger.Info("Checking migration info...")
	for _, peer := range cfg.NATSPeers {
		natsMigrateServer := fmt.Sprintf("%s:%s", peer, cfg.NATSMigratePort)
		for i := 0; i < retryCount; i++ {
			logger.Info(fmt.Sprintf("Try #%d", i))
			migrateServerResponse, err := CheckMigrationInfo(natsMigrateServerClient, natsMigrateServer)
			if err != nil {
				logger.Error("Error connecting to NATS server", err, lager.Data{"url": natsMigrateServer})

				if i == retryCount-1 {
					// exceeded retry count, fail the deploy
					os.Exit(1)
				}
				continue
			} else {

				if migrateServerResponse.Bootstrap {
					bootstrapMigrateServer = peer
				}

				logger.Info("Got response", lager.Data{"resp": migrateServerResponse, "url": natsMigrateServer})
				break
			}
		}
	}

	if bootstrapMigrateServer == "" {
		logger.Error("Can't migrate", errors.New("No bootstrap migrate server found"))
		os.Exit(1)
	}

	logger.Info("Migrating bootstrap server", lager.Data{"url": bootstrapMigrateServer})
	// TODO: retry if post fails with err not unexpected status code

	for i := 0; i < retryCount; i++ {
		err = PerformMigration(natsMigrateServerClient, bootstrapMigrateServer)
		if err == nil {
			break
		}

		if i == retryCount-1 {
			// exceeded retry count, fail the deploy
			os.Exit(1)
		}

		usce, ok := err.(*UnexpectedStatusCodeError)
		if ok {
			logger.Error("Unexpected Status Code: ", err, lager.Data{"url": usce.ServerUrl, "code": usce.StatusCode})
			os.Exit(1)
		}
	}

	wg := sync.WaitGroup{}
	aggregateError := &AggregateError{}

	logger.Info("Migration of bootstrap server succeeded, migrating the rest")
	for _, natsMigrateServerUrl := range cfg.NATSPeers {
		if natsMigrateServerUrl == bootstrapMigrateServer {
			continue
		}
		wg.Add(1)

		go func(serverUrl string) {
			defer wg.Done()
			logger.Info("Migrating server", lager.Data{"url": serverUrl})

			for i := 0; i < retryCount; i++ {
				logger.Info(fmt.Sprintf("Try #%v", i))
				err = PerformMigration(natsMigrateServerClient, serverUrl)
				if err == nil {
					break
				}

				if err != nil {
					usce, ok := err.(*UnexpectedStatusCodeError)
					if ok {
						logger.Error("Unexpected Status Code: ", err, lager.Data{"url": usce.ServerUrl, "code": usce.StatusCode})
						return
					}

					if i == retryCount-1 {
						// exceeded retry count, fail the instance
						logger.Error("Exceeded retrying count; failing this instances but other instances may migrate", err, lager.Data{"url": serverUrl})
						return
					}
					logger.Error("Error migrating server, retrying: ", err, lager.Data{"url": serverUrl})
					aggregateError.Append(err)
				}
			}
		}(natsMigrateServerUrl)
	}

	wg.Wait()
	if aggregateError != nil {
		logger.Error("Some nats instances failed to migrate. Re-start the following VMs to finish migraation: ", aggregateError)
		os.Exit(1)
	}
	logger.Info("Finished migration")
}

func CheckMigrationInfo(natsMigrateServerClient *http.Client, serverUrl string) (*MigrateServerResponse, error) {
	resp, err := natsMigrateServerClient.Get(serverUrl + "/info")

	if err != nil {
		return nil, errors.New("Failed to connect to NATS migrate server. Assuming it does not have a new version of nats yet.")
	}
	defer resp.Body.Close()

	var migrateServerResponse MigrateServerResponse
	err = json.NewDecoder(resp.Body).Decode(&migrateServerResponse)
	if err != nil {
		return nil, errors.New("Failed to parse response from NATS migrate server. Assuming it does not have a new version of nats yet.")
	}

	return &migrateServerResponse, nil
}

func PerformMigration(natsMigrateServerClient *http.Client, serverUrl string) error {
	resp, err := natsMigrateServerClient.Post(serverUrl+"/migrate", "application/json", bytes.NewReader([]byte{}))
	if err != nil {
		return errors.New(fmt.Sprintf("Failed to migrate NATS server: %s", serverUrl))
	}

	if resp.StatusCode != http.StatusOK {
		return &UnexpectedStatusCodeError{
			ServerUrl:  serverUrl,
			StatusCode: resp.StatusCode,
		}
	}

	return nil
}

type UnexpectedStatusCodeError struct {
	ServerUrl  string
	StatusCode int
}

func (usce *UnexpectedStatusCodeError) Error() string {
	return fmt.Sprintf("Failed to migrate NATS server (Unexpected status code): %s, %v", usce.ServerUrl, usce.StatusCode)
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
