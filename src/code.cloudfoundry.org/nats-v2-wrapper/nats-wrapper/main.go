package main

import (
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
	"code.cloudfoundry.org/nats-v2-wrapper/config"
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

	logger, _ := lagerflags.NewFromConfig("nats-wrapper", lagerflags.LagerConfig{LogLevel: lagerflags.INFO, TimeFormat: lagerflags.FormatRFC3339})

	natsRunner := &NATSRunner{
		Logger:     logger,
		BinPath:    cfg.NATSV2BinPath,
		ConfigPath: cfg.NATSConfigPath,
	}

	tlsConfig, err := tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(cfg.NATSV2WrapperServerCertFile, cfg.NATSV2WrapperServerKeyFile),
	).Server(tlsconfig.WithClientAuthenticationFromFile(cfg.NATSV2WrapperServerCAFile))
	if err != nil {
		logger.Fatal("tls-configuration-failed", err)
	}

	sm := http.NewServeMux()
	sm.HandleFunc("POST /shutdown-gracefully", func(w http.ResponseWriter, r *http.Request) {
		logger.Info("graceful-shutdown-triggered")
		w.WriteHeader(http.StatusOK)
		// #nosec G104 - don't handle error writing http response. at best, we could log it and be susceptible to DoS's filling up disks with logs
		w.Write(nil)
		syscall.Kill(os.Getpid(), syscall.SIGINT) // SIGINT stops the nats server gracefully
		return
	})

	wrapperServer := http_server.NewTLSServer(fmt.Sprintf("0.0.0.0:%d", cfg.NATSV2WrapperPort), sm, tlsConfig)

	members := grouper.Members{
		{Name: "nats-runner", Runner: natsRunner},
		{Name: "wrapper-server", Runner: wrapperServer},
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

type NATSRunner struct {
	Logger     lager.Logger
	BinPath    string
	ConfigPath string
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

	session.command.Stdout = os.Stdout
	session.command.Stderr = os.Stderr

	err := session.command.Start()
	if err != nil {
		return nil, err
	}

	go session.waitForExit(exited)

	return session, nil
}

func (s *NATSSession) Signal(signal os.Signal) {
	// #nosec G104 - ignore errors signaling the proces. it's ok if it's already shutdown
	s.command.Process.Signal(signal)
}

func (s *NATSSession) ExitCode() int {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.exitCode
}

func (s *NATSSession) waitForExit(exited chan<- struct{}) {
	// #nosec G104 - before calling waitForExit we check to ensure the process started, and we check exit code further down.
	s.command.Wait()
	status := s.command.ProcessState.Sys().(syscall.WaitStatus)
	s.lock.Lock()
	s.exitCode = status.ExitStatus()
	s.lock.Unlock()
	close(exited)
}
