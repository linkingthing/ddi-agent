package metric

import (
	"fmt"
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
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			assignedAddrsCount, err := dhcp.getAssignedAddrsCount()
			if err != nil {
				log.Warnf("get lps failed: %s", err.Error())
				continue
			}

			now := time.Now()
			if seconds := now.Sub(dhcp.lastGetTime).Seconds(); seconds > 0 && dhcp.lastAssignedAddrsCount != 0 {
				if assignedAddrsCount >= dhcp.lastAssignedAddrsCount {
					atomic.StoreUint64(&dhcp.lps, uint64((assignedAddrsCount-dhcp.lastAssignedAddrsCount)/now.Sub(dhcp.lastGetTime).Seconds()))
				}
			}

			dhcp.lastAssignedAddrsCount = assignedAddrsCount
			dhcp.lastGetTime = now
		}
	}
}

func (dhcp *DHCPCollector) getAssignedAddrsCount() (float64, error) {
	resp4s, err := dhcp.getStats(dhcpsrv.DHCP4Name)
	if err != nil {
		return 0, fmt.Errorf("get node %s dhcp4 stats failed: %s", dhcp.nodeIP, err.Error())
	}

	resp6s, err := dhcp.getStats(dhcpsrv.DHCP6Name)
	if err != nil {
		return 0, fmt.Errorf("get node %s dhcp6 stats failed: %s", dhcp.nodeIP, err.Error())
	}

	return getAssignedAddrsCountByVersion(resp4s, DHCPVersion4) + getAssignedAddrsCountByVersion(resp6s, DHCPVersion6), nil
}

func getAssignedAddrsCountByVersion(resps []dhcpsrv.DHCPCmdResponse, version string) float64 {
	subnetStats := make(SubnetStats)
	for _, resp := range resps {
		if statistics, ok := resp.Arguments.(map[string]interface{}); ok {
			for statsName, stats := range statistics {
				setDHCPSubnetAddressStats(statsName, stats, subnetStats, version)
			}
		}
	}

	var assignedAddrsCount float64
	for _, addrStats := range subnetStats {
		if addrStats.totalAddrsCount != 0 {
			assignedAddrsCount += addrStats.assignedAddrsCount
		}
	}

	return assignedAddrsCount
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

	ch <- prometheus.MustNewConstMetric(DHCPLPS, prometheus.GaugeValue, float64(atomic.LoadUint64(&dhcp.lps)), dhcp.nodeIP)
	var leasesCount float64
	if count, err := dhcp.Collect4(ch); err != nil {
		log.Warnf("collect node %s dhcp4 statistic failed: %s", dhcp.nodeIP, err.Error())
	} else {
		leasesCount += count
	}

	if count, err := dhcp.Collect6(ch); err != nil {
		log.Warnf("collect node %s dhcp6 statistic failed: %s", dhcp.nodeIP, err.Error())
	} else {
		leasesCount += count
	}

	ch <- prometheus.MustNewConstMetric(DHCPLeasesTotal, prometheus.GaugeValue, leasesCount, dhcp.nodeIP)
}

func (dhcp *DHCPCollector) Collect4(ch chan<- prometheus.Metric) (float64, error) {
	resps, err := dhcp.getStats(dhcpsrv.DHCP4Name)
	if err != nil {
		return 0, err
	}

	subnetStats := make(SubnetStats)
	for _, resp := range resps {
		if statistics, ok := resp.Arguments.(map[string]interface{}); ok {
			for statsName, stats := range statistics {
				switch statsName {
				case DHCP4StatsDiscover:
					dhcp.collectPacketStats(ch, DHCPVersion4, DHCP4PacketTypeDiscover, stats)
					continue
				case DHCP4StatsOffer:
					dhcp.collectPacketStats(ch, DHCPVersion4, DHCP4PacketTypeOffer, stats)
					continue
				case DHCP4StatsRequest:
					dhcp.collectPacketStats(ch, DHCPVersion4, DHCP4PacketTypeRequest, stats)
					continue
				case DHCP4StatsAck:
					dhcp.collectPacketStats(ch, DHCPVersion4, DHCP4PacketTypeAck, stats)
					continue
				}

				setDHCPSubnetAddressStats(statsName, stats, subnetStats, DHCPVersion4)
			}
		}
	}

	return dhcp.collectDHCPUsagesAndGetLeasesCount(ch, subnetStats), nil
}

