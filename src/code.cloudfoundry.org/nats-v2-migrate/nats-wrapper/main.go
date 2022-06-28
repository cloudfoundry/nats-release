package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

func main() {
	configFilePath := flag.String("config-file", "", "path to config file")
	flag.Parse()

	cfg, err := config.NewConfig(*configFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		os.Exit(1)
	}

	logger, _ := lagerflags.NewFromConfig("nats-migrate-server", lagerflags.LagerConfig{LogLevel: lagerflags.INFO, TimeFormat: lagerflags.FormatRFC3339})

	migrateCh := make(chan struct{})

	natsRunner := &NATSRunner{}

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(cfg.NATSMigrateServerClientCertFile, cfg.NATSMigrateServerClientKeyFile),
	).Server(tlsconfig.WithClientAuthenticationFromFile(cfg.NATSMigrateServerCAFile))
	if err != nil {
		logger.Fatal("tls-configuration-failed", err)
	}

	httpServer := NewHttpServer(logger, cfg, migrateCh)

	sm := http.NewServeMux()
	sm.HandleFunc("/info", httpServer.Info)
	sm.HandleFunc("/migrate", httpServer.Migrate)

	migrateServer := http_server.NewTLSServer(fmt.Sprintf(":%d", cfg.NATSMigratePort), sm, tlsConfig)

	members := grouper.Members{
		"nats-runner": natsRunner,
		"server":      migrateServer,
	}
	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))
	go func() {
		monitor.Signal(os.Interrupt)
	}()

	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}
}

type NATSRunner struct {
	Logger     lager.Logger
	BinPath    string
	V2BinPath  string
	ConfigPath string
	MigrateCh  <-chan struct{}
}

func (r *NATSRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	natsSession, err := NewNATSSession(r.BinPath, r.ConfigPath)
	if err != nil {
		return err
	}

	close(ready)

	for {
		select {
		case <-r.MigrateCh:
			r.Logger.Info("Received migration signal")
			natsSession.Signal(os.Interrupt)

			<-natsSession.Exited // TODO check timeout

			natsSession, err = NewNATSSession(r.V2BinPath, r.ConfigPath)
			if err != nil {
				return err
			}
			r.Logger.Info("Migrated to V2")
		case signal := <-signals:
			natsSession.Signal(signal)
			return nil
		case <-natsSession.Exited:
			if natsSession.ExitCode() == 0 {
				return nil
			}

			return fmt.Errorf("exit status %d", natsSession.ExitCode())
		}
	}
}

type NATSSession struct {
	Exited   <-chan struct{}
	lock     *sync.Mutex
	exitCode int

	command *exec.Cmd
}

func NewNATSSession(binPath string, configPath string) (*NATSSession, error) {
	exited := make(chan struct{})

	session := &NATSSession{
		command:  exec.Command(binPath, "-c", configPath),
		Exited:   exited,
		lock:     &sync.Mutex{},
		exitCode: -1,
	}
	err := session.command.Start()
	if err != nil {
		return nil, err
	}

	go session.waitForExit(exited)

	return session, nil
}

func (s *NATSSession) Signal(signal os.Signal) {
	s.command.Process.Signal(signal)
}

func (s *NATSSession) ExitCode() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.exitCode
}

func (s *NATSSession) waitForExit(exited chan<- struct{}) {
	s.command.Wait()
	status := s.command.ProcessState.Sys().(syscall.WaitStatus)
	s.lock.Lock()
	s.exitCode = status.ExitStatus()
	s.lock.Unlock()
	close(exited)
}

type httpServer struct {
	logger                lager.Logger
	migrateEndpointHit    bool
	migrateEndpointHitMux *sync.Mutex
	cfg                   config.Config
	migrateCh             chan<- struct{}
}

func NewHttpServer(logger lager.Logger, cfg config.Config, migrateCh chan<- struct{}) *httpServer {
	return &httpServer{
		logger:                logger,
		migrateEndpointHit:    false,
		migrateEndpointHitMux: &sync.Mutex{},
		cfg:                   cfg,
		migrateCh:             migrateCh,
	}
}

func (s *httpServer) Info(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := make(map[string]bool)

	response["bootstrap"] = s.cfg.Bootstrap

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
	s.migrateEndpointHitMux.Lock()
	defer s.migrateEndpointHitMux.Unlock()
	if s.migrateEndpointHit == true {
		w.WriteHeader(http.StatusConflict)
		w.Write(nil)
		return
	}

	// migrateEndpointHit var guards against race condition,
	// (multiple hits from different post-start instances)
	s.migrateEndpointHit = true

	close(s.migrateCh)

	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}
