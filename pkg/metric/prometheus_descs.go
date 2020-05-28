package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

const (
	MetricNameDNSQPS           = "lx_dns_qps"
	MetricNameDNSQueries       = "lx_dns_queries_total"
	MetricNameDNSQueryTypes    = "lx_dns_query_type_ratios"
	MetricNameDNSCacheHits     = "lx_dns_cache_hits"
	MetricNameDNSResolveRatios = "lx_dns_resolve_ratios"

	MetricNameDHCPLPS     = "lx_dhcp_lps"
	MetricNameDHCPPackets = "lx_dhcp_packets_stats"
	MetricNameDHCPLeases  = "lx_dhcp_leases_total"
	MetricNameDUsages     = "lx_dhcp_usages"
)

var (
	DNSQPS           = prometheus.NewDesc("lx_dns_qps", "dns qps per node", []string{"node"}, nil)
	DNSQueries       = prometheus.NewDesc("lx_dns_queries_total", "dns queries per node", []string{"node"}, nil)
	DNSQueryTypes    = prometheus.NewDesc("lx_dns_qtype_ratios", "dns qtypes ratio per node,type", []string{"node", "type"}, nil)
	DNSCacheHits     = prometheus.NewDesc("lx_dns_cache_hits", "dns cache hits per node,view", []string{"node", "view"}, nil)
	DNSResolveRatios = prometheus.NewDesc("lx_dns_resolve_ratios", "dns resolve ratio per node,rcode", []string{"node", "rcode"}, nil)

	DHCPLPS     = prometheus.NewDesc("lx_dhcp_lps", "dhcp lps per node", []string{"node"}, nil)
	DHCPPackets = prometheus.NewDesc("lx_dhcp_packets_stats", "dhcp packets statistic per node,type", []string{"node", "type"}, nil)
	DHCPLeases  = prometheus.NewDesc("lx_dhcp_leases_total", "dhcp leases statistic per node", []string{"node"}, nil)
	DHCPUsages  = prometheus.NewDesc("lx_dhcp_usages", "dhcp usages statistic per node,subnet", []string{"node", "subnet_id"}, nil)
)

var DNSPrometheusDescs = []*prometheus.Desc{DNSQPS, DNSQueries, DNSQueryTypes, DNSCacheHits, DNSResolveRatios}
var DHCPPrometheusDescs = []*prometheus.Desc{DHCPLPS, DHCPPackets, DHCPLeases, DHCPUsages}
