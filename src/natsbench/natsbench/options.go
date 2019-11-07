package natsbench

import (
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go"
)

// BenchmarkOption can be used to configure the parameters of a benchmark.
type BenchmarkOption func(*options)

type options struct {
	username string
	password string
	hosts    []string
	port     int

	pubCount int
	subCount int
	msgCount int
	msgSize  int

	natsOpts []nats.Option
}

func defaultOpts() *options {
	return &options{
		pubCount: DefaultPublisherCount,
		subCount: DefaultSubscriberCount,
		msgCount: DefaultMessageCount,
		msgSize:  DefaultMessageSize,
		port:     4222,
	}
}

func (o *options) benchName(name string) string {
	return fmt.Sprintf(
		"%s [msgs=%d, size=%d, pubs=%d, subs=%d]\n",
		name,
		o.msgCount,
		o.msgSize,
		o.pubCount,
		o.subCount,
	)
}

func (o *options) urls() string {
	var urls []string

	for _, host := range o.hosts {
		addr := net.JoinHostPort(host, strconv.Itoa(o.port))

		u := &url.URL{
			Scheme: "nats",
			Host:   addr,
		}

		if o.username != "" {
			u.User = url.UserPassword(o.username, o.password)
		}

		urls = append(urls, u.String())
	}

	return strings.Join(urls, ",")
}

// WithPort sets the port which will be used to connect to the NATS servers in
// the cluster.
func WithPort(port int) BenchmarkOption {
	return func(o *options) {
		o.port = port
	}
}

// WithHosts sets the hosts which should be connected to in the NATS cluster.
func WithHosts(hosts ...string) BenchmarkOption {
	return func(o *options) {
		o.hosts = hosts
	}
}

// WithAuth sets the username and password which should be used when connecting
// to the NATS servers.
func WithAuth(username, password string) BenchmarkOption {
	return func(o *options) {
		o.username = username
		o.password = password
	}
}

// WithNATSOptions allows for specifying additional nats.Options which will be
// passed through to the new NATS connections
func WithNATSOptions(opts ...nats.Option) BenchmarkOption {
	return func(o *options) {
		o.natsOpts = opts
	}
}
