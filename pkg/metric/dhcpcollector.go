package metric

import (
	"net/http"
	"net/url"
	"regexp"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-agent/config"
	dhcpsrv "github.com/linkingthing/ddi-agent/pkg/dhcp/grpcservice"
)

const (
	GetStatisticAll         = "statistic-get-all"
	DHCP4StatsDiscover      = "pkt4-discover-received"
	DHCP4StatsOffer         = "pkt4-offer-sent"
	DHCP4StatsRequest       = "pkt4-request-received"
	DHCP4StatsAck           = "pkt4-ack-sent"
	DHCP4PacketTypeDiscover = "discover"
	DHCP4PacketTypeOffer    = "offer"
	DHCP4PacketTypeRequest  = "request"
	DHCP4PacketTypeAck      = "ack"
	TimeLayout              = "2006-01-02 15:04:05"
)

var (
	SubnetRegexp                  = regexp.MustCompile(`^subnet\[(\d+)\]\.(\S+)`)
	SubnetTotalAddressesRegexp    = regexp.MustCompile(`^subnet\[(\d+)\]\.total-addresses`)
	SubnetAssignedAddressesRegexp = regexp.MustCompile(`^subnet\[(\d+)\]\.assigned-addresses`)
)

type SubnetStats map[string]SubnetAddressStats

type SubnetAddressStats struct {
	assignedAddrsCount uint64
	totalAddrsCount    uint64
}

type DHCPCollector struct {
	enabled      bool
	nodeIP       string
	url          string
	httpClient   *http.Client
	lastAckCount uint64
	lastGetTime  time.Time
	lps          uint64
}

func newDHCPCollector(conf *config.AgentConfig, cli *http.Client) (*DHCPCollector, error) {
	if conf.DHCP.Enabled {
		return &DHCPCollector{enabled: conf.DHCP.Enabled}, nil
	}

	cmdUrl, err := url.Parse(HttpScheme + conf.DHCP.CmdAddr)
	if err != nil {
		return nil, err
	}

	c := &DHCPCollector{
		enabled:    conf.DHCP.Enabled,
		nodeIP:     conf.Server.IP,
		url:        cmdUrl.String(),
		httpClient: cli,
	}
	go c.Run()
	return c, nil
}

func (dhcp *DHCPCollector) Run() {
	ticker := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ticker.C:
			statistics, err := dhcp.getStats()
			if err != nil {
				continue
			}

			var lps uint64
			for statsName, stats := range statistics {
				if statsName == DHCP4StatsAck {
					if v, ok := getStatsValue(stats); ok {
						now := time.Now()
						if dhcp.lastAckCount != 0 {
							lps = (v - dhcp.lastAckCount) / uint64(now.Sub(dhcp.lastGetTime).Seconds())
						}
						dhcp.lastAckCount = v
						dhcp.lastGetTime = now
					}
				}
			}

			atomic.StoreUint64(&dhcp.lps, lps)
		}
	}
}

func (dhcp *DHCPCollector) Describe(ch chan<- *prometheus.Desc) {
	if dhcp.enabled {
		for _, desc := range DHCPPrometheusDescs {
			ch <- desc
		}
	}
}

func (dhcp *DHCPCollector) Collect(ch chan<- prometheus.Metric) {
	if dhcp.enabled == false {
		return
	}

	statistics, err := dhcp.getStats()
	if err != nil {
		log.Warnf("get dhcp statistics with node %s failed: %s", dhcp.nodeIP, err.Error())
		return
	}

	ch <- prometheus.MustNewConstMetric(DHCPLPS, prometheus.GaugeValue, float64(atomic.LoadUint64(&dhcp.lps)), dhcp.nodeIP)

	subnetStats := make(SubnetStats)
	for statsName, stats := range statistics {
		switch statsName {
		case DHCP4StatsDiscover:
			dhcp.collectPacketStats(ch, DHCP4PacketTypeDiscover, stats)
		case DHCP4StatsOffer:
			dhcp.collectPacketStats(ch, DHCP4PacketTypeOffer, stats)
		case DHCP4StatsRequest:
			dhcp.collectPacketStats(ch, DHCP4PacketTypeRequest, stats)
		case DHCP4StatsAck:
			dhcp.collectPacketStats(ch, DHCP4PacketTypeAck, stats)
		}

		if totalAddrsSlice := SubnetTotalAddressesRegexp.FindAllStringSubmatch(statsName, -1); len(totalAddrsSlice) == 1 &&
			len(totalAddrsSlice[0]) == 2 {
			if v, ok := getStatsValue(stats); ok {
				if addrStats, ok := subnetStats[totalAddrsSlice[0][1]]; ok == false {
					subnetStats[totalAddrsSlice[0][1]] = SubnetAddressStats{totalAddrsCount: v}
				} else {
					addrStats.totalAddrsCount = v
					subnetStats[totalAddrsSlice[0][1]] = addrStats
				}
			}
		}

		if assignedAddrsSlice := SubnetAssignedAddressesRegexp.FindAllStringSubmatch(statsName, -1); len(assignedAddrsSlice) == 1 &&
			len(assignedAddrsSlice[0]) == 2 {
			if v, ok := getStatsValue(stats); ok {
				if addrStats, ok := subnetStats[assignedAddrsSlice[0][1]]; ok == false {
					subnetStats[assignedAddrsSlice[0][1]] = SubnetAddressStats{assignedAddrsCount: v}
				} else {
					addrStats.assignedAddrsCount = v
					subnetStats[assignedAddrsSlice[0][1]] = addrStats
				}
			}
		}
	}

	var leasesCount uint64
	for subnetID, addrStats := range subnetStats {
		if addrStats.totalAddrsCount != 0 {
			leasesCount += addrStats.assignedAddrsCount
			ch <- prometheus.MustNewConstMetric(DHCPUsages, prometheus.GaugeValue,
				float64(addrStats.assignedAddrsCount)/float64(addrStats.totalAddrsCount), dhcp.nodeIP, subnetID)
		}
	}

	ch <- prometheus.MustNewConstMetric(DHCPLeasesTotal, prometheus.GaugeValue,
		float64(leasesCount), dhcp.nodeIP)

}

func (dhcp *DHCPCollector) collectPacketStats(ch chan<- prometheus.Metric, packetType string, stats [][]interface{}) {
	if v, ok := getStatsValue(stats); ok {
		ch <- prometheus.MustNewConstMetric(DHCPPacketsStats, prometheus.GaugeValue, float64(v), dhcp.nodeIP, packetType)
	}
}

func getStatsValue(stats [][]interface{}) (uint64, bool) {
	for _, ss := range stats {
		for _, s := range ss {
			if v, ok := s.(uint64); ok {
				return v, true
			}
		}
	}

	return 0, false
}

func (dhcp *DHCPCollector) getStats() (map[string][][]interface{}, error) {
	resp, err := dhcpsrv.SendHttpRequestToDHCP(dhcp.httpClient, dhcp.url, &dhcpsrv.DHCPCmdRequest{
		Command: GetStatisticAll,
	})
	if err != nil {
		return nil, err
	}

	return resp[0].Arguments.(map[string][][]interface{}), nil
}
