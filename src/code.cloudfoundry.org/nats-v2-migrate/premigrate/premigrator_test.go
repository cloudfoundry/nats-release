package premigrate_test

import (
	"crypto/tls"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/nats-v2-migrate/fakes"
	nats "code.cloudfoundry.org/nats-v2-migrate/nats-interface"
	"code.cloudfoundry.org/nats-v2-migrate/premigrate"
)

var _ = Describe("EnsureNatsConnections", func() {
	var (
		premigrateConfig config.Config
		tlsConfig        tls.Config

		nc *fakes.NatsClient

		// natsConn1 nats.NatsConn
		// natsConn2 nats.NatsConn
	)
	BeforeEach(func() {
		premigrateConfig = config.Config{
			NATSMachines: []string{"1.nats.url", "2.nats.url"},
			NatsUser:     "nats",
			NatsPassword: "some-password",
			NatsPort:     4224,
		}
		tlsConfig = tls.Config{}

		nc = &fakes.NatsClient{}
	})

	Context("When every connection is successful", func() {
		BeforeEach(func() {
			nc.ConnectReturnsOnCall(0, _, nil)
			nc.ConnectReturnsOnCall(1, _, nil)
		})

		FIt("returns the array of connection objects", func() {
			_, err = premigrate.EnsureNatsConnections(premigrateConfig, tlsConfig)

			// TODO can we fake a nats connetion for asserting expected the result?
			//Expect(result[0]).To(Equal(&natsConn1))
			expectedUrl, _ := nc.ConnectArgsForCall(0)
			Expect(expectedUrl).To(Equal("nats:some-password@1.nats.url:4224"))
			// Can we assert on the options tlsConfig.ServerName?
			//Expect(expectedOptions).To(Equal())

			expectedUrl, _ = nc.ConnectArgsForCall(1)
			Expect(expectedUrl).To(Equal("nats:some-password@2.nats.url:4224"))
			//	Expect(result[1]).To(Equal(&natsConn2))

			Expect(err).To(NotTo(HaveOccurred()))

			Expect(nc.ConnectCallCount).To(Equal(2))
		})

	})
	// Context("When at least one connection is unsuccessful", func() {
	// 	It("returns an error", func() {
	// 	)

	// })
})

