package metric

import (
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-agent/config"
)

type MetricHandler struct {
	metricPort uint32
	exporter   *Exporter
}

func New(conf *config.AgentConfig) (*MetricHandler, error) {
	exporter, err := NewExporter(conf)
	if err != nil {
		return nil, err
	}

	return &MetricHandler{metricPort: conf.Metric.Port, exporter: exporter}, nil
}

func (h *MetricHandler) Run() {
	prometheus.MustRegister(h.exporter)
	http.Handle("/metrics", promhttp.Handler())
	if err := http.ListenAndServe(":"+strconv.Itoa(int(h.metricPort)), nil); err != nil {
		log.Fatalf(err.Error())
	}
}
