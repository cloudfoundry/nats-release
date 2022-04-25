package premigrate

import (
	"fmt"
	"strconv"
	"strings"

	"code.cloudfoundry.org/lager"
	bpm_rewriter "code.cloudfoundry.org/nats-v2-migrate/bpm-rewriter"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	nats "code.cloudfoundry.org/nats-v2-migrate/nats-interface"
)

type PreMigrator struct {
	BpmRewriter bpm_rewriter.Rewriter
	Config      *config.Config
	Logger      lager.Logger
	NatsConns   []nats.NatsConn
}

func NewPreMigrator(natsConns []nats.NatsConn, bpmRewriter bpm_rewriter.Rewriter, config *config.Config, logger lager.Logger) *PreMigrator {
	return &PreMigrator{
		NatsConns:   natsConns,
		BpmRewriter: bpmRewriter,
		Config:      config,
		Logger:      logger,
	}
}

func (pm *PreMigrator) PrepareForMigration() error {
	for _, conn := range pm.NatsConns {
		version := conn.ConnectedServerVersion()

		pm.Logger.Info(fmt.Sprintf("Finding server version: %s", version))
		semanticVersions := strings.Split(version, ".")
		if len(semanticVersions) < 3 {
			return fmt.Errorf("Unable to determine nats server version: %s", version)
		}

		majorVersion, err := strconv.Atoi(semanticVersions[0])
		if err != nil {
			fmt.Printf("Error parsing semantic version: %v\n", err)
		}

		if majorVersion < 2 {
			pm.Logger.Info("Cluster contains at least 1 NATS v1 node. Adding v1 executable.")

			err = pm.BpmRewriter.Rewrite(pm.Config.V1BPMConfigPath, pm.Config.NATSBPMConfigPath)
			if err != nil {
				return fmt.Errorf("Error replacing bpm config %s", err)
			}
			break
		}
		fmt.Println("Cluster does not contain any NATS v1 nodes. Using v2 executable.")
	}
	return nil
}
