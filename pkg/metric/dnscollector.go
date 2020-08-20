package metric

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-agent/config"
)

const (
	HttpScheme                = "http://"
	StatsServerPath           = "/xml/v3/server"
	ServerCounterTypeOpCode   = "opcode"
	ServerCounterTypeNSStat   = "nsstat"
	ServerCounterTypeQType    = "qtype"
	ViewCounterTypeCacheStats = "cachestats"
	CacheStatsQueryHits       = "QueryHits"
	OpcodeQUERY               = "QUERY"
	QrySuccess                = "QrySuccess"
	QryReferral               = "QryReferral"
	QryNxrrset                = "QryNxrrset"
	QrySERVFAIL               = "QrySERVFAIL"
	QryFORMERR                = "QryFORMERR"
	QryNXDOMAIN               = "QryNXDOMAIN"
	QryDuplicate              = "QryDuplicate"
	QryDropped                = "QryDropped"
	QryFailure                = "QryFailure"
	RcodeSuccess              = "Success"
	RcodeReferral             = "Referral"
	RcodeNxrrset              = "Nxrrset"
	RcodeSERVFAIL             = "SERVFAIL"
	RcodeFORMERR              = "FORMERR"
	RcodeNXDOMAIN             = "NXDOMAIN"
	RcodeDuplicate            = "Duplicate"
	RcodeDropped              = "Dropped"
	RcodeFailure              = "Failure"

	Uint32Max = 4294967295
)

type DNSCollector struct {
	enabled        bool
	nodeIP         string
	url            string
	httpClient     *http.Client
	lastQueryCount uint64
	lastGetTime    time.Time
	qps            uint64
}

func newDNSCollector(conf *config.AgentConfig, cli *http.Client) (*DNSCollector, error) {
	if conf.DNS.Enabled == false {
		return &DNSCollector{enabled: conf.DNS.Enabled}, nil
	}

	u, err := url.Parse(HttpScheme + conf.DNS.StatsAddr + StatsServerPath)
	if err != nil {
		return nil, err
	}

	c := &DNSCollector{
		enabled:    conf.DNS.Enabled,
		nodeIP:     conf.Server.IP,
		url:        u.String(),
		httpClient: cli,
	}
	go c.Run()
	return c, nil
}

func (dns *DNSCollector) Run() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			statistics, err := dns.getStats()
			if err != nil {
				continue
			}

			var qps float64
			for _, cs := range statistics.Server.Counters {
				if cs.Type == ServerCounterTypeOpCode {
					for _, c := range cs.Counters {
						if c.Name == OpcodeQUERY {
							now := time.Now()
							if seconds := now.Sub(dns.lastGetTime).Seconds(); seconds > 0 && dns.lastQueryCount != 0 {
								if c.Counter >= dns.lastQueryCount {
									qps = float64(c.Counter-dns.lastQueryCount) / seconds
								} else {
									qps = float64(c.Counter+Uint32Max-dns.lastQueryCount) / seconds
								}
							}
							dns.lastQueryCount = c.Counter
							dns.lastGetTime = now
							break
						}
					}
					break
				}
			}

			atomic.StoreUint64(&dns.qps, uint64(qps))
		}
	}
}

func (dns *DNSCollector) Describe(ch chan<- *prometheus.Desc) {
	if dns.enabled {
		for _, desc := range DNSPrometheusDescs {
			ch <- desc
		}
	}
}

func (dns *DNSCollector) Collect(ch chan<- prometheus.Metric) {
	if dns.enabled == false {
		return
	}

	statistics, err := dns.getStats()
	if err != nil {
		log.Warnf("get dns statistics with node %s failed: %s", dns.nodeIP, err.Error())
		return
	}

	totalQueries, ok := dns.getQueryTotal(statistics.Server.Counters)
	if ok == false || totalQueries == 0 {
		return
	}

	totalCacheHits := dns.collectCacheHits(ch, statistics.Views)

	ch <- prometheus.MustNewConstMetric(DNSQPS, prometheus.CounterValue, float64(atomic.LoadUint64(&dns.qps)), dns.nodeIP)
	ch <- prometheus.MustNewConstMetric(DNSCacheHitsRatio, prometheus.CounterValue, totalCacheHits/totalQueries, dns.nodeIP)
	ch <- prometheus.MustNewConstMetric(DNSQueriesTotal, prometheus.CounterValue, totalQueries, dns.nodeIP)
	for _, cs := range statistics.Server.Counters {
		switch cs.Type {
		case ServerCounterTypeNSStat:
			dns.collectNSStat(ch, totalQueries, cs.Counters)
		case ServerCounterTypeQType:
			dns.collectQTypeRatio(ch, totalQueries, cs.Counters)
		}
	}
}

func (dns *DNSCollector) collectCacheHits(ch chan<- prometheus.Metric, views []View) float64 {
	var totalCacheHits uint64
	for _, v := range views {
		for _, cs := range v.Counters {
			if cs.Type == ViewCounterTypeCacheStats {
				for _, c := range cs.Counters {
					if c.Name == CacheStatsQueryHits {
						ch <- prometheus.MustNewConstMetric(DNSCacheHits, prometheus.CounterValue, float64(c.Counter),
							dns.nodeIP, v.Name)
						totalCacheHits += c.Counter
						break
					}
				}
				break
			}
		}
	}

	return float64(totalCacheHits)
}

func (dns *DNSCollector) getQueryTotal(counters []Counters) (float64, bool) {
	for _, cs := range counters {
		if cs.Type == ServerCounterTypeOpCode {
			for _, c := range cs.Counters {
				if c.Name == OpcodeQUERY {
					return float64(c.Counter), true
				}
			}
		}
	}

	return 0, false
}

func (dns *DNSCollector) collectNSStat(ch chan<- prometheus.Metric, totalQueries float64, counters []Counter) {
	for _, c := range counters {
		switch c.Name {
		case QrySuccess, QryReferral, QryNxrrset, QrySERVFAIL, QryFORMERR, QryNXDOMAIN, QryDuplicate, QryDropped, QryFailure:
			ch <- prometheus.MustNewConstMetric(DNSResolvedRatios, prometheus.CounterValue,
				float64(c.Counter)/totalQueries, dns.nodeIP, strings.TrimPrefix(c.Name, "Qry"))
		}
	}
}

func (dns *DNSCollector) collectQTypeRatio(ch chan<- prometheus.Metric, totalQueries float64, counters []Counter) {
	for _, c := range counters {
		ch <- prometheus.MustNewConstMetric(DNSQueryTypeRatios, prometheus.CounterValue, float64(c.Counter)/totalQueries, dns.nodeIP, c.Name)
	}
}

func (dns *DNSCollector) getStats() (*DNSStatistics, error) {
	var stats DNSStatistics
	if err := dns.get(&stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

func (dns *DNSCollector) get(resp interface{}) error {
	httpResp, err := dns.httpClient.Get(dns.url)
	if err != nil {
		return fmt.Errorf("query dns stats failed: %s", err.Error())
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("query dns stats failed with status code %s", httpResp.Status)
	}

	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read dns stats response failed: %s", err.Error())
	}

	if err := xml.Unmarshal(body, resp); err != nil {
		return fmt.Errorf("unmarshal dns stats with XML failed: %s", err.Error())
	}

	return nil
}
