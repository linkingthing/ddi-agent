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
	Dhcp       DhcpConf       `yaml:"dhcp"`
	DB         DBConf         `yaml:"db"`
}

type ServerConf struct {
	ControllerEnabled bool   `yaml:"controller_enabled"`
	DHCPEnabled       bool   `yaml:"dhcp_enabled"`
	DNSEnabled        bool   `yaml:"dns_enabled"`
	IP                string `yaml:"ip"`
	Hostname          string `yaml:"hostname"`
	ParentIP          string `yaml:"parent_ip"`
}

type GrpcConf struct {
	Addr string `yaml:"addr"`
}

type KafkaConf struct {
	Addr  string `yaml:"addr"`
	Topic string `yaml:"topic"`
}

type PrometheusConf struct {
	IP   string `yaml:"ip"`
	Port string `yaml:"port"`
}

type MetricConf struct {
	Port          string `yaml:"port"`
	HistoryLength int    `yaml:"history_length"`
	Period        int    `yaml:"period"`
}

type DnsConf struct {
	ConfDir string `yaml:"conf_dir"`
	DBDir   string `yaml:"db_dir"`
}

type DhcpConf struct {
	Addr string `yaml:"addr"`
}

type DBConf struct {
	Dir string `yaml:"dir"`
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
