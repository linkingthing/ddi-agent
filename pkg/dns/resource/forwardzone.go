package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableAgentForwardZone = restdb.ResourceDBType(&AgentForwardZone{})

type AgentForwardZone struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" db:"uk"`
	ForwardStyle          string   `json:"forwardStyle"`
	Ips                   []string `json:"ips"`
	AgentView             string   `json:"-" db:"ownby,uk"`
}

func (forwardZone AgentForwardZone) ToZoneData() ZoneData {
	return ZoneData{
		Name:         forwardZone.Name,
		ForwardStyle: forwardZone.ForwardStyle,
		IPs:          forwardZone.Ips}
}
