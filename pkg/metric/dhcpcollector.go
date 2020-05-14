package metric

import (
	"encoding/json"
	"os/exec"
	"regexp"
)

type CurlKeaArguments struct {
	Pkt4Received []interface{} `json:"pkt4-received"`
}
type CurlKeaStats struct {
	Arguments CurlKeaArguments `json:"arguments"`
	Result    string           `json:"result"`
}

type CurlKeaStatsAll struct {
	Arguments map[string][]interface{} `json:"arguments"`
	Result    string                   `json:"result"`
}

type DHCPCollector struct {
	dhcpAddr string
}

func newDHCPCollector(addr string) *DHCPCollector {
	return &DHCPCollector{dhcpAddr: addr}
}

func (c *DHCPCollector) GetDhcpPacketStatistics() (float64, error) {
	out, err := RunCmd("curl -X POST http://\"" + c.dhcpAddr + "\"" + " -H 'Content-Type: application/json' -d '" +
		`   { "command": "statistic-get", "service": ["dhcp4"], "arguments": { "name": "pkt4-received" } } ' 2>/dev/null`)
	if err != nil {
		return 0, err
	}

	if len(out) <= 1 {
		return 0, nil
	}

	var curlRet CurlKeaStats
	if err := json.Unmarshal(out[1:len(out)-1], &curlRet); err != nil {
		return 0, err
	}

	return float64(len(curlRet.Arguments.Pkt4Received)), nil
}

func GetKeaStatisticsAll(dhcpAddr string) (*CurlKeaStatsAll, error) {
	out, err := RunCmd("curl -X POST http://\"" + dhcpAddr + "\"" + " -H 'Content-Type: application/json' -d '" +
		`   { "command": "statistic-get-all", "service": ["dhcp4"], "arguments": { }}' 2>/dev/null`)
	if err != nil {
		return nil, err
	}

	if len(out) <= 1 {
		return nil, nil
	}

	var curlRet CurlKeaStatsAll
	if err := json.Unmarshal(out[1:len(out)-1], &curlRet); err != nil {
		return nil, err
	}

	return &curlRet, nil
}

func (c *DHCPCollector) GetDhcpLeasesStatistics() (float64, error) {
	curlRet, err := GetKeaStatisticsAll(c.dhcpAddr)
	if err != nil {
		return 0, err
	}

	if curlRet == nil {
		return 0, nil
	}

	leaseNum := 0
	rex := regexp.MustCompile(`^subnet\[(\d+)\]\.assigned-addresses`)
	for k, v := range curlRet.Arguments {
		out := rex.FindAllStringSubmatch(k, -1)
		if len(out) > 0 {
			for range out {
				leaseNum += len(v)
			}
		}
	}

	return float64(leaseNum), nil
}

func (c *DHCPCollector) GetDhcpUsageStatistics() (float64, error) {
	out, err := RunCmd("curl -X POST http://\"" + c.dhcpAddr + "\"" + " -H 'Content-Type: application/json' -d '" +
		`   { "command": "statistic-get-all", "service": ["dhcp4"], "arguments": { } } ' 2>/dev/null`)
	if err != nil {
		return 0, err
	}

	if len(out) <= 1 {
		return 0, nil
	}

	var curlRet CurlKeaStatsAll
	if err := json.Unmarshal(out[1:len(out)-1], &curlRet); err != nil {
		return 0, err
	}

	var leaseNum, totalNum float64
	rex := regexp.MustCompile(`^subnet\[(\d+)\]\.(\S+)`)
	for k, v := range curlRet.Arguments {
		if len(v) == 0 {
			continue
		}

		for _, i := range rex.FindAllStringSubmatch(k, -1) {
			if len(i) < 3 {
				continue
			}

			addrType := i[2]
			if addrType == "total-addresses" {
				if totals, ok := v[0].([]interface{}); ok {
					totalNum += totals[0].(float64)
				}
			} else if addrType == "assigned-addresses" {
				leaseNum += float64(len(v))
			}
		}
	}

	if totalNum > 0 {
		return leaseNum / totalNum * 100, nil
	}

	return 0, nil
}

func RunCmd(command string) ([]byte, error) {
	return exec.Command("bash", "-c", command).CombinedOutput()
}