func setDHCPSubnetAddressStats(statsName string, stats interface{}, subnetStats SubnetStats, version string) {
	if subnetID, totalAddrsCount, ok := getSubnetIdAndStatsValue(statsName, stats, SubnetTotalAddressesRegexp); ok {
		addrStats := subnetStats[subnetID]
		if version == DHCPVersion4 {
			addrStats.totalAddrsCount = totalAddrsCount
		} else {
			addrStats.totalAddrsCount += totalAddrsCount
		}
		subnetStats[subnetID] = addrStats
	} else if subnetID, assignedAddrsCount, ok := getSubnetIdAndStatsValue(statsName, stats, SubnetAssignedAddressesRegexp); ok {
		addrStats := subnetStats[subnetID]
		if version == DHCPVersion4 {
			addrStats.assignedAddrsCount = assignedAddrsCount
		} else {
			addrStats.assignedAddrsCount += assignedAddrsCount
		}
		subnetStats[subnetID] = addrStats
	}
}

func getSubnetIdAndStatsValue(statsName string, stats interface{}, subnetRegexp *regexp.Regexp) (string, float64, bool) {
	if slices := subnetRegexp.FindAllStringSubmatch(statsName, -1); len(slices) == 1 && len(slices[0]) == 2 {
		if v, ok := getStatsValue(stats); ok {
			return slices[0][1], v, true
		}
	}

	return "", 0, false
}

func (dhcp *DHCPCollector) collectDHCPUsagesAndGetLeasesCount(ch chan<- prometheus.Metric, subnetStats SubnetStats) float64 {
	var leasesCount float64
	for subnetID, addrStats := range subnetStats {
		if addrStats.totalAddrsCount != 0 {
			leasesCount += addrStats.assignedAddrsCount
			ch <- prometheus.MustNewConstMetric(DHCPUsages, prometheus.GaugeValue,
				addrStats.assignedAddrsCount/addrStats.totalAddrsCount, dhcp.nodeIP, subnetID)
		}
	}

	return leasesCount
}

func (dhcp *DHCPCollector) Collect6(ch chan<- prometheus.Metric) (float64, error) {
	resps, err := dhcp.getStats(dhcpsrv.DHCP6Name)
	if err != nil {
		return 0, err
	}

	subnetStats := make(SubnetStats)
	for _, resp := range resps {
		if statistics, ok := resp.Arguments.(map[string]interface{}); ok {
			for statsName, stats := range statistics {
				switch statsName {
				case DHCP6StatsSolicit:
					dhcp.collectPacketStats(ch, DHCPVersion6, DHCP6PacketTypeSolicit, stats)
					continue
				case DHCP6StatsAdvertise:
					dhcp.collectPacketStats(ch, DHCPVersion6, DHCP6PacketTypeAdvertise, stats)
					continue
				case DHCP6StatsRequest:
					dhcp.collectPacketStats(ch, DHCPVersion6, DHCP6PacketTypeRequest, stats)
					continue
				case DHCP6StatsReply:
					dhcp.collectPacketStats(ch, DHCPVersion6, DHCP6PacketTypeReply, stats)
					continue
				}

				setDHCPSubnetAddressStats(statsName, stats, subnetStats, DHCPVersion6)
			}
		}
	}

	return dhcp.collectDHCPUsagesAndGetLeasesCount(ch, subnetStats), nil
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

func (dhcp *DHCPCollector) getStats(service string) ([]dhcpsrv.DHCPCmdResponse, error) {
	resps, err := dhcpsrv.SendHttpRequestToDHCP(dhcp.httpClient, dhcp.url, &dhcpsrv.DHCPCmdRequest{
		Command:  GetStatisticAll,
		Services: []string{service},
	})
	if err != nil {
		return nil, err
	}

	return resps, nil
}
