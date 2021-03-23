package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableAgentUrlRedirect = restdb.ResourceDBType(&AgentUrlRedirect{})

type AgentUrlRedirect struct {
	resource.ResourceBase `json:",inline"`
	Domain                string `json:"domain" db:"uk"`
	Url                   string `json:"url"`
	IsHttps               bool   `json:"isHttps" db:"uk"`
}
