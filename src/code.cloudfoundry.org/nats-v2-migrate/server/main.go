package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"

	"code.cloudfoundry.org/nats-v2-migrate/config"
)

var gCfg config.Config

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	cfg, err := config.NewConfig(*configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	gCfg = cfg

	// TODO: maybe just one handler with verb differentiation?
	http.HandleFunc("/info", info)
	http.HandleFunc("/migrate", migrate)

	fmt.Println("Server listening for migration...")
	http.ListenAndServe(fmt.Sprintf(":%d", cfg.NATSMigratePort), nil)
}

func info(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := make(map[string]bool)

	response["bootstrap"] = gCfg.Bootstrap

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("Error during marshal: %s", err)
		w.Write(nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func migrate(w http.ResponseWriter, req *http.Request) {
	fmt.Println("MIGRATE HIT: 2")
	err := replaceBPMConfig(gCfg.NATSBPMv2ConfigPath, gCfg.NATSBPMConfigPath)
	if err != nil {
		fmt.Printf("Failed to replace bpm config file: %s", err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(nil)
		// shutdownNATS()
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(nil)

	// restartNATS()
}

func restartNATS() error {
	err := withRetries(func() error {
		cmd := exec.Command("/var/vcap/jobs/bpm/bin/bpm", "stop", "nats-tls", "-p", "nats-tls")
		return cmd.Run()
	})
	if err != nil {
		return err
	}
	err = withRetries(func() error {
		cmd := exec.Command("/var/vcap/jobs/bpm/bin/bpm", "start", "nats-tls", "-p", "nats-tls")
		return cmd.Run()
	})
	if err != nil {
		return err
	}
	return nil

	// shutdownNATS()
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

func shutdownNATS() {
	for i := 0; i < 5; i++ {
		cmd := exec.Command("monit", "stop", "nats-tls")
		_, err := cmd.Output()
		if err == nil {
			return
		}

		log.Fatalf("Error during stop: %s", err)
	}

	panic("Could not shutdown, panicing...")
}

func replaceBPMConfig(sourcePath, destinationPath string) error {
	bytesRead, err := ioutil.ReadFile(sourcePath)
	fmt.Fprintf(os.Stdout, "Source: %s", sourcePath)
	if err != nil {
		return fmt.Errorf("Error reading source file: %v", err)
	}

	err = ioutil.WriteFile(destinationPath, bytesRead, 0644)
	fmt.Fprintf(os.Stdout, "Destination: %s", destinationPath)
	if err != nil {
		return fmt.Errorf("Error writing destination file: %v", err)
	}
	fmt.Fprintf(os.Stdout, "Success")

	return nil
}
