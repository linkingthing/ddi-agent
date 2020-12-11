package config

import (
	"github.com/zdnscloud/cement/configure"
)

type AgentConfig struct {
	Path            string         `yaml:"-"`
	Server          ServerConf     `yaml:"server"`
	DNS             DNSConf        `yaml:"dns"`
	DHCP            DHCPConf       `yaml:"dhcp"`
	Kafka           KafkaConf      `yaml:"kafka"`
	Prometheus      PrometheusConf `yaml:"prometheus"`
	Metric          MetricConf     `yaml:"metric"`
	DB              DBConf         `yaml:"db"`
	NginxDefaultDir string         `yaml:"nginx_default_dir"`
	Monitor         MonitorConf    `yaml:"monitor"`
}

type ServerConf struct {
	IP       string `yaml:"ip"`
	IPV6     string `yaml:"ipv6"`
	Hostname string `yaml:"hostname"`
	GrpcAddr string `yaml:"grpc_addr"`
}

type DNSConf struct {
	Enabled   bool   `yaml:"enabled"`
	ConfDir   string `yaml:"conf_dir"`
	DBDir     string `yaml:"db_dir"`
	StatsAddr string `yaml:"stats_addr"`
	GroupID   string `yaml:"group_id"`
	ServerIp  string `yaml:"server_ip"`
	Dbport    uint32 `yaml:"db_port"`
	Dbhost    string `yaml:"db_host"`
}

type DHCPConf struct {
	Enabled   bool   `yaml:"enabled"`
	CmdAddr   string `yaml:"cmd_addr"`
	ConfigDir string `yaml:"config_dir"`
	GroupID   string `yaml:"group_id"`
}

type KafkaConf struct {
	Addr  []string `yaml:"addr"`
	Topic string   `yaml:"topic"`
}

type PrometheusConf struct {
	IP   string `yaml:"ip"`
	Port string `yaml:"port"`
}

type MetricConf struct {
	Port uint32 `yaml:"port"`
}

type MonitorConf struct {
	GrpcAddr string `yaml:"grpc_addr"`
}

type DBConf struct {
	Name     string `yaml:"name"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
	Port     uint32 `yaml:"port"`
	Host     string `json:"host"`
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
