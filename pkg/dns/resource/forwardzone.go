package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableForwardZone = restdb.ResourceDBType(&AgentForwardZone{})

type AgentForwardZone struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=254" db:"uk"`
	ForwardType           string   `json:"forwardtype" rest:"required=true,options=only|first"`
	ForwardIds            []string `json:"forward" db:"uk"`
	Ips                   []string `json:"ips"`
	AgentView             string   `json:"-" db:"ownby,uk"`
}

func (forwardZone AgentForwardZone) ToZoneData() ZoneData {
	return ZoneData{
		Name:        forwardZone.Name,
		ForwardType: forwardZone.ForwardType,
		IPs:         forwardZone.Ips}
}
