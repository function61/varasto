package stoserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/felixge/httpsnoop"
	"github.com/function61/gokit/promconstmetrics"
	"github.com/function61/varasto/pkg/blobstore"
	"github.com/function61/varasto/pkg/stoserver/stodb"
	"github.com/function61/varasto/pkg/stoserver/stoservertypes"
	"github.com/function61/varasto/pkg/stotypes"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.etcd.io/bbolt"
)

type metricsController struct {
	registry *prometheus.Registry
	volumes  map[int]*volumeMetrics

	httpRequests *prometheus.CounterVec

	scheduledJobRuntime *promconstmetrics.Ref

	constMetricsCollector *promconstmetrics.Collector
}

// metrics for a single volume
type volumeMetrics struct {
	// const metrics b/c these are difficult / non-performant to populate in realtime, so
	// there are refreshed at interval, but readings can be much older (like SMART temperature
	// reading) and thus they have a specific "value at" timestamp
	blobs               *promconstmetrics.Ref
	spaceUsed           *promconstmetrics.Ref
	spaceFree           *promconstmetrics.Ref
	replicationProgress *promconstmetrics.Ref
	temperature         *promconstmetrics.Ref

	// using (totalRequests, errors) instead of (successes, errors) b/c:
	//   https://promcon.io/2017-munich/slides/best-practices-and-beastly-pitfalls.pdf
	readRequests prometheus.Counter
	readBytes    prometheus.Counter
	readErrors   prometheus.Counter

	writeRequests prometheus.Counter
	writtenBytes  prometheus.Counter
	writeErrors   prometheus.Counter
}

func newMetricsController() *metricsController {
	reg := prometheus.NewRegistry()

	constMetricsCollector := promconstmetrics.NewCollector()

	m := &metricsController{
		registry: reg,
		volumes:  map[int]*volumeMetrics{},
		httpRequests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "sto_http_requests_total",
			Help: "HTTP server's handled requests",
		}, []string{"code", "method"}),
		scheduledJobRuntime:   constMetricsCollector.Register("sto_scheduledjob_runtime_seconds", "Scheduled job's runtime (seconds)", prometheus.Labels{}, "job"),
		constMetricsCollector: constMetricsCollector,
	}

	reg.MustRegister(m.httpRequests)
	reg.MustRegister(m.constMetricsCollector)

	return m
}

// builds a cancellable metrics collection task that can be given to taskrunner
func (m *metricsController) Task(conf *ServerConfig, db *bbolt.DB) func(context.Context) error {
	return func(ctx context.Context) error {
		metricsCollectionInterval := time.NewTicker(5 * time.Second)

		for {
			select {
			case <-ctx.Done():
				return nil
			case <-metricsCollectionInterval.C:
				if err := m.collectMetrics(conf, db); err != nil {
					return err
				}
			}
		}
	}
}

func (m *metricsController) MetricsHTTPHandler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{})
}

// instruments a HTTP handler
func (m *metricsController) WrapHTTPServer(actual http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		stats := httpsnoop.CaptureMetrics(actual, w, r)

		m.httpRequests.With(prometheus.Labels{
			"code":   strconv.Itoa(stats.Code),
			"method": r.Method,
		}).Inc()
	})
}

// decorates a blobstore driver with a proxy driver that doesn't change any behaviour, but
// records metrics for the operations
func (m *metricsController) WrapDriver(
	origin blobstore.Driver,
	volID int,
	volUUID string,
	volLabel string,
) blobstore.Driver {
	volMetrics := m.createVolumeMetrics(volUUID, volLabel)

	m.volumes[volID] = volMetrics

	return &proxyDriver{origin, volMetrics}
}

func (m *metricsController) createVolumeMetrics(volUUID string, volLabel string) *volumeMetrics {
	// shorthand for new'ing and registering
	counter := func(opts prometheus.CounterOpts) prometheus.Counter {
		c := prometheus.NewCounter(opts)
		m.registry.MustRegister(c)
		return c
	}

	volMetricLabels := prometheus.Labels{
		"uuid":  volUUID,
		"label": volLabel,
	}

	constMetrics := m.constMetricsCollector // shorthand

	vm := &volumeMetrics{
		blobs:               constMetrics.Register("sto_vol_blobs", "Blobs in a given volume", volMetricLabels),
		spaceUsed:           constMetrics.Register("sto_vol_space_used_bytes", "Actual used space (after deduplication & compression)", volMetricLabels),
		spaceFree:           constMetrics.Register("sto_vol_space_free_bytes", "Free space (quota - used)", volMetricLabels),
		replicationProgress: constMetrics.Register("sto_vol_replication_progress", "Volume replication controller's progress %", volMetricLabels),
		temperature:         constMetrics.Register("sto_vol_temperature_celsius", "Disk temperature", volMetricLabels),

		readRequests: counter(prometheus.CounterOpts{
			Name:        "sto_vol_read_requests_total",
			Help:        "Volume read operations (incl. errors)",
			ConstLabels: volMetricLabels,
		}),
		readBytes: counter(prometheus.CounterOpts{
			Name:        "sto_vol_read_bytes_total",
			Help:        "Volume read bytes",
			ConstLabels: volMetricLabels,
		}),
		readErrors: counter(prometheus.CounterOpts{
			Name:        "sto_vol_read_errors_total",
			Help:        "Volume failed read operations",
			ConstLabels: volMetricLabels,
		}),
		writeRequests: counter(prometheus.CounterOpts{
			Name:        "sto_vol_write_requests_total",
			Help:        "Volume write operations (incl. errors)",
			ConstLabels: volMetricLabels,
		}),
		writtenBytes: counter(prometheus.CounterOpts{
			Name:        "sto_vol_write_bytes_total",
			Help:        "Volume written bytes",
			ConstLabels: volMetricLabels,
		}),
		writeErrors: counter(prometheus.CounterOpts{
			Name:        "sto_vol_write_errors_total",
			Help:        "Volume failed write operations",
			ConstLabels: volMetricLabels,
		}),
	}

	return vm
}

