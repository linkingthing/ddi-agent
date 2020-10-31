package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

var TableDnsGlobalConfig = restdb.ResourceDBType(&AgentDnsGlobalConfig{})

type AgentDnsGlobalConfig struct {
	restresource.ResourceBase `json:",inline"`
	LogEnable                 bool     `json:"isLogOpen" rest:"required=true"`
	Ttl                       uint32   `json:"ttl" rest:"required=true,min=0,max=3000000"`
	DnssecEnable              bool     `json:"isDnssecOpen" rest:"required=true"`
	BlackholeEnable           bool     `json:"blackholeEnable"`
	Blackholes                []string `json:"blackholes"`
	RecursionEnable           bool     `json:"recursionEnable"`
	RecursiveClients          uint32   `json:"recursiveClients" rest:"required=true"`
}

func CreateDefaultResource() restresource.Resource {
	return &AgentDnsGlobalConfig{LogEnable: true, Ttl: 3600, RecursiveClients: 1000,
		DnssecEnable: false, BlackholeEnable: false, RecursionEnable: true}
}
