package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricLabelNode     = "node"
	MetricLabelType     = "type"
	MetricLabelVersion  = "version"
	MetricLabelView     = "view"
	MetricLabelRcode    = "rcode"
	MetricLabelSubnetId = "subnet_id"

	MetricNameDNSQPS                 = "lx_dns_qps"
	MetricNameDNSQueriesTotal        = "lx_dns_queries_total"
	MetricNameDNSQueryTypeRatios     = "lx_dns_query_type_ratios"
	MetricNameDNSCacheHits           = "lx_dns_cache_hits"
	MetricNameDNSCacheHitsRatioTotal = "lx_dns_cache_hits_ratio_total"
	MetricNameDNSCacheHitsRatio      = "lx_dns_cache_hits_ratio"
	MetricNameDNSResolvedRatios      = "lx_dns_resolved_ratios"

	MetricNameDHCPLPS          = "lx_dhcp_lps"
	MetricNameDHCPPacketsStats = "lx_dhcp_packets_stats"
	MetricNameDHCPLeasesTotal  = "lx_dhcp_leases_total"
	MetricNameDHCPUsages       = "lx_dhcp_usages"
)

var (
	DNSQPS = prometheus.NewDesc(MetricNameDNSQPS, "dns qps per node",
		[]string{MetricLabelNode}, nil)
	DNSQueriesTotal = prometheus.NewDesc(MetricNameDNSQueriesTotal, "dns queries per node",
		[]string{MetricLabelNode}, nil)
	DNSQueryTypeRatios = prometheus.NewDesc(MetricNameDNSQueryTypeRatios, "dns qtypes ratio per node,type",
		[]string{MetricLabelNode, MetricLabelType}, nil)
	DNSCacheHits = prometheus.NewDesc(MetricNameDNSCacheHits, "dns cache hits per node,view",
		[]string{MetricLabelNode, MetricLabelView}, nil)
	DNSCacheHitsRatioTotal = prometheus.NewDesc(MetricNameDNSCacheHitsRatioTotal, "dns cache hits ratio per node",
		[]string{MetricLabelNode}, nil)
	DNSCacheHitsRatio = prometheus.NewDesc(MetricNameDNSCacheHitsRatio, "dns cache hits ratio per node, view",
		[]string{MetricLabelNode, MetricLabelView}, nil)
	DNSResolvedRatios = prometheus.NewDesc(MetricNameDNSResolvedRatios, "dns resolve ratio per node,rcode",
		[]string{MetricLabelNode, MetricLabelRcode}, nil)

	DHCPLPS = prometheus.NewDesc(MetricNameDHCPLPS, "dhcp lps per node",
		[]string{MetricLabelNode}, nil)
	DHCPPacketsStats = prometheus.NewDesc(MetricNameDHCPPacketsStats, "dhcp packets stats per node,type",
		[]string{MetricLabelNode, MetricLabelVersion, MetricLabelType}, nil)
	DHCPLeasesTotal = prometheus.NewDesc(MetricNameDHCPLeasesTotal, "dhcp leases statistic per node",
		[]string{MetricLabelNode}, nil)
	DHCPUsages = prometheus.NewDesc(MetricNameDHCPUsages, "dhcp usages statistic per node,subnet",
		[]string{MetricLabelNode, MetricLabelSubnetId}, nil)
)

var DNSPrometheusDescs = []*prometheus.Desc{DNSQPS, DNSQueriesTotal, DNSQueryTypeRatios,
	DNSCacheHits, DNSCacheHitsRatioTotal, DNSCacheHitsRatio, DNSResolvedRatios}
var DHCPPrometheusDescs = []*prometheus.Desc{DHCPLPS, DHCPPacketsStats, DHCPLeasesTotal, DHCPUsages}
