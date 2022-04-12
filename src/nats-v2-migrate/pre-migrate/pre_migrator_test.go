package pre_migrate_test

import (
	pre_migrate "code.cloudfoundry.org/nats-release/nats-v2-migrate/pre-migrate"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("PreMigrator", func() {
	var (
		natsMachines  []string
		natsBPMPath   string
		natsV1BPMPath string
	)
	BeforeEach(func() {
		natsMachines = []string{
			"ab123.nats.service.cf.internal",
			"de4567.nats.service.cf.internal",
			"fa123.nats.service.cf.internal",
		}
		natsBPMPath = "/var/vcap/jobs/nats-tls/config/bpm.yml"
		natsV1BPMPath = "/var/vcap/jobs/nats-tls/config/bpm.v1.yml"
	})

	Describe("NewPreMigrator", func() {
		pre_migrator := pre_migrate.NewPreMigrator(natsMachines, natsBPMPath, natsV1BPMPath)

		Expect(pre_migrator).To(Equal(pre_migrate.PreMigrator{
			NatsMachines:  natsMachines,
			NatsBpmPath:   natsBPMPath,
			NatsV1BpmPath: natsV1BPMPath,
		}))
	})

	Describe("PrepareForMigration", func() {
		Context("There are nats v1 machines in the cluster", func() {
			It("does replace the bpm config", func() {
				
			})
			})
		)
		Context("There are no nats v1 machines in the cluster", func() {
			It("does not replace the bpm config", func() {
				
			})
			})
		)
		Context("Error contacting nats cluster", func() {
			It("does not replace the bpm config", func() {
				
			})
			})
		)
		Context("Error replacing bpm config", func() {
			It("Merp", func() {
				
			})
			})
		)
	})
		
	})
	)
})
