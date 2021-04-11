package metric

import (
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/linkingthing/ddi-agent/config"
)

const HttpClientTimeout = 10

type Exporter struct {
	dnsCollector  *DNSCollector
	dhcpCollector *DHCPCollector
}

func NewExporter(conf *config.AgentConfig) (*Exporter, error) {
	httpClient := &http.Client{
		Timeout:   HttpClientTimeout * time.Second,
		Transport: &http.Transport{DisableKeepAlives: true},
	}
	dnsCollector, err := newDNSCollector(conf, httpClient)
	if err != nil {
		return nil, err
	}

	dhcpCollector, err := newDHCPCollector(conf, httpClient)
	if err != nil {
		return nil, err
	}

	return &Exporter{dnsCollector: dnsCollector, dhcpCollector: dhcpCollector}, nil
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.dnsCollector.Describe(ch)
	e.dhcpCollector.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	e.dnsCollector.Collect(ch)
	e.dhcpCollector.Collect(ch)
}
