package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableUrlRedirect = restdb.ResourceDBType(&AgentUrlRedirect{})

type AgentUrlRedirect struct {
	resource.ResourceBase `json:",inline"`
	Domain                string `json:"domain" rest:"required=true" db:"uk"`
	Url                   string `json:"url" rest:"required=true,minLen=1,maxLen=500"`
	View                  string `json:"-" db:"uk"`
}
