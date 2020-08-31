package main

import (
	"flag"

	"github.com/linkingthing/ddi-agent/pkg/dns"

	"github.com/linkingthing/ddi-agent/pkg/db"

	"github.com/linkingthing/ddi-agent/config"

	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	dhcpconsumer "github.com/linkingthing/ddi-agent/pkg/dhcp/kafkaconsumer"
	dnsconsumer "github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	"github.com/linkingthing/ddi-agent/pkg/grpcserver"
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

	s, err := grpcserver.New(conf)
	if err != nil {
		log.Fatalf("new grpc server failed: %s", err.Error())
	}

	conn, err := grpc.Dial(conf.Grpc.Addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dial grpc server failed: %s", err.Error())
	}
	defer conn.Close()

	go dnsconsumer.Run(conn, conf)
	go dhcpconsumer.Run(conn, conf)
	s.Run()
}
