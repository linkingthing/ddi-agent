package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableAcl = restdb.ResourceDBType(&AgentAcl{})

type AgentAcl struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=20" db:"uk"`
	Ips                   []string `json:"ips"`
}
