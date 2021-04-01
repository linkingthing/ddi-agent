package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableAgentNginxProxy = restdb.ResourceDBType(&AgentNginxProxy{})

type AgentNginxProxy struct {
	resource.ResourceBase `json:",inline"`
	Domain                string `json:"domain" db:"uk"`
	Url                   string `json:"url"`
	IsHttps               bool   `json:"isHttps" db:"uk"`
}
