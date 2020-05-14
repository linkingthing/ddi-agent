package metric

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/shell"

	"github.com/linkingthing/ddi-agent/config"
	"github.com/linkingthing/ddi-agent/pkg/boltdb"
)

const (
	DnsStatsFile = "named.stats"
)

type MetricsHandler struct {
	configPath    string
	HistoryLength int
	dbHandler     *boltdb.BoltHandler
}

func Init(conf *config.AgentConfig) {
	handler := MetricsHandler{
		configPath:    conf.Dns.ConfDir,
		HistoryLength: conf.Metric.HistoryLength,
	}
	handler.dbHandler = boltdb.New(conf.Dns.DBDir, "dnsmetrics.db")

	go handler.Statics(conf.Metric.Period)
	go handler.Exporter(conf.Dhcp.Addr, conf.Metric.Port)

}

func (h *MetricsHandler) Statics(tickerInterval int) {
	dnsStats := path.Join(h.configPath, DnsStatsFile)
	rndcPath := path.Join(h.configPath, "rndc")
	rndcConfPath := path.Join(h.configPath, "rndc.conf")
	ticker := time.NewTicker(time.Duration(tickerInterval) * time.Second)
	for {
		select {
		case <-h.Ticker.C:
			if _, err := shell.Shell(rndcPath, "-c"+rndcConfPath, "stats"); err != nil {
				log.Warnf("run rndc failed: %s", err.Error())
				continue
			}

			for {
				if _, err := os.Stat(dnsStats); err == nil {
					break
				}
			}

			if err := h.QueryStatics(); err != nil {
				log.Warnf("query statics failed: %s", err.Error())
			}

			if err := h.RecurQueryStatics(); err != nil {
				log.Warnf("recursive query statics failed: %s", err.Error())
			}

			if err := h.CacheHitStatis(); err != nil {
				log.Warnf("query mem hit statics failed: %s", err.Error())
			}

			if err := h.RetCodeStatics("NOERROR", TableNOERROR); err != nil {
				log.Warnf("static ret code noerror failed: %s", err.Error())
			}

			if err := h.RetCodeStatics("SERVFAIL", TableSERVFAIL); err != nil {
				log.Warnf("static ret code servfail failed: %s", err.Error())
			}

			if err := h.RetCodeStatics("NXDOMAIN", TableNXDOMAIN); err != nil {
				log.Warnf("static ret code nxdomain failed: %s", err.Error())
			}

			if err := h.RetCodeStatics("REFUSED", TableREFUSED); err != nil {
				log.Warnf("static ret code refused failed: %s", err.Error())
			}

			if err := os.Remove(dnsStats); err != nil {
				log.Warnf("remove stats file failed: %s", err.Error())
			}
		}
	}
}

func (h *MetricsHandler) QueryStatics() error {
	statsPath := path.Join(h.configPath, DnsStatsFile)
	key, value, err := getKeyAndValueFromStatsFile(statsPath, "QUERY", statsPath)
	if err != nil {
		return err
	}

	return h.SaveToDB(key, getStringLastBytes(value), TableQuery)
}

func getKeyAndValueFromStatsFile(keyParam string, valueParam ...string) (string, string, error) {
	key, err := shell.Shell("grep", "Dump ---", keyparam)
	if err != nil {
		return "", "", fmt.Errorf("get key from dns statistic file: %s", err.Error())
	}

	if key == "" {
		return "", "", fmt.Errorf("no found key from dns statistic file")
	}

	value, err := shell.Shell("grep", valueParam...)
	return string(getStringLastBytes(key)), value, err
}

func getStringLastBytes(value string) []byte {
	s := strings.Split(strings.TrimSuffix(value, "\n"), "\n")
	if slen := len(s); slen > 1 {
		return getNumBytesFromString(s[slen-1])
	}

	return nil
}

func getNumBytesFromString(s string) []byte {
	var bytes []byte
	for _, r := range s {
		if r >= '0' && r <= '9' {
			bytes = append(bytes, byte(r))
		}
	}
	return bytes
}

func (h *MetricsHandler) CacheHitStatis() error {
	statsPath := path.Join(h.configPath, DnsStatsFile)
	key, value, err := getKeyAndValueFromStatsFile(statsPath, "cache hits (from query)", statsPath)
	if err != nil {
		return err
	}

	return h.SaveToDB(key, getStatsTotal(value), TableCacheHit)
}

func getStatsTotal(value string) []byte {
	var total int
	for _, v := range strings.Split(strings.TrimSuffix(value, "\n"), "\n") {
		num, err := strconv.Atoi(getNumStrFromString(v))
		if err != nil {
			break
		}
		total += num
	}

	return []byte(strconv.Itoa(total))
}

func (h *MetricsHandler) RecurQueryStatics() error {
	statsPath := path.Join(h.configPath, DnsStatsFile)
	key, value, err := getKeyAndValueFromStatsFile(statsPath, "queries sent", statsPath)
	if err != nil {
		return err
	}

	return h.SaveToDB(key, getStatsTotal(value), TableRecurQuery)
}

func (h *MetricsHandler) RetCodeStatics(retCode string, table string) error {
	statsPath := path.Join(h.configPath, DnsStatsFile)
	key, value, err := getKeyAndValueFromStatsFile(statsPath, "-E", "[0-9]+ "+retCode+"$", statsPath)
	if err != nil {
		return err
	}

	var retcode []byte
	if values := strings.Split(strings.TrimSuffix(value, "\n"), "\n"); len(values) == 2 {
		retcode = getNumBytesFromString(values[0])
	}

	return h.SaveToDB(key, retcode, table)
}

func (h *MetricsHandler) SaveToDB(key string, value []byte, table string) error {
	values, err := h.dbHandler.TableKVs(table)
	if err != nil {
		return err
	}

	var timeStamps []string
	for k, _ := range values {
		timeStamps = append(timeStamps, k)
	}

	var delKeys []string
	count := len(values)
	sort.Strings(timeStamps)
	for i := 0; i < count-h.HistoryLength+1; i++ {
		delKeys = append(delKeys, timeStamps[i])
	}

	newKVs := map[string][]byte{key: value}
	if err := h.dbHandler.DeleteKVs(table, delKeys); err != nil {
		return err
	}

	if err := h.dbHandler.AddKVs(table, newKVs); err != nil {
		return err
	}
	return nil
}

func (h *MetricsHandler) Exporter(dhcpAddr, metricPort string) {
	registry := prometheus.NewRegistry()
	registry.MustRegister(newCollector(h.dbHandler, dhcpAddr))
	http.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	log.Fatal(http.ListenAndServe(":"+metricPort, nil))
}
