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
	GetStatisticAll = "statistic-get-all"

	DHCPVersion4            = "4"
	DHCP4StatsDiscover      = "pkt4-discover-received"
	DHCP4StatsOffer         = "pkt4-offer-sent"
	DHCP4StatsRequest       = "pkt4-request-received"
	DHCP4StatsAck           = "pkt4-ack-sent"
	DHCP4PacketTypeDiscover = "discover"
	DHCP4PacketTypeOffer    = "offer"
	DHCP4PacketTypeRequest  = "request"
	DHCP4PacketTypeAck      = "ack"

	DHCPVersion6             = "6"
	DHCP6StatsSolicit        = "pkt6-solicit-received"
	DHCP6StatsAdvertise      = "pkt6-advertise-sent"
	DHCP6StatsRequest        = "pkt6-request-received"
	DHCP6StatsReply          = "pkt6-reply-sent"
	DHCP6PacketTypeSolicit   = "solicit"
	DHCP6PacketTypeAdvertise = "advertise"
	DHCP6PacketTypeRequest   = "request"
	DHCP6PacketTypeReply     = "reply"
)

var (
	SubnetTotalAddressesRegexp    = regexp.MustCompile(`^subnet\[(\d+)\]\.total-`)
	SubnetAssignedAddressesRegexp = regexp.MustCompile(`^subnet\[(\d+)\]\.assigned-`)
)

type SubnetStats map[string]SubnetAddressStats

type SubnetAddressStats struct {
	assignedAddrsCount float64
	totalAddrsCount    float64
}

type DHCPCollector struct {
	enabled                bool
	nodeIP                 string
	url                    string
	httpClient             *http.Client
	lastAssignedAddrsCount float64
	lastGetTime            time.Time
	lps                    uint64
}

func newDHCPCollector(conf *config.AgentConfig, cli *http.Client) (*DHCPCollector, error) {
	if conf.DHCP.Enabled == false {
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
			resps, err := dhcp.getStats()
			if err != nil {
				continue
			}

			var assignedAddrsCount float64
			for _, resp := range resps {
				if statistics, ok := resp.Arguments.(map[string]interface{}); ok {
					assignedAddrsCount += getSubnetStatsValue(statistics, SubnetAssignedAddressesRegexp)
				}
			}

			now := time.Now()
			if dhcp.lastAssignedAddrsCount != 0 {
				lps := (assignedAddrsCount - dhcp.lastAssignedAddrsCount) / now.Sub(dhcp.lastGetTime).Seconds()
				atomic.StoreUint64(&dhcp.lps, uint64(lps))
			}

			dhcp.lastAssignedAddrsCount = assignedAddrsCount
			dhcp.lastGetTime = now
		}
	}
}

func getSubnetStatsValue(statistics map[string]interface{}, subnetRegexp *regexp.Regexp) float64 {
	var count float64
	for statsName, stats := range statistics {
		if _, c, ok := getSubnetIdAndStatsValue(statsName, stats, subnetRegexp); ok {
			count += c
		}
	}
	return count
}

