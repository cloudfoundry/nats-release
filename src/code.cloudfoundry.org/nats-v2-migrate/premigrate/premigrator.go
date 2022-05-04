package premigrate

import (
	"crypto/tls"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cf-networking-helpers/mutualtls"
	"code.cloudfoundry.org/lager"
	bpm_rewriter "code.cloudfoundry.org/nats-v2-migrate/bpm-rewriter"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	nats "code.cloudfoundry.org/nats-v2-migrate/nats-interface"

	natsClient "github.com/nats-io/nats.go"
)

func EnsureNatsConnections(c *config.Config, tlsConfig *tls.Config) ([]nats.NatsConn, error) {
	var natsConns []nats.NatsConn
	var optionFunc natsClient.Option

	for _, url := range c.NATSMachines {
		if c.InternalTLSEnabled {
			if tlsConfig == nil {
				return nil, errors.New("Argument mismatch: InternalTLSEnabled but no TLSConfig specified")
			}

			tlsConfig.ServerName = url
			if optionFunc == nil {
				optionFunc = AddTLSConfig(tlsConfig)
			}
		}
		natsConn, err := natsClient.Connect(fmt.Sprintf("%s:%s@%s:%d", c.NatsUser, c.NatsPassword, url, c.NatsPort), optionFunc)
		if err != nil {
			return nil, err
		}
		natsConns = append(natsConns, natsConn)
	}
	return natsConns, nil
}

func AddTLSConfig(tls *tls.Config) natsClient.Option {
	return func(o *natsClient.Options) error {
		if tls != nil {
			o.TLSConfig = tls
		}
		return nil
	}
}

type PreMigrator struct {
	BpmRewriter bpm_rewriter.Rewriter
	Config      *config.Config
	Logger      lager.Logger
}

func NewPreMigrator(config *config.Config, logger lager.Logger) *PreMigrator {
	return &PreMigrator{
		BpmRewriter: &bpm_rewriter.BPMRewriter{},
		Config:      config,
		Logger:      logger,
	}
}

func (pm *PreMigrator) PrepareForMigration() error {
	pm.Logger.Info(fmt.Sprintf("Starting premigration. Nats instances: %s", pm.Config.NATSMachines))

	if len(pm.Config.NATSMachines) == 0 {
		pm.Logger.Info("Single-instance NATS cluster. Restarting as v2")
		return nil
	}

	// CheckForSingleInstance(/*arg TBD*/)
	// error check

	var tlsConfig *tls.Config = nil
	var err error

	if pm.Config.InternalTLSEnabled {
		tlsConfig, err = mutualtls.NewClientTLSConfig(pm.Config.CertFile, pm.Config.KeyFile, pm.Config.CaFile)
		if err != nil {
			pm.Logger.Error("Error creating TLS config for nats client", err)
			return err
		}
	}

	// CreateClientTLS(config)
	// error check

	natsConns, err := EnsureNatsConnections(pm.Config, tlsConfig)
	if err != nil {
		pm.Logger.Error("Unable to connect to NATs peers to verify existing server version", err)
		return err
	}

	for _, conn := range natsConns {
		version := conn.ConnectedServerVersion()

		pm.Logger.Info(fmt.Sprintf("Finding server version: %s", version))
		semanticVersions := strings.Split(version, ".")
		if len(semanticVersions) < 3 {
			return fmt.Errorf("Unable to determine nats server version: %s", version)
		}

		majorVersion, err := strconv.Atoi(semanticVersions[0])
		if err != nil {
			return fmt.Errorf("Error parsing semantic version: %v\n", err)
		}

		if majorVersion < 2 {
			pm.Logger.Info("Cluster contains at least 1 NATS v1 node. Adding v1 executable.")

			err = pm.BpmRewriter.Rewrite(pm.Config.V1BPMConfigPath, pm.Config.NATSBPMConfigPath)
			if err != nil {
				return fmt.Errorf("Error replacing bpm config %s", err)
			}
			break
		}
	}
	// UpgradeExecutable(natsConns, bpmRewriter)
	// error check

	pm.Logger.Info("Cluster does not contain any NATS v1 nodes. Using v2 executable.")
	return nil
}
