package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

var TableDnsGlobalConfig = restdb.ResourceDBType(&AgentDnsGlobalConfig{})

type AgentDnsGlobalConfig struct {
	restresource.ResourceBase `json:",inline"`
	LogEnable                 bool `json:"isLogOpen"`
	Ttl                       int  `json:"ttl" rest:"min=0,max=3000000"`
	DnssecEnable              bool `json:"isDnssecOpen"`
}
