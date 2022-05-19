package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

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

	http.ListenAndServe(":4242", nil)
}

func info(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := make(map[string]string)
	cfgJson, _ := json.Marshal(gCfg)
	log.Fatal(fmt.Sprintf(string(cfgJson)))

	response["bootstrap"] = gCfg.Bootstrap
	response["bootstrap"] = "whatisgoingon"

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
	err := replaceBPMConfig(gCfg.NATSBPMv2ConfigPath, gCfg.NATSBPMConfigPath)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Fatalf("Error during migration: %s", err)
		w.Write(nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}

func replaceBPMConfig(sourcePath, destinationPath string) error {
	bytesRead, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("Error reading source file: %v", err)
	}

	err = ioutil.WriteFile(destinationPath, bytesRead, 0644)
	if err != nil {
		return fmt.Errorf("Error writing destination file: %v", err)
	}

	return nil
}
