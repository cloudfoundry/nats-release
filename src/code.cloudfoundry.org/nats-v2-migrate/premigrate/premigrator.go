package premigrate

import (
	"fmt"
	"strconv"
	"strings"

	bpm_rewriter "code.cloudfoundry.org/nats-v2-migrate/bpm-rewriter"
	"code.cloudfoundry.org/nats-v2-migrate/nats"
)

type PreMigrator struct {
	NatsConns     []nats.NatsConn
	BpmRewriter   bpm_rewriter.Rewriter
	NatsV1BpmPath string
	NatsBpmPath   string
}

func NewPreMigrator(natsConns []nats.NatsConn, bpmRewriter bpm_rewriter.Rewriter, natsV1BpmPath string, natsBpmPath string) *PreMigrator {
	return &PreMigrator{
		NatsConns:     natsConns,
		BpmRewriter:   bpmRewriter,
		NatsV1BpmPath: natsV1BpmPath,
		NatsBpmPath:   natsBpmPath,
	}
}

func (pm *PreMigrator) PrepareForMigration() error {
	for _, conn := range pm.NatsConns {
		version := conn.ConnectedServerVersion()
		fmt.Printf(version)
		semanticVersions := strings.Split(version, ".")
		if len(semanticVersions) < 3 {
			return fmt.Errorf("Version is not normal semantic version\n")
		}

		majorVersion, err := strconv.Atoi(semanticVersions[0])
		if err != nil {
			fmt.Printf("Error parsing semantic version: %v\n", err)
		}

		if majorVersion < 2 {
			err = pm.BpmRewriter.Rewrite(pm.NatsBpmPath, pm.NatsV1BpmPath)
			if err != nil {
				return fmt.Errorf("Error replacing bpm config: %v\n", err)
			}
			break
		}
	}
	return nil
}
