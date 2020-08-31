package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableForward = restdb.ResourceDBType(&AgentForward{})

type AgentForward struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=50" db:"uk"`
	Ips                   []string `json:"ips" rest:"required=true"`
}
