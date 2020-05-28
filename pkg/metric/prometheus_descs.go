package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricNameDNSQPS             = "lx_dns_qps"
	MetricNameDNSQueriesTotal    = "lx_dns_queries_total"
	MetricNameDNSQueryTypeRatios = "lx_dns_query_type_ratios"
	MetricNameDNSCacheHits       = "lx_dns_cache_hits"
	MetricNameDNSResolvedRatios  = "lx_dns_resolved_ratios"

	MetricNameDHCPLPS          = "lx_dhcp_lps"
	MetricNameDHCPPacketsStats = "lx_dhcp_packets_stats"
	MetricNameDHCPLeasesTotal  = "lx_dhcp_leases_total"
	MetricNameDUsages          = "lx_dhcp_usages"
)

var (
	DNSQPS             = prometheus.NewDesc("lx_dns_qps", "dns qps per node", []string{"node"}, nil)
	DNSQueriesTotal    = prometheus.NewDesc("lx_dns_queries_total", "dns queries per node", []string{"node"}, nil)
	DNSQueryTypeRatios = prometheus.NewDesc("lx_dns_qtype_ratios", "dns qtypes ratio per node,type", []string{"node", "type"}, nil)
	DNSCacheHits       = prometheus.NewDesc("lx_dns_cache_hits", "dns cache hits per node,view", []string{"node", "view"}, nil)
	DNSResolvedRatios  = prometheus.NewDesc("lx_dns_resolved_ratios", "dns resolve ratio per node,rcode", []string{"node", "rcode"}, nil)

	DHCPLPS          = prometheus.NewDesc("lx_dhcp_lps", "dhcp lps per node", []string{"node"}, nil)
	DHCPPacketsStats = prometheus.NewDesc("lx_dhcp_packets_stats", "dhcp packets stats per node,type", []string{"node", "type"}, nil)
	DHCPLeasesTotal  = prometheus.NewDesc("lx_dhcp_leases_total", "dhcp leases statistic per node", []string{"node"}, nil)
	DHCPUsages       = prometheus.NewDesc("lx_dhcp_usages", "dhcp usages statistic per node,subnet", []string{"node", "subnet_id"}, nil)
)

var DNSPrometheusDescs = []*prometheus.Desc{DNSQPS, DNSQueriesTotal, DNSQueryTypeRatios, DNSCacheHits, DNSResolvedRatios}
var DHCPPrometheusDescs = []*prometheus.Desc{DHCPLPS, DHCPPacketsStats, DHCPLeasesTotal, DHCPUsages}
