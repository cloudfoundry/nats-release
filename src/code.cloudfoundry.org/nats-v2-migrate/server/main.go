package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"
)

var (
	gCfg config.Config
)

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	cfg, err := config.NewConfig(*configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	gCfg = cfg

	logger, _ := lagerflags.NewFromConfig("nats-migrate-server", lagerflags.LagerConfig{LogLevel: lagerflags.INFO, TimeFormat: lagerflags.FormatRFC3339})
	server := NewHttpServer(logger)

	http.HandleFunc("/info", server.Info)
	http.HandleFunc("/migrate", server.Migrate)

	logger.Info("Server listening for migration...")
	http.ListenAndServeTLS(fmt.Sprintf(":%d", cfg.NATSMigratePort), cfg.NATSMigrateServerClientCertFile, cfg.NATSMigrateServerClientKeyFile, nil)
}

type httpServer struct {
	logger             lager.Logger
	migrateEndpointHit bool
}

func NewHttpServer(logger lager.Logger) *httpServer {
	return &httpServer{
		logger:             logger,
		migrateEndpointHit: false,
	}
}

func (s *httpServer) Info(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := make(map[string]bool)

	response["bootstrap"] = gCfg.Bootstrap

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Error("Error during marshal", err)
		w.Write(nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func (s *httpServer) Migrate(w http.ResponseWriter, req *http.Request) {
	// guard against race condition, multiple hits from different
	// instances
	if s.migrateEndpointHit == true {
		w.WriteHeader(http.StatusConflict)
		w.Write(nil)
		return
	}

	s.migrateEndpointHit = true

	err := replaceBPMConfig(gCfg.NATSBPMv2ConfigPath, gCfg.NATSBPMConfigPath)
	if err != nil {
		s.logger.Error("Failed to replace bpm config file", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(nil)
		shutdownNATS(s.logger)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(nil)

	err = restartNATS(s.logger)
	if err != nil {
		s.logger.Error("Failed to restart nats", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(nil)
		shutdownNATS(s.logger)
		return
	}
}

func restartNATS(logger lager.Logger) error {
	logger.Info("Attempting restart")

	err := withRetries(func() error {
		cmd := exec.Command(gCfg.MonitPath, "restart", gCfg.Job)
		return cmd.Run()
	})
	if err != nil {
		logger.Error("Error shutting down", err)
		return err
	}
	logger.Info("Successfully restarted")
	return nil
}

func shutdownNATS(logger lager.Logger) error {
	logger.Info("Attempting shutdown")

	err := withRetries(func() error {
		cmd := exec.Command(gCfg.MonitPath, "stop", gCfg.Job)
		return cmd.Run()
	})

	if err != nil {
		logger.Error("Error shutting down", err)
		return err
	}
	logger.Info("Successfully shut down")
	return nil
}

func withRetries(f func() error) error {
	var err error

	for i := 0; i < 5; i++ {
		err = f()
		if err == nil {
			return nil
		}
	}

	return err
}

func replaceBPMConfig(sourcePath, destinationPath string) error {
	bytesRead, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(destinationPath, bytesRead, 0644)
	if err != nil {
		return err
	}

	return nil
}
