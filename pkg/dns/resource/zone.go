package resource

import (
	"strconv"

	"github.com/zdnscloud/g53"

	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableZone = restdb.ResourceDBType(&AgentZone{})

type AgentZone struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name" rest:"required=true,minLen=1,maxLen=254" db:"uk"`
	Ttl                   uint   `json:"ttl" rest:"required=true, min=0,max=3000000"`
	ZoneFile              string `json:"-"`
	AgentView             string `json:"-" db:"ownby,uk"`
}

type ZoneData struct {
	Name        string
	ZoneFile    string
	ForwardType string
	IPs         []string
}

type ZoneFileData struct {
	ViewName string
	Name     string
	NsName   string
	RootName string
	ZoneFile string
	TTL      string
	RRs      []RRData
}

func (zone *AgentZone) ToZoneData() ZoneData {
	return ZoneData{Name: zone.Name, ZoneFile: zone.ZoneFile}
}

func (zone *AgentZone) ToZoneFileData() ZoneFileData {
	var rootName, nsName string
	name, _ := g53.NameFromString(zone.Name)
	if zone.Name == "@" {
		zone.Name = name.String(true)
		rootName = "root."
		nsName = "ns."
	} else {
		zone.Name = name.String(false)
		rootName = "root." + zone.Name
		nsName = "ns." + zone.Name
	}
	return ZoneFileData{
		ViewName: zone.AgentView,
		Name:     zone.Name,
		RootName: rootName,
		NsName:   nsName,
		ZoneFile: zone.ZoneFile,
		TTL:      strconv.FormatUint(uint64(zone.Ttl), 10)}
}