var _ = Describe("PreMigrator", func() {
	var (
		natsConn1   *fakes.NatsConn
		natsConn2   *fakes.NatsConn
		natsConn3   *fakes.NatsConn
		natsConns   []nats.NatsConn
		rewriter    *fakes.Rewriter
		c           config.Config
		premigrator *premigrate.PreMigrator
		logger      lager.Logger
	)
	BeforeEach(func() {
		c = config.Config{
			V1BPMConfigPath:   "/var/vcap/jobs/nats-tls/config/bpm.v1.yml",
			NATSBPMConfigPath: "/var/vcap/jobs/nats-tls/config/bpm.yml",
		}
		natsConn1 = &fakes.NatsConn{}
		natsConn2 = &fakes.NatsConn{}
		natsConn3 = &fakes.NatsConn{}
		rewriter = &fakes.Rewriter{}
		logger = lager.NewLogger("nats-test-logger")
	})

	Describe("NewPreMigrator", func() {
		It("Creates the PreMigrator object with expeted properties", func() {

			natsConns = []nats.NatsConn{natsConn1, natsConn2, natsConn3}
			rewriter = &fakes.Rewriter{}
			premigrator = premigrate.NewPreMigrator(natsConns, rewriter, &c, logger)
			Expect(premigrator).To(Equal(&premigrate.PreMigrator{
				NatsConns:   natsConns,
				BpmRewriter: rewriter,
				Config:      &c,
			}))
		})
	})

	Describe("PrepareForMigration", func() {
		JustBeforeEach(func() {
			natsConns = []nats.NatsConn{natsConn1, natsConn2, natsConn3}
			premigrator = premigrate.NewPreMigrator(natsConns, rewriter, &c, logger)
		})

		Context("There are nats v1 machines in the cluster", func() {
			BeforeEach(func() {
				natsConn1.ConnectedServerVersionReturns("1.1.1")
			})

			It("does replace the bpm config", func() {
				err := premigrator.PrepareForMigration()
				Expect(err).NotTo(HaveOccurred())

				Expect(natsConn1.ConnectedServerVersionCallCount()).To(Equal(1))
				Expect(natsConn2.ConnectedServerVersionCallCount()).To(Equal(0))
				Expect(natsConn3.ConnectedServerVersionCallCount()).To(Equal(0))

				Expect(rewriter.RewriteCallCount()).To(Equal(1))
				arg1, arg2 := rewriter.RewriteArgsForCall(0)
				Expect(arg1).To(Equal("/var/vcap/jobs/nats-tls/config/bpm.v1.yml"))
				Expect(arg2).To(Equal("/var/vcap/jobs/nats-tls/config/bpm.yml"))
			})
		})

		Context("There are some nats v1 and some v2 machines in the cluster", func() {
			BeforeEach(func() {
				natsConn1.ConnectedServerVersionReturns("2.2.2")
				natsConn2.ConnectedServerVersionReturns("1.1.1")
			})

			It("does replace the bpm config", func() {
				err := premigrator.PrepareForMigration()
				// premigrator.UpgradfeExecutable()
				Expect(err).NotTo(HaveOccurred())

				Expect(natsConn1.ConnectedServerVersionCallCount()).To(Equal(1))
				Expect(natsConn2.ConnectedServerVersionCallCount()).To(Equal(1))
				Expect(natsConn3.ConnectedServerVersionCallCount()).To(Equal(0))

				Expect(rewriter.RewriteCallCount()).To(Equal(1))
				arg1, arg2 := rewriter.RewriteArgsForCall(0)
				Expect(arg1).To(Equal("/var/vcap/jobs/nats-tls/config/bpm.v1.yml"))
				Expect(arg2).To(Equal("/var/vcap/jobs/nats-tls/config/bpm.yml"))
			})
		})

		Context("There are only v2 machines in the cluster", func() {
			BeforeEach(func() {
				natsConn1.ConnectedServerVersionReturns("2.2.2")
				natsConn2.ConnectedServerVersionReturns("2.2.2")
				natsConn3.ConnectedServerVersionReturns("2.2.2")
			})

			It("does replace the bpm config", func() {
				err := premigrator.PrepareForMigration()
				Expect(err).NotTo(HaveOccurred())

				Expect(natsConn1.ConnectedServerVersionCallCount()).To(Equal(1))
				Expect(natsConn2.ConnectedServerVersionCallCount()).To(Equal(1))
				Expect(natsConn3.ConnectedServerVersionCallCount()).To(Equal(1))

				Expect(rewriter.RewriteCallCount()).To(Equal(0))
			})
		})
		Context("Unexpected semantnic version format", func() {
			BeforeEach(func() {
				natsConn1.ConnectedServerVersionReturns("1.0")
			})
			It("does not replace the bpm config", func() {

				err := premigrator.PrepareForMigration()
				Expect(err).To(HaveOccurred())

				Expect(natsConn1.ConnectedServerVersionCallCount()).To(Equal(1))
				Expect(natsConn2.ConnectedServerVersionCallCount()).To(Equal(0))
				Expect(natsConn3.ConnectedServerVersionCallCount()).To(Equal(0))

				Expect(rewriter.RewriteCallCount()).To(Equal(0))
			})
		})

		Context("Invalid version response", func() {
			BeforeEach(func() {
				natsConn1.ConnectedServerVersionReturns("notanumber")
			})
			It("does not replace the bpm config", func() {

				err := premigrator.PrepareForMigration()
				Expect(err).To(HaveOccurred())

				Expect(natsConn1.ConnectedServerVersionCallCount()).To(Equal(1))
				Expect(natsConn2.ConnectedServerVersionCallCount()).To(Equal(0))
				Expect(natsConn3.ConnectedServerVersionCallCount()).To(Equal(0))

				Expect(rewriter.RewriteCallCount()).To(Equal(0))
			})
		})
	})

})
