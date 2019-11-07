package natsbench

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/bench"
)

const (
	// DefaultPublisherCount is the default number of publishers to benchmark.
	DefaultPublisherCount = 10

	// DefaultSubscriberCount is the default number of subscribers to
	// benchmark.
	DefaultSubscriberCount = 2

	// DefaultMessageSize is the default size (in bytes) of a message to
	// benchmark.
	DefaultMessageSize = 128

	// DefaultMessageCount is the default number of messages to send (across
	// all publishers).
	DefaultMessageCount = 100000
)

// Benchmark represents a single benchmark run.
type Benchmark struct {
	Name string

	bm   *bench.Benchmark
	urls string
	opts *options
}

// NewBenchmark creates a new benchmark.
func NewBenchmark(name string, bos ...BenchmarkOption) *Benchmark {
	opts := defaultOpts()
	for _, opt := range bos {
		opt(opts)
	}

	return &Benchmark{
		Name: name,
		opts: opts,
		urls: opts.urls(),
		bm:   bench.NewBenchmark(opts.benchName(name), opts.subCount, opts.pubCount),
	}
}

// Run runs the benchmark. It should only be called once per benchmark.
func (b *Benchmark) Run() {
	var allDone sync.WaitGroup
	allDone.Add(b.opts.subCount + b.opts.pubCount)
	b.startSubs(&allDone)
	b.startPubs(&allDone)
	allDone.Wait()
	b.bm.Close()

	fmt.Println(b.bm.Report())
}

func (b *Benchmark) startSubs(allDone *sync.WaitGroup) {
	var subsDone sync.WaitGroup
	subsDone.Add(b.opts.subCount)
	for i := 0; i < b.opts.subCount; i++ {
		go b.runSubscriber(&subsDone, allDone)
	}

	subsDone.Wait()
}

func (b *Benchmark) startPubs(allDone *sync.WaitGroup) {
	var pubsDone sync.WaitGroup
	pubsDone.Add(b.opts.pubCount)
	pubCounts := bench.MsgsPerClient(b.opts.msgCount, b.opts.pubCount)
	for i := 0; i < b.opts.pubCount; i++ {
		go b.runPublisher(&pubsDone, allDone, pubCounts[i])
	}

	pubsDone.Wait()
}

func (b *Benchmark) runPublisher(pubsDone, allDone *sync.WaitGroup, count int) {
	nc := b.newNatsConn()
	pubsDone.Done()
	subj := "test"
	msg := make([]byte, b.opts.msgSize)

	start := time.Now()

	for i := 0; i < count; i++ {
		nc.Publish(subj, msg)
	}
	nc.Flush()
	b.bm.AddPubSample(bench.NewSample(b.opts.msgCount, b.opts.msgSize, start, time.Now(), nc))

	nc.Close()
	allDone.Done()
}

func (b *Benchmark) runSubscriber(subsDone, allDone *sync.WaitGroup) {
	nc := b.newNatsConn()
	subj := "test"
	received := 0
	ch := make(chan time.Time, 2)
	sub, _ := nc.Subscribe(subj, func(msg *nats.Msg) {
		received++
		if received == 1 {
			ch <- time.Now()
		}
		if received >= b.opts.msgCount {
			ch <- time.Now()
			return
		}
	})
	sub.SetPendingLimits(-1, -1)
	nc.Flush()
	subsDone.Done()

	start := <-ch
	end := <-ch
	b.bm.AddSubSample(bench.NewSample(b.opts.msgCount, b.opts.msgSize, start, end, nc))

	nc.Close()
	allDone.Done()
}

func (b *Benchmark) newNatsConn() *nats.Conn {
	nc, err := nats.Connect(b.urls, b.opts.natsOpts...)
	if err != nil {
		log.Fatalf("connection error: %v", err)
	}

	return nc
}
