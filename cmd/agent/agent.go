package main

import (
	"flag"

	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-agent/config"
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
}
