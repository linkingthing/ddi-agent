package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableIpBlackHole = restdb.ResourceDBType(&AgentIpBlackHole{})

type AgentIpBlackHole struct {
	resource.ResourceBase `json:",inline"`
	Acl                   string `json:"acl" rest:"required=true"`
}
