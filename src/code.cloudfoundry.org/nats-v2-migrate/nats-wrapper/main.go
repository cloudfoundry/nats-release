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
	"time"

	"code.cloudfoundry.org/lager/v3"
	"code.cloudfoundry.org/lager/v3/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/nats-v2-migrate/natsinfo"
	"code.cloudfoundry.org/tlsconfig"
	"github.com/tedsuo/ifrit"
	"github.com/tedsuo/ifrit/grouper"
	"github.com/tedsuo/ifrit/http_server"
	"github.com/tedsuo/ifrit/sigmon"
)

const (
	NATSShutdownTimeout = 2 * time.Second
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
	migrateFinished := make(chan error)

	natsBinPath, err := getNATSBinPath(cfg, logger)
	if err != nil {
		logger.Fatal("getting-nats-bin-path", err)
	}

	natsRunner := &NATSRunner{
		Logger:          logger,
		BinPath:         natsBinPath,
		V2BinPath:       cfg.NATSV2BinPath,
		ConfigPath:      cfg.NATSConfigPath,
		MigrateCh:       migrateCh,
		MigrateFinished: migrateFinished,
	}

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(cfg.NATSMigrateServerCertFile, cfg.NATSMigrateServerKeyFile),
	).Server(tlsconfig.WithClientAuthenticationFromFile(cfg.NATSMigrateServerCAFile))
	if err != nil {
		logger.Fatal("tls-configuration-failed", err)
	}

	httpServer := NewHttpServer(logger, cfg, migrateCh, migrateFinished)

	sm := http.NewServeMux()
	sm.HandleFunc("/info", httpServer.Info)
	sm.HandleFunc("/migrate", httpServer.Migrate)

	migrateServer := http_server.NewTLSServer(fmt.Sprintf("0.0.0.0:%d", cfg.NATSMigratePort), sm, tlsConfig)

	members := grouper.Members{
		{Name: "nats-runner", Runner: natsRunner},
		{Name: "migrate-server", Runner: migrateServer},
	}
	group := grouper.NewOrdered(os.Interrupt, members)

	monitor := ifrit.Invoke(sigmon.New(group))
	logger.Info("started")

	err = <-monitor.Wait()
	if err != nil {
		logger.Error("exited-with-failure", err)
		os.Exit(1)
	}
}

func getNATSBinPath(cfg config.Config, logger lager.Logger) (string, error) {
	if len(cfg.NATSInstances) == 1 {
		logger.Info("single-instance-nats-cluster.starting-as-v2")
		return cfg.NATSV2BinPath, nil
	}
	localNATSMachineUrl := fmt.Sprintf("%s:%d", cfg.Address, cfg.NATSPort)
	for _, natsMachineUrl := range cfg.NATSInstances {
		if natsMachineUrl == localNATSMachineUrl {
			continue
		}
		majorVersion, err := natsinfo.GetMajorVersion(natsMachineUrl)
		if err != nil {
			if _, ok := err.(*natsinfo.ErrConnectingToNATS); ok {
				logger.Error("ignoring-machine-due-to-connection-error", err, lager.Data{"url": natsMachineUrl})
				continue
			}
			logger.Error("error-getting-nats-version", err)
			return "", err
		}
		if majorVersion < 2 {
			logger.Info("starting-as-v1", lager.Data{"instance": natsMachineUrl, "version": majorVersion})

			return cfg.NATSV1BinPath, nil
		} else {
			logger.Info("found-v2-instance", lager.Data{"instance": natsMachineUrl, "version": majorVersion})
		}
	}
	return cfg.NATSV2BinPath, nil
}

type NATSRunner struct {
	Logger          lager.Logger
	BinPath         string
	V2BinPath       string
	ConfigPath      string
	MigrateCh       <-chan struct{}
	MigrateFinished chan<- error
}

func (r *NATSRunner) Run(signals <-chan os.Signal, ready chan<- struct{}) error {
	natsSession, err := NewNATSSession(r.BinPath, r.ConfigPath)
	if err != nil {
		return err
	}
	r.Logger.Info("started-nats")

	close(ready)

	for {
		select {
		case <-r.MigrateCh:
			r.Logger.Info("received-migration-signal")
			if r.BinPath == r.V2BinPath {
				r.Logger.Info("skipping-migration-already-on-v2")
				r.MigrateFinished <- nil
				break
			}
			natsSession.Shutdown()

			natsSession, err = NewNATSSession(r.V2BinPath, r.ConfigPath)
			if err != nil {
				r.MigrateFinished <- err
				return err
			}

			r.Logger.Info("migrated-to-v2")
			r.MigrateFinished <- nil
		case signal := <-signals:
			r.Logger.Info("signalled-nats")
			natsSession.Signal(signal)
			return nil
		case <-natsSession.Exited:
			r.Logger.Info("exited-nats")
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

func (s *NATSSession) Shutdown() {
	s.Signal(os.Interrupt)

	t := time.NewTimer(NATSShutdownTimeout)
	select {
	case <-s.Exited:
		return
	case <-t.C:
		s.Signal(os.Kill)
	}
	t.Reset(NATSShutdownTimeout)

	select {
	case <-s.Exited:
		return
	case <-t.C:
		return
	}
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
	migrateFinished       <-chan error
}

func NewHttpServer(logger lager.Logger, cfg config.Config, migrateCh chan<- struct{}, migrateFinished <-chan error) *httpServer {
	return &httpServer{
		logger:                logger,
		migrateEndpointHit:    false,
		migrateEndpointHitMux: &sync.Mutex{},
		cfg:                   cfg,
		migrateCh:             migrateCh,
		migrateFinished:       migrateFinished,
	}
}

func (s *httpServer) Info(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	response := make(map[string]bool)

	response["bootstrap"] = s.cfg.Bootstrap

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Error("error-during-marshal", err)
		w.Write(nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(jsonResponse)
}

func (s *httpServer) Migrate(w http.ResponseWriter, req *http.Request) {
	s.logger.Info("received-migrate-api-call")
	s.migrateEndpointHitMux.Lock()
	defer s.migrateEndpointHitMux.Unlock()
	if s.migrateEndpointHit {
		w.WriteHeader(http.StatusConflict)
		w.Write(nil)
		return
	}

	// migrateEndpointHit var guards against race condition,
	// (multiple hits from different post-start instances)
	s.migrateEndpointHit = true

	s.migrateCh <- struct{}{}
	err := <-s.migrateFinished
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.logger.Error("migration-failed", err)
		w.Write(nil)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write(nil)
}
