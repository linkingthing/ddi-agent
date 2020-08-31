package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableView = restdb.ResourceDBType(&AgentView{})

type AgentView struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=20"`
	Priority              uint     `json:"priority" rest:"required=true,min=1,max=100"`
	Acls                  []string `json:"acls" rest:"required=true"`
	Dns64                 string   `json:"dns64" rest:"min=1,max=100"`
	Key                   string   `json:"-" db:"uk"`
}