func getSubnetIdAndStatsValue(statsName string, stats interface{}, subnetRegexp *regexp.Regexp) (string, float64, bool) {
	if slices := subnetRegexp.FindAllStringSubmatch(statsName, -1); len(slices) == 1 && len(slices[0]) == 2 {
		if v, ok := getStatsValue(stats); ok {
			return slices[0][1], v, true
		}
	}

	return "", 0, false
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

	resps, err := dhcp.getStats()
	if err != nil {
		log.Warnf("get dhcp statistics with node %s failed: %s", dhcp.nodeIP, err.Error())
		return
	}

	ch <- prometheus.MustNewConstMetric(DHCPLPS, prometheus.GaugeValue, float64(atomic.LoadUint64(&dhcp.lps)), dhcp.nodeIP)

	subnetStats := make(SubnetStats)
	for _, resp := range resps {
		if statistics, ok := resp.Arguments.(map[string]interface{}); ok {
			for statsName, stats := range statistics {
				switch statsName {
				case DHCP4StatsDiscover:
					dhcp.collectPacketStats(ch, DHCPVersion4, DHCP4PacketTypeDiscover, stats)
				case DHCP4StatsOffer:
					dhcp.collectPacketStats(ch, DHCPVersion4, DHCP4PacketTypeOffer, stats)
				case DHCP4StatsRequest:
					dhcp.collectPacketStats(ch, DHCPVersion4, DHCP4PacketTypeRequest, stats)
				case DHCP4StatsAck:
					dhcp.collectPacketStats(ch, DHCPVersion4, DHCP4PacketTypeAck, stats)
				case DHCP6StatsSolicit:
					dhcp.collectPacketStats(ch, DHCPVersion6, DHCP6PacketTypeSolicit, stats)
				case DHCP6StatsAdvertise:
					dhcp.collectPacketStats(ch, DHCPVersion6, DHCP6PacketTypeAdvertise, stats)
				case DHCP6StatsRequest:
					dhcp.collectPacketStats(ch, DHCPVersion6, DHCP6PacketTypeRequest, stats)
				case DHCP6StatsReply:
					dhcp.collectPacketStats(ch, DHCPVersion6, DHCP6PacketTypeReply, stats)
				}

				if subnetID, totalAddrsCount, ok := getSubnetIdAndStatsValue(statsName, stats, SubnetTotalAddressesRegexp); ok {
					if addrStats, ok := subnetStats[subnetID]; ok == false {
						subnetStats[subnetID] = SubnetAddressStats{totalAddrsCount: totalAddrsCount}
					} else {
						addrStats.totalAddrsCount = totalAddrsCount
						subnetStats[subnetID] = addrStats
					}
				}

				if subnetID, assignedAddrsCount, ok := getSubnetIdAndStatsValue(statsName, stats, SubnetAssignedAddressesRegexp); ok {
					if addrStats, ok := subnetStats[subnetID]; ok == false {
						subnetStats[subnetID] = SubnetAddressStats{assignedAddrsCount: assignedAddrsCount}
					} else {
						addrStats.assignedAddrsCount = assignedAddrsCount
						subnetStats[subnetID] = addrStats
					}
				}
			}
		}
	}

	var leasesCount float64
	for subnetID, addrStats := range subnetStats {
		if addrStats.totalAddrsCount != 0 {
			leasesCount += addrStats.assignedAddrsCount
			ch <- prometheus.MustNewConstMetric(DHCPUsages, prometheus.GaugeValue,
				addrStats.assignedAddrsCount/addrStats.totalAddrsCount, dhcp.nodeIP, subnetID)
		}
	}

	ch <- prometheus.MustNewConstMetric(DHCPLeasesTotal, prometheus.GaugeValue, leasesCount, dhcp.nodeIP)
}

func (dhcp *DHCPCollector) collectPacketStats(ch chan<- prometheus.Metric, dhcpVersion, packetType string, stats interface{}) {
	if v, ok := getStatsValue(stats); ok {
		ch <- prometheus.MustNewConstMetric(DHCPPacketsStats, prometheus.GaugeValue, v, dhcp.nodeIP, dhcpVersion, packetType)
	}
}

func getStatsValue(statsInterface interface{}) (float64, bool) {
	statsInterfaces, ok := statsInterface.([]interface{})
	if ok == false {
		return 0, false
	}

	for _, stats := range statsInterfaces {
		ss, ok := stats.([]interface{})
		if ok == false {
			return 0, false
		}

		for _, s := range ss {
			if v, ok := s.(float64); ok {
				return v, true
			}
		}
	}

	return 0, false
}

func (dhcp *DHCPCollector) getStats() ([]dhcpsrv.DHCPCmdResponse, error) {
	resps, err := dhcpsrv.SendHttpRequestToDHCP(dhcp.httpClient, dhcp.url, &dhcpsrv.DHCPCmdRequest{
		Command:  GetStatisticAll,
		Services: []string{dhcpsrv.DHCP4Name, dhcpsrv.DHCP6Name},
	})
	if err != nil {
		return nil, err
	}

	return resps, nil
}
