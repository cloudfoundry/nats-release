package integration

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"time"

	"code.cloudfoundry.org/inigo/helpers/certauthority"
	"code.cloudfoundry.org/lager/lagerflags"
	"code.cloudfoundry.org/nats-v2-migrate/config"
	"code.cloudfoundry.org/nats-v2-migrate/integration/helpers"
	"code.cloudfoundry.org/tlsconfig"

	"github.com/nats-io/nats.go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/onsi/gomega/ghttp"
)

var _ = FDescribe("Migrate", func() {
	var (
		cfg         config.Config
		configFile  *os.File
		migrateBin  string
		migrateSess *gexec.Session
		// migrateServerURL string
	)

	BeforeEach(func() {
		var err error
		migrateBin, err = gexec.Build("code.cloudfoundry.org/nats-v2-migrate/cmd/migrate")
		Expect(err).NotTo(HaveOccurred())
		cfg = config.Config{
			LagerConfig: lagerflags.DefaultLagerConfig(),
			// NATSMigratePort: 10000 + config.GinkgoConfig.ParallelNode,
		}
		// migrateServerURL = fmt.Sprintf("http://127.0.0.1:%d", cfg.NATSMigratePort)
	})

	JustBeforeEach(func() {
		var err error
		configFile, err = ioutil.TempFile("", "migrate_config")
		Expect(err).NotTo(HaveOccurred())

		cfgJSON, err := json.Marshal(cfg)
		Expect(err).NotTo(HaveOccurred())

		_, err = configFile.Write(cfgJSON)
		Expect(err).NotTo(HaveOccurred())

		migrateCmd := exec.Command(migrateBin, "-config-file", configFile.Name())
		migrateSess, err = gexec.Start(migrateCmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())
		// migrateServerAddress := fmt.Sprintf("127.0.0.1:%d", cfg.NATSMigratePort)
		// Eventually(func() error {
		// 	_, err := net.Dial("tcp", migrateServerAddress)
		// 	return err
		// }).Should(Succeed())
	})

	AfterEach(func() {
		migrateSess.Kill()
		os.Remove(configFile.Name())
	})

	Context("when there are no other nats machines", func() {
		// premigrate runs it as v2
		BeforeEach(func() {
			cfg.NATSMigrateServers = []string{}
		})

		It("exits succesfully", func() {
			Eventually(migrateSess).Should(gexec.Exit(0))
			// resp, err := http.Get(migrateServerURL)
			// Expect(err).NotTo(HaveOccurred())
			// Expect(resp.StatusCode).To(Equal(http.StatusOK))

		})
	})

	Context("when there are other nats machines", func() {
		var (
			certDepoDir                                                string
			natsMigrateServer1, natsMigrateServer2, natsMigrateServer3 *ghttp.Server
			natsPort                                                   uint16
			natsRunner                                                 *helpers.NATSRunner
		)

		BeforeEach(func() {
			var err error
			certDepoDir, err = ioutil.TempDir("", "")
			Expect(err).NotTo(HaveOccurred())

			ca, err := certauthority.NewCertAuthority(certDepoDir, "nats-v2-migrate-ca")
			Expect(err).NotTo(HaveOccurred())

			serverKeyFile, serverCertFile, err := ca.GenerateSelfSignedCertAndKey("server", []string{}, false)
			Expect(err).NotTo(HaveOccurred())
			_, serverCAFile := ca.CAAndKey()
			cfg.NATSMigrateServerCAFile = serverCAFile
			cfg.NATSMigrateServerClientCertFile = serverCertFile
			cfg.NATSMigrateServerClientKeyFile = serverKeyFile

			natsMigrateServer1 = NewNATSMigrateServer(serverCAFile, serverCertFile, serverKeyFile, false)
			natsMigrateServer1.HTTPTestServer.StartTLS()

			natsMigrateServer2 = NewNATSMigrateServer(serverCAFile, serverCertFile, serverKeyFile, true)
			natsMigrateServer2.HTTPTestServer.StartTLS()

			natsMigrateServer3 = NewNATSMigrateServer(serverCAFile, serverCertFile, serverKeyFile, false)
			natsMigrateServer3.HTTPTestServer.StartTLS()

			cfg.NATSMigrateServers = []string{natsMigrateServer1.URL(), natsMigrateServer2.URL(), natsMigrateServer3.URL()}
			natsPort = 4224
			natsRunner = helpers.NewNATSRunner(int(natsPort))
			cfg.LocalNATSAddr = natsRunner.Addr()
		})

		AfterEach(func() {
			Expect(os.RemoveAll(certDepoDir)).To(Succeed())

			if natsRunner != nil {
				natsRunner.Stop()
			}

			natsMigrateServer1.Close()
			natsMigrateServer2.Close()
			natsMigrateServer3.Close()
		})

		Context("when the local NATS server is running the v2 version", func() {
			// premigrate decided to run it as v2, either it is a fresh deploy or everything is v2 already
			BeforeEach(func() {
				natsRunner.Start()
				conn, err := nats.Connect(natsRunner.URL())
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("2.8.2"))
			})

			It("exits succesfully", func() {
				Eventually(migrateSess).Should(gexec.Exit(0))
			})
		})

		Context("when it fails to connect to local NATS server", func() {
			It("fails with error after the timeout", func() {
				Eventually(migrateSess).WithTimeout(12 * time.Second).Should(gexec.Exit(1))
			})

			It("retries with the timeout", func() {
				Consistently(migrateSess).WithTimeout(2 * time.Second).ShouldNot(gexec.Exit())
				natsRunner.Start()
				Eventually(migrateSess).Should(gexec.Exit(0))
			})
		})

		Context("when the local NATS server is running the v1 version", func() {
			BeforeEach(func() {
				natsRunner.StartV1()
				conn, err := nats.Connect(natsRunner.URL())
				Expect(err).ToNot(HaveOccurred())
				version := conn.ConnectedServerVersion()
				Expect(version).To(Equal("1.4.1"))
			})
			// premigrate decided to run it as v1, means it found other machines that are v1

			It("validates if other nats machines have migrate server running", func() {
				// did they get new version yet?

				Eventually(natsMigrateServer1.ReceivedRequests).Should(HaveLen(2))
				Expect(natsMigrateServer1.ReceivedRequests()[0].URL.Path).To(Equal("/info"))
				Eventually(natsMigrateServer2.ReceivedRequests).Should(HaveLen(2))
				Expect(natsMigrateServer2.ReceivedRequests()[0].URL.Path).To(Equal("/info"))
				Eventually(natsMigrateServer3.ReceivedRequests).Should(HaveLen(2))
				Expect(natsMigrateServer3.ReceivedRequests()[0].URL.Path).To(Equal("/info"))
			})

			Context("when at least one migrate server does not respond", func() {
				// they have not finished an update yet
				// and they don't have a new version of release with v2 binary installed yet
				// or the migrate process is down (in that case the deployment will fail on that machine)
				BeforeEach(func() {
					natsMigrateServer3.Close()
				})

				It("continues running as v1 and does not migrate", func() {
					Eventually(natsMigrateServer1.ReceivedRequests).Should(HaveLen(1))
					Expect(natsMigrateServer1.ReceivedRequests()[0].URL.Path).To(Equal("/info"))
					Eventually(natsMigrateServer2.ReceivedRequests).Should(HaveLen(1))
					Expect(natsMigrateServer2.ReceivedRequests()[0].URL.Path).To(Equal("/info"))

					Eventually(natsMigrateServer3.ReceivedRequests).Should(HaveLen(0))

					Eventually(migrateSess).Should(gexec.Exit(0))
				})
			})

			Context("when all migrate servers respond", func() {
				// they have been updated, they have v2 binary ready
				// this is most likely will happen on a machine(s) that will be updated last

				Context("when there is a migrate server on bootstrap VM", func() {
					Context("when migration to v2 succeeds on bootstrap VM", func() {
						It("tells other migrate servers to migrate to v2", func() {
							Eventually(natsMigrateServer1.ReceivedRequests).Should(HaveLen(2))
							Expect(natsMigrateServer1.ReceivedRequests()[0].URL.Path).To(Equal("/info"))
							Expect(natsMigrateServer1.ReceivedRequests()[1].URL.Path).To(Equal("/migrate"))
							Eventually(natsMigrateServer3.ReceivedRequests).Should(HaveLen(2))
							Expect(natsMigrateServer3.ReceivedRequests()[0].URL.Path).To(Equal("/info"))
							Expect(natsMigrateServer3.ReceivedRequests()[1].URL.Path).To(Equal("/migrate"))
						})

						Context("when there is an error migrating one of the servers", func() {
							BeforeEach(func() {
								natsMigrateServer1.RouteToHandler("POST", "/migrate", func(w http.ResponseWriter, r *http.Request) {
									natsMigrateServer1.CloseClientConnections()
								})
							})

							It("still migrates the rest of the servers", func() {
								Eventually(natsMigrateServer1.ReceivedRequests).Should(HaveLen(2))
								Expect(natsMigrateServer1.ReceivedRequests()[1].URL.Path).To(Equal("/migrate"))
								Eventually(natsMigrateServer3.ReceivedRequests).Should(HaveLen(2))
								Expect(natsMigrateServer3.ReceivedRequests()[1].URL.Path).To(Equal("/migrate"))
							})

							It("exits with the error", func() {
								Eventually(migrateSess).Should(gexec.Exit(1))
							})
						})

						Context("when one of the migration servers responds with non 200 status code", func() {
							BeforeEach(func() {
								natsMigrateServer1.RouteToHandler("POST", "/migrate", ghttp.CombineHandlers(
									ghttp.VerifyRequest("POST", "/migrate"),
									ghttp.RespondWith(http.StatusBadRequest, ""),
								))
							})
							It("still migrates the rest of the servers", func() {
								Eventually(natsMigrateServer1.ReceivedRequests).Should(HaveLen(2))
								Expect(natsMigrateServer1.ReceivedRequests()[1].URL.Path).To(Equal("/migrate"))
								Eventually(natsMigrateServer3.ReceivedRequests).Should(HaveLen(2))
								Expect(natsMigrateServer3.ReceivedRequests()[1].URL.Path).To(Equal("/migrate"))

							})
							It("exits with the error", func() {
								Eventually(migrateSess).Should(gexec.Exit(1))
							})
						})
					})

					Context("when migration to v2 fails on bootstrap VM", func() {
						Context("when request to the migration endpoint errors", func() {
							BeforeEach(func() {
								natsMigrateServer2.RouteToHandler("POST", "/migrate", func(w http.ResponseWriter, r *http.Request) {
									natsMigrateServer2.CloseClientConnections()
								})
							})

							It("does not tell other migrate servers to migrate to v2", func() {
								Eventually(natsMigrateServer1.ReceivedRequests).Should(HaveLen(1))
								Expect(natsMigrateServer1.ReceivedRequests()[0].URL.Path).To(Equal("/info"))
								Eventually(natsMigrateServer3.ReceivedRequests).Should(HaveLen(1))
								Expect(natsMigrateServer3.ReceivedRequests()[0].URL.Path).To(Equal("/info"))
							})

							It("exits with error", func() {
								Eventually(migrateSess).Should(gexec.Exit(1))
							})
						})

						// it is the responsibility of migrate server itself to rollback to v1 if it fails to start as v2
						Context("when the migration endpoint on bootstrap VM  fails with a non 200 status code", func() {
							BeforeEach(func() {
								natsMigrateServer2.RouteToHandler("POST", "/migrate", ghttp.CombineHandlers(
									ghttp.VerifyRequest("POST", "/migrate"),
									ghttp.RespondWith(http.StatusBadRequest, ""),
								))
							})

							It("does not tell other migrate servers to migrate to v2", func() {
								Eventually(natsMigrateServer1.ReceivedRequests).Should(HaveLen(1))
								Eventually(natsMigrateServer3.ReceivedRequests).Should(HaveLen(1))
							})

							It("exits with error", func() {
								Eventually(migrateSess).Should(gexec.Exit(1))
							})
						})
					})
				})

				Context("when there is no bootstrap migrate server", func() {
					// this should not happen, bosh makes one VM as bootstrap
					BeforeEach(func() {
						natsMigrateServer2.Close()
						cfg.NATSMigrateServers = []string{natsMigrateServer1.URL(), natsMigrateServer3.URL()}
					})

					It("exits with error", func() {
						Eventually(migrateSess).Should(gexec.Exit(1))
					})
				})
			})
		})
	})
})

func NewNATSMigrateServer(serverCAFile, serverCertFile, serverKeyFile string, isBootstrap bool) *ghttp.Server {
	natsMigrateServer := ghttp.NewUnstartedServer()
	var err error
	natsMigrateServer.HTTPTestServer.TLS, err = tlsconfig.Build(
		tlsconfig.WithInternalServiceDefaults(),
		tlsconfig.WithIdentityFromFile(serverCertFile, serverKeyFile),
	).Server(tlsconfig.WithClientAuthenticationFromFile(serverCAFile))
	Expect(err).NotTo(HaveOccurred())

	natsMigrateServer.RouteToHandler("GET", "/info", ghttp.CombineHandlers(
		ghttp.VerifyRequest("GET", "/info"),
		ghttp.RespondWith(http.StatusOK, fmt.Sprintf(`{"bootstrap":%t}`, isBootstrap)),
	))

	natsMigrateServer.RouteToHandler("POST", "/migrate", ghttp.CombineHandlers(
		ghttp.VerifyRequest("POST", "/migrate"),
		ghttp.RespondWith(http.StatusOK, ""),
	))
	return natsMigrateServer
}
