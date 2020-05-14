package metric

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-agent/pkg/boltdb"
)

const (
	TableQuery      = "queries"
	TableRecurQuery = "recurqueries"
	TableCacheHit   = "cachehit"
	TableNOERROR    = "NOERROR"
	TableSERVFAIL   = "SERVFAIL"
	TableNXDOMAIN   = "NXDOMAIN"
	TableREFUSED    = "REFUSED"
)

var (
	DNSLabels  = []string{"agent", "dns"}
	DHCPLabels = []string{"agent", "dhcp"}
)

type Collector struct {
	dhcp *DHCPCollector
	dns  *DNSCollector
}

func newCollector(namespace string, db *boltdb.BoltHandler, dhcpAddr string) *Collector {
	return &Collector{
		dhcp: newDHCPCollector(dhcpAddr),
		dns:  newDNSCollector(db),
	}
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	for _, desc := range PrometheusDescs {
		ch <- desc
	}
}

func (c *Collector) Collect(ch chan<- prometheus.Metric) {
	if qps, err := c.dns.GetQPS(TableQuery); err != nil {
		log.Warnf("collect dns qps failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DNSQPS, prometheus.GaugeValue, qps, DNSLabels)
	}

	if queries, err := c.dns.GetQueries(TableQuery); err != nil {
		log.Warnf("collect dns queries failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DNSQueries, prometheus.CounterValue, queries, DNSLabels)
	}

	if recurQueries, err := c.dns.GetQueries(TableRecurQuery); err != nil {
		log.Warnf("collect dns recursive queries failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DNSRecrusiveQueries, prometheus.CounterValue, recurQueries, DNSLabels)
	}

	if hits, err := c.dns.GetQueries(TableCacheHit); err != nil {
		log.Warnf("collect dns cache hits failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DNSCacheHits, prometheus.CounterValue, hits, DNSLabels)
	}

	if noerrors, err := c.dns.GetQueries(TableNOERROR); err != nil {
		log.Warnf("collect dns retcode noerror failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DNSRetCodeNOERROR, prometheus.CounterValue, noerrors, DNSLabels)
	}

	if servfails, err := c.dns.GetQueries(TableSERVFAIL); err != nil {
		log.Warnf("collect dns retcode servfail failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DNSRetCodeSERVFAIL, prometheus.CounterValue, servfails, DNSLabels)
	}

	if nxdomains, err := c.dns.GetQueries(TableNXDOMAIN); err != nil {
		log.Warnf("collect dns retcode nxdomain failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DNSRetCodeNXDOMAIN, prometheus.CounterValue, nxdomains, DNSLabels)
	}

	if refuseds, err := c.dns.GetQueries(TableREFUSED); err != nil {
		log.Warnf("collect dns retcode refused failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DNSRetCodeREFUSED, prometheus.CounterValue, refuseds, DNSLabels)
	}

	if pkts, err := c.dhcp.GetDhcpPacketStatistics(); err != nil {
		log.Warnf("collect dhcp packet statistic failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DHCPPacketStatistic, prometheus.GaugeValue, pkts, DHCPLabels)
	}

	if leases, err := c.dhcp.GetDhcpLeasesStatistics(); err != nil {
		log.Warnf("collect dhcp leases statistic failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DHCPLeaseStatistic, prometheus.GaugeValue, leases, DHCPLabels)
	}

	if usage, err := c.dhcp.GetDhcpUsageStatistics(); err != nil {
		log.Warnf("collect dhcp usage statistic failed: %s", err.Error())
	} else {
		ch <- prometheus.MustNewConstMetric(DHCPUsageStatistic, prometheus.GaugeValue, usage, DHCPLabels)
	}
}
