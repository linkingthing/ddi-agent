package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableZone = restdb.ResourceDBType(&AgentZone{})

type AgentZone struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name" rest:"required=true,minLen=1,maxLen=254" db:"uk"`
	Ttl                   uint   `json:"ttl" rest:"required=true, min=0,max=3000000"`
	ZoneFile              string `json:"-"`
	RrsRole               string `json:"rrsRole"`
	View                  string `json:"-" db:"uk"`
}
