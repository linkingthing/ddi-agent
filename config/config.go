package config

import (
	"github.com/zdnscloud/cement/configure"
)

type AgentConfig struct {
	Path            string         `yaml:"-"`
	Server          ServerConf     `yaml:"server"`
	Controller      ControllerConf `yaml:"controller"`
	DNS             DNSConf        `yaml:"dns"`
	DHCP            DHCPConf       `yaml:"dhcp"`
	Grpc            GrpcConf       `yaml:"grpc"`
	Kafka           KafkaConf      `yaml:"kafka"`
	Prometheus      PrometheusConf `yaml:"prometheus"`
	Metric          MetricConf     `yaml:"metric"`
	DB              BoltDBConf     `yaml:"db"`
	NginxDefaultDir string         `yaml:"nginx_default_dir"`
}

type ServerConf struct {
	IP       string `yaml:"ip"`
	Hostname string `yaml:"hostname"`
}

type ControllerConf struct {
	IP            string `yaml:"ip"`
	IsCurrentNode bool   `yaml:"is_current_node"`
}

type DNSConf struct {
	Enabled   bool   `yaml:"enabled"`
	ConfDir   string `yaml:"conf_dir"`
	DBDir     string `yaml:"db_dir"`
	StatsAddr string `yaml:"stats_addr"`
	GroupID   string `yaml:"group_id"`
}

type DHCPConf struct {
	Enabled   bool       `yaml:"enabled"`
	CmdAddr   string     `yaml:"cmd_addr"`
	ConfigDir string     `yaml:"config_dir"`
	DB        DHCPDBConf `yaml:"db"`
	GroupID   string     `yaml:"group_id"`
}

type DHCPDBConf struct {
	Name     string `json:"name"`
	User     string `json:"user"`
	Password string `json:"password"`
	Port     uint32 `json:"port"`
	Host     string `json:"host"`
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
	Port uint32 `yaml:"port"`
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
