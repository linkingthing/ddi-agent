package main

import (
	"flag"

	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	"github.com/linkingthing/ddi-agent/config"
	"github.com/linkingthing/ddi-agent/pkg/db"
	dhcpconsumer "github.com/linkingthing/ddi-agent/pkg/dhcp/kafkaconsumer"
	"github.com/linkingthing/ddi-agent/pkg/dns"
	dnsconsumer "github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	"github.com/linkingthing/ddi-agent/pkg/grpcclient"
	"github.com/linkingthing/ddi-agent/pkg/grpcserver"
	"github.com/linkingthing/ddi-agent/pkg/kafkaproducer"
	"github.com/linkingthing/ddi-agent/pkg/metric"
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

	db.RegisterResources(dns.PersistentResources()...)
	if err := db.Init(conf); err != nil {
		log.Fatalf("new db failed: %s", err.Error())
	}

	m, err := metric.New(conf)
	if err != nil {
		log.Fatalf("new metric failed: %s", err.Error())
	}
	go m.Run()

	monitorConn, err := grpc.Dial(conf.Monitor.GrpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dial monitor grpc server failed: %s", err.Error())
	}
	defer monitorConn.Close()
	grpcclient.New(monitorConn)

	s, err := grpcserver.New(conf)
	if err != nil {
		log.Fatalf("new grpc server failed: %s", err.Error())
	}

	conn, err := grpc.Dial(conf.Server.GrpcAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dial grpc server failed: %s", err.Error())
	}
	defer conn.Close()

	kafkaproducer.Init(conf)
	go dnsconsumer.Run(conn, conf)
	go dhcpconsumer.Run(conn, conf)
	s.Run()
}
