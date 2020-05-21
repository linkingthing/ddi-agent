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
	DNS        DNSConf        `yaml:"dns"`
	DHCP       DHCPConf       `yaml:"dhcp"`
	DB         BoltDBConf     `yaml:"db"`
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

type DNSConf struct {
	ConfDir string `yaml:"conf_dir"`
	DBDir   string `yaml:"db_dir"`
}

type DHCPConf struct {
	CmdAddr   string     `yaml:"cmd_addr"`
	ConfigDir string     `yaml:"config_dir"`
	DB        DHCPDBConf `yaml:"db"`
}

type DHCPDBConf struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	Port     uint32 `json:"port"`
}

type BoltDBConf struct {
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
