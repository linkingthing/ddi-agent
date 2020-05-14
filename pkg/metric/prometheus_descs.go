package metric

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	DNSQPS              = prometheus.NewDesc("lx_dns_qps", "the gauge of dns qps", []string{"server", "module"}, nil)
	DNSQueries          = prometheus.NewDesc("lx_dns_queries", "the counter of dns queries", []string{"server", "module"}, nil)
	DNSRecrusiveQueries = prometheus.NewDesc("lx_dns_recursive_queries", "the counter of dns recursive queries", []string{"server", "module"}, nil)
	DNSCacheHits        = prometheus.NewDesc("lx_dns_cache_hits", "the counter of dns cache hits", []string{"server", "module"}, nil)
	DNSRetCodeNOERROR   = prometheus.NewDesc("lx_dns_retcode_noerror", "the counter of dns noerror return code", []string{"server", "module"}, nil)
	DNSRetCodeNXDOMAIN  = prometheus.NewDesc("lx_dns_retcode_nxdomain", "the counter of dns nxdomain return code", []string{"server", "module"}, nil)
	DNSRetCodeSERVFAIL  = prometheus.NewDesc("lx_dns_retcode_servfail", "the counter of dns servfail return code", []string{"server", "module"}, nil)
	DNSRetCodeREFUSED   = prometheus.NewDesc("lx_dns_retcode_refused", "the counter of dns refused return code", []string{"server", "module"}, nil)
	DHCPPacketStatistic = prometheus.NewDesc("lx_dhcp_packet_statistic", "the gauge of dhcp packet statistic", []string{"server", "module"}, nil)
	DHCPLeaseStatistic  = prometheus.NewDesc("lx_dhcp_lease_statistic", "the gauge of dhcp lease statistic", []string{"server", "module"}, nil)
	DHCPUsageStatistic  = prometheus.NewDesc("lx_dhcp_usage_statistic", "the gauge of dhcp usage statistic", []string{"server", "module"}, nil)
)

var PrometheusDescs = []*prometheus.Desc{DNSQPS, DNSQueries, DNSRecrusiveQueries, DNSCacheHits, DNSRetCodeNOERROR, DNSRetCodeNXDOMAIN, DNSRetCodeSERVFAIL, DNSRetCodeREFUSED, DHCPPacketStatistic, DHCPLeaseStatistic, DHCPUsageStatistic}
