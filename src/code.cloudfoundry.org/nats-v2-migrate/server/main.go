package main

import (
	"bytes"
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
	http.HandleFunc("/restart", restart)
	http.HandleFunc("/shutdown", shutdown)

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
		//shutdownNATS()
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(nil)

	// err = restartNATS()
	// if err != nil {
	// 	fmt.Printf("Failed to restart nats: %s", err.Error())
	// }
}

func restart(w http.ResponseWriter, req *http.Request) {
	restartNATS()
}

func shutdown(w http.ResponseWriter, req *http.Request) {
	shutdownNATS()
}

func restartNATS() error {
	fmt.Fprintf(os.Stdout, "Attempting restart")
	err := withRetries(func() error {
		// cmd := exec.Command("/var/vcap/bosh/bin/monit", "start", "nats-tls")
		cmd := exec.Command("/var/vcap/packages/nats-v2-migrate/bin/restart.sh")
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		return cmd.Run()
	})
	if err != nil {
		fmt.Printf("Error shutting down: %s", err.Error())
		return err
	}
	fmt.Printf("Successfully restarted")
	return nil
}

func shutdownNATS() error {
	fmt.Fprintf(os.Stdout, "Attempting shutdown")
	err := withRetries(func() error {
		// cmd := exec.Command("/var/vcap/bosh/bin/monit", "stop", "nats-tls")
		cmd := exec.Command("/var/vcap/packages/nats-v2-migrate/bin/shutdown.sh")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	})
	if err != nil {
		fmt.Printf("Error shutting down: %s", err.Error())
		return err
	}
	fmt.Printf("Successfully shut down")
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
