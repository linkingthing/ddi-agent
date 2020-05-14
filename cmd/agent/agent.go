package main

import (
	"flag"
	"fmt"
	"strings"

	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	"github.com/linkingthing/ddi-agent/config"
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

	metric.Init(conf)
	s, err := grpcserver.New(conf)
	if err != nil {
		log.Fatalf("new grpc server failed: %s", err.Error())
	}

	conn, err := grpc.Dial(conf.Grpc.Addr, grpc.WithInsecure())
	if err != nil {
		log.Fatalf("dial grpc server failed: %s", err.Error())
	}
	defer conn.Close()

	go dnsconsumer.New(conn, conf)

	s.Run()
	s.Stop()
}
