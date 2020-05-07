package config

import (
	"github.com/zdnscloud/cement/configure"
)

type AgentConfig struct {
	Path       string         `yaml:"-"`
	Server     ServerConf     `yaml:"server"`
	Grpc       GrpcConf       `yaml:"grpc"`
	Kafka      KafkaConf      `yaml:"kafka"`
	Prometheus PrometheusConf `yaml:"prometheus"`
	Metric     MetricConf     `yaml:"metric"`
	Dns        DnsConf        `yaml:"dns"`
}

type ServerConf struct {
	IsController bool   `yaml:"is_controller"`
	IsDHCP       bool   `yaml:"is_dhcp"`
	IsDNS        bool   `yaml:"is_dns"`
	IP           string `yaml:"ip"`
	Port         string `yaml:"port"`
	Hostname     string `yaml:"hostname"`
	ParentIP     string `yaml:"parent_ip"`
}

type GrpcConf struct {
	Addr string `yaml:"addr"`
}

type KafkaConf struct {
	Addr  string `yaml:"addr"`
	Topic string `yaml:"topic"`
}

type PrometheusConf struct {
	Addr string `yaml:"addr"`
}

type MetricConf struct {
	HistoryLength int `yaml:"history_length"`
	Period        int `yaml:"period"`
}

type DnsConf struct {
	ConfDir string `yaml:"conf_dir"`
	DBDir   string `yaml:"db_dir"`
}

func LoadConfig(path string) (*AgentConfig, error) {
	var conf AgentConfig
	conf.Path = path
	if err := conf.Reload(); err != nil {
		return nil, err
	}

	return &conf, nil
}

func (c *AgentConfig) Reload() error {
	var newConf AgentConfig
	if err := configure.Load(&newConf, c.Path); err != nil {
		return err
	}

	newConf.Path = c.Path
	*c = newConf
	return nil
}
