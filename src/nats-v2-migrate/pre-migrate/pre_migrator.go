package pre_migrate

import (
	"code.cloudfoundry.org/nats-release/nats-v2-migrate/nats_client"
)

type PreMigrator struct {
	NatsMachines  []string
	NatsClient    nats_client.NatsClient
	NatsV1BpmPath string
	NatsBpmPath   string
}

func NewPreMigrator(natsMachines []string, natsClient nats_client.NatsClient, natsV1BpmPath string, natsBpmPath string) *PreMigrator {
	return &PreMigrator{
		NatsMachines:  natsMachines,
		NatsClient:    natsClient,
		NatsV1BpmPath: natsV1BpmPath,
		NatsBpmPath:   natsBpmPath,
	}
}

func PrepareForMigration() {
}
