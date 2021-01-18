package resource

import (
	"net"
	"strconv"

	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableAgentForwardZone = restdb.ResourceDBType(&AgentForwardZone{})

type AgentForwardZone struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=254" db:"uk"`
	ForwardStyle          string   `json:"forwardStyle" rest:"required=true,options=only|first"`
	Addresses             []string `json:"addresses"`
	AgentView             string   `json:"-" db:"ownby,uk"`
}

func (forwardZone AgentForwardZone) ToZoneData() (ZoneData, error) {
	var addresses []string
	for _, address := range forwardZone.Addresses {
		if net.ParseIP(address) != nil {
			addresses = append(addresses, address)
		} else if addr, err := net.ResolveTCPAddr("tcp", address); err != nil {
			return ZoneData{}, err
		} else {
			addresses = append(addresses, addr.IP.String()+" port "+strconv.Itoa(addr.Port))
		}
	}

	return ZoneData{Name: forwardZone.Name, ForwardStyle: forwardZone.ForwardStyle, IPs: addresses}, nil
}
