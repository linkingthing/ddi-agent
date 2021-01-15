package resource

import (
	"net"
	"strconv"
	"strings"

	"github.com/zdnscloud/g53"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

var TableAgentAuthZone = restdb.ResourceDBType(&AgentAuthZone{})

type AgentAuthZone struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string       `json:"name" db:"uk"`
	Ttl                       uint32       `json:"ttl"`
	Role                      AuthZoneRole `json:"role"`
	Slaves                    []string     `json:"slaves"`
	Masters                   []string     `json:"masters"`
	AgentView                 string       `json:"-" db:"ownby,uk"`
}

type AuthZoneRole string

const (
	AuthZoneRoleMaster AuthZoneRole = "master"
	AuthZoneRoleSlave  AuthZoneRole = "slave"
)

type ZoneData struct {
	Name         string
	Role         string
	Slaves       string
	Masters      string
	ZoneFile     string
	ForwardStyle string
	IPs          []string
}

type AuthZoneFileData struct {
	View     string
	Name     string
	NSName   string
	RootName string
	TTL      string
	RRs      []RR
}

func (zone *AgentAuthZone) GetZoneFile() string {
	return zone.AgentView + "#" + zone.Name + ".zone"
}

func (zone *AgentAuthZone) ToZoneData() ZoneData {
	var masters, slaves string
	if zone.Role == AuthZoneRoleMaster {
		slaves = formatAddress(zone.Slaves)
	} else if zone.Role == AuthZoneRoleSlave {
		masters = formatAddress(zone.Masters)
	}

	return ZoneData{Name: zone.Name, ZoneFile: zone.GetZoneFile(),
		Role: string(zone.Role), Masters: masters, Slaves: slaves}
}

func formatAddress(ipOrAddress []string) string {
	if len(ipOrAddress) == 0 {
		return ""
	}

	var addresses []string
	for _, address := range ipOrAddress {
		if net.ParseIP(address) != nil {
			addresses = append(addresses, address)
		} else if address_ := strings.Split(address, ":"); len(address_) == 2 {
			addresses = append(addresses, address_[0]+" port "+address_[1])
		}
	}

	return strings.Join(addresses, ";") + ";"
}

func (zone *AgentAuthZone) ToAuthZoneFileData() AuthZoneFileData {
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
	return AuthZoneFileData{
		View:     zone.AgentView,
		Name:     zone.Name,
		RootName: rootName,
		NSName:   nsName,
		TTL:      strconv.FormatUint(uint64(zone.Ttl), 10)}
}

func (zone *AgentAuthZone) Validate() error {
	name, err := g53.NameFromString(zone.Name)
	if err != nil {
		return err
	}
	zone.Name = name.String(true)
	return nil
}
