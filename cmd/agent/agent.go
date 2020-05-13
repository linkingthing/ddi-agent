package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/linkingthing/ddi-agent/config"
	dnsclient "github.com/linkingthing/ddi-agent/pkg/dns/client"
	"github.com/linkingthing/ddi-agent/pkg/grpcserver"
	"github.com/linkingthing/ddi-agent/pkg/metric/exporter"
	"github.com/linkingthing/ddi-metric/register"
	"github.com/linkingthing/ddi-metric/utils/currentdirectory"
	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"
)

var (
	configFile string
)

func main() {
	flag.StringVar(&configFile, "c", "agent.conf", "configure file path")
	flag.Parse()

	log.InitLogger(log.Debug)

	conf, err := config.LoadConfig(configFile)
	if err != nil {
		log.Fatalf("load config file failed: %s", err.Error())
	}
	handler := exporter.NewMetricsHandler(conf.Dns.ConfDir, conf.Metric.HistoryLength, conf.Metric.Period, conf.Dns.DBDir, conf.Dhcp.Addr)
	go handler.Statics()
	go handler.DNSExporter(conf.Metric.Port, "/metrics", "dns")
	prometheusAddr := strings.Split(conf.Prometheus.Addr, ":")
	if len(prometheusAddr) < 2 {
		log.Errorf("prometheus' address is not correct!")
		return
	}
	if conf.Server.IsDNS {
		register.RegisterNode(conf.Server.Hostname, prometheusAddr[0], prometheusAddr[1], conf.Server.IP, conf.Server.ParentIP, register.DNSRole, conf.Kafka.Addr)
	}
	if conf.Server.IsDHCP {
		register.RegisterNode(conf.Server.Hostname, prometheusAddr[0], prometheusAddr[1], conf.Server.IP, conf.Server.ParentIP, register.DHCPRole, conf.Kafka.Addr)
	}
	var currentPath *string
	currentPath, err = currentdirectory.GetCurrentDirectory()
	if err != nil {
		log.Fatalf("path is not current:%s", err.Error())
	}
	fmt.Println("currentPath:", *currentPath)
	s, err := grpcserver.NewGRPCServer(conf.Grpc.Addr, conf.Dns.ConfDir, *currentPath, conf.Server.IsDNS, conf.Server.IsDHCP)
	if err != nil {
		return
	}
	conn, err := grpc.Dial(conf.Grpc.Addr, grpc.WithInsecure())
	if err != nil {
		return
	}
	defer conn.Close()
	if conf.Server.IsDNS {
		go dnsclient.DNSClient(conn, conf.Kafka.Addr)
	}
	s.Start()
	defer s.Stop()
}