func (m *metricsController) collectMetrics(conf *ServerConfig, db *bbolt.DB) error {
	tx, rollback, err := readTx(db)
	if err != nil {
		return err
	}
	defer rollback()

	now := time.Now()

	constMetrics := m.constMetricsCollector // shorthand

	for volID, volMetrics := range m.volumes {
		vol, err := stodb.Read(tx).Volume(volID)
		if err != nil {
			return err
		}

		constMetrics.Observe(volMetrics.blobs, float64(vol.BlobCount), now)
		constMetrics.Observe(volMetrics.spaceUsed, float64(vol.BlobSizeTotal), now)
		constMetrics.Observe(volMetrics.spaceFree, float64(vol.Quota-vol.BlobSizeTotal), now)

		if ctrl, has := conf.ReplicationControllers[vol.ID]; has {
			constMetrics.Observe(volMetrics.replicationProgress, float64(ctrl.Progress())/100, now)
		}

		// TODO: filter out old SMART reports
		if vol.SmartReport != "" {
			report := &stoservertypes.SmartReport{}
			if err := json.Unmarshal([]byte(vol.SmartReport), report); err != nil {
				return err
			}

			if report.Temperature != nil {
				constMetrics.Observe(volMetrics.temperature, float64(*report.Temperature), report.Time)
			}
		}
	}

	jobs := []stotypes.ScheduledJob{}
	if err := stodb.ScheduledJobRepository.Each(stodb.Appender(&jobs), tx); err != nil {
		return err
	}

	for _, job := range jobs {
		if job.LastRun != nil {
			lastrun := job.LastRun // shorthand

			constMetrics.Observe(m.scheduledJobRuntime, lastrun.Runtime().Seconds(), lastrun.Finished, job.Description)
		}
	}

	return nil
}

// wraps origin store in a new driver that doesn't change the behaviour, but is used to
// record metrics
type proxyDriver struct {
	blobstore.Driver
	volMetrics *volumeMetrics
}

func (p *proxyDriver) RawStore(ctx context.Context, ref stotypes.BlobRef, content io.Reader) error {
	p.volMetrics.writeRequests.Inc()

	err := p.Driver.RawStore(ctx, ref, newReadCounter(content, func(bytesRead int64, errRead error) {
		if errRead == nil {
			// not all succesfully read bytes were necessarily written to the volume in error
			// cases, but this is the least invasive way grab this info where and it's accurate
			// on successes
			p.volMetrics.writtenBytes.Add(float64(bytesRead))
		}
	}))

	if err != nil {
		p.volMetrics.writeErrors.Inc()
	}

	return err
}

func (p *proxyDriver) RawFetch(ctx context.Context, ref stotypes.BlobRef) (io.ReadCloser, error) {
	// will be called (once) much later than we return from this func
	readFinished := func(bytesRead int64, err error) {
		p.volMetrics.readBytes.Add(float64(bytesRead))

		if err != nil {
			p.volMetrics.readErrors.Inc()
		}
	}

	p.volMetrics.readRequests.Inc()

	content, err := p.Driver.RawFetch(ctx, ref)

	contentMaybeWrapped := content

	if err != nil {
		readFinished(0, err)
	} else {
		contentMaybeWrapped = newReadCounter(content, readFinished)
	}

	return contentMaybeWrapped, err
}

type readCounter struct {
	bytesRead int64 // has to be first b/c sync/atomic alignment rules
	io.ReadCloser
	stats     func(int64, error)
	statsOnce sync.Once
}

// "stats" will only be called once, and when:
// a) all reads succeeded to io.EOF OR
// b) first read failed
func newReadCounter(content io.Reader, stats func(int64, error)) io.ReadCloser {
	rc, ok := content.(io.ReadCloser)
	if !ok {
		rc = io.NopCloser(content)
	}

	return &readCounter{
		ReadCloser: rc,
		stats:      stats,
	}
}

var _ io.ReadCloser = (*readCounter)(nil)

func (r *readCounter) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)

	// probably won't be used concurrently, but let's be safe anyway
	atomic.AddInt64(&r.bytesRead, int64(n))

	// stream read will always end up in either..
	if err != nil {
		if err == io.EOF { // a) pseudo-error
			_ = r.emitStats(nil)
		} else { // b) actual error
			_ = r.emitStats(err)
		}
	}

	return n, err
}

func (r *readCounter) emitStats(err error) error {
	r.statsOnce.Do(func() {
		r.stats(r.bytesRead, err)
	})

	return err
}
