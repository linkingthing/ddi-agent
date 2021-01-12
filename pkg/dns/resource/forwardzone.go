package resource

import (
	"fmt"
	"net"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableForwardZone = restdb.ResourceDBType(&AgentForwardZone{})

type AgentForwardZone struct {
	resource.ResourceBase `json:",inline"`
	Name                  string   `json:"name" rest:"required=true,minLen=1,maxLen=254" db:"uk"`
	ForwardType           string   `json:"forwardtype" rest:"required=true,options=only|first"`
	Addresses             []string `json:"addresses"`
	AgentView             string   `json:"-" db:"ownby,uk"`
}

func (forwardZone AgentForwardZone) ToZoneData() (ZoneData, error) {
	var addresses []string
	zoneData := ZoneData{Name: forwardZone.Name, ForwardType: forwardZone.ForwardType, IPs: addresses}

	for _, address := range forwardZone.Addresses {
		if net.ParseIP(address) != nil {
			addresses = append(addresses, address)
		} else if address_ := strings.Split(address, ":"); len(address_) == 2 {
			addresses = append(addresses, address_[0]+" port "+address_[1])
		} else {
			return zoneData, fmt.Errorf("bad forward address:%s", address_)
		}
	}

	zoneData.IPs = addresses
	return zoneData, nil
}
