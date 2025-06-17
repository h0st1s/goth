package goth

import (
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
)

var statusNames = map[int32]string{
	statusStopped: "stopped",
	statusRunning: "running",
	statusWaiting: "waiting",
	statusCrashed: "crashed",
}

const (
	namespace = "goth"
	subsystem = "threads"
)

// prometheus metric collector
type StatCollector struct {
	stat *ThreadStat
	// metric descriptions
	descRestarts *prometheus.Desc
	descCrashes  *prometheus.Desc
	descStatus   *prometheus.Desc
}

func NewStatCollector(label string, stat *ThreadStat) *StatCollector {
	labels := prometheus.Labels{"thread": label}
	return &StatCollector{
		stat: stat,
		descRestarts: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "restarts"),
			"The number of the thread restarts",
			nil, labels,
		),
		descCrashes: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "crashes"),
			"The number of the thread crashes",
			nil, labels,
		),
		descStatus: prometheus.NewDesc(
			prometheus.BuildFQName(namespace, subsystem, "status"),
			"Thread status",
			[]string{"status"}, labels,
		),
	}
}

// implements prometheus.Collector interface
func (c *StatCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.descRestarts
	ch <- c.descCrashes
	ch <- c.descStatus
}

// implements prometheus.Collector interface
func (c *StatCollector) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(c.descRestarts, prometheus.CounterValue, float64(atomic.LoadInt32(&c.stat.Restarts)))
	ch <- prometheus.MustNewConstMetric(c.descCrashes, prometheus.CounterValue, float64(atomic.LoadInt32(&c.stat.Crashes)))
	status := atomic.LoadInt32(&c.stat.Status)
	for k, v := range statusNames {
		if k == status {
			ch <- prometheus.MustNewConstMetric(c.descStatus, prometheus.GaugeValue, 1.0, v)
		} else {
			ch <- prometheus.MustNewConstMetric(c.descStatus, prometheus.GaugeValue, 0.0, v)
		}
	}
}
