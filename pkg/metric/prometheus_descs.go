package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DNSQPS        = prometheus.NewDesc("lx_dns_qps", "dns qps per node", []string{"node"}, nil)
	DNSQueries    = prometheus.NewDesc("lx_dns_queries_total", "dns queries per node", []string{"node"}, nil)
	DNSQueryTypes = prometheus.NewDesc("lx_dns_query_types", "dns query types per type of node", []string{"node", "type"}, nil)
	DNSCacheHits  = prometheus.NewDesc("lx_dns_cache_hits", "dns cache hits per view of node", []string{"node", "view"}, nil)
	DNSRCodes     = prometheus.NewDesc("lx_dns_rcodes", "dns return code per node", []string{"node", "rcode"}, nil)

	DHCPLPS     = prometheus.NewDesc("lx_dhcp_lps", "dhcp lps per node", []string{"node"}, nil)
	DHCPPackets = prometheus.NewDesc("lx_dhcp_packets_stats", "dhcp packets statistic per type of node", []string{"node", "type"}, nil)
	DHCPLeases  = prometheus.NewDesc("lx_dhcp_leases_total", "dhcp leases statistic per node", []string{"node"}, nil)
	DHCPUsages  = prometheus.NewDesc("lx_dhcp_usages", "dhcp usages statistic per subnet of node", []string{"node", "subnet_id"}, nil)
)

var DNSPrometheusDescs = []*prometheus.Desc{DNSQPS, DNSQueries, DNSQueryTypes, DNSCacheHits, DNSRCodes}
var DHCPPrometheusDescs = []*prometheus.Desc{DHCPLPS, DHCPPackets, DHCPLeases, DHCPUsages}
