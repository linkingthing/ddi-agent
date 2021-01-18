package resource

import (
	"fmt"
	"strconv"

	"github.com/zdnscloud/g53"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

var TableAgentRedirection = restdb.ResourceDBType(&AgentRedirection{})

const (
	RRTypePTR     = "PTR"
	LocalZoneType = "localzone"
	NXDOMAIN      = "nxdomain"
)

type AgentRedirection struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" db:"uk"`
	Ttl                       uint32 `json:"ttl"`
	RrType                    string `json:"rrType" db:"uk"`
	RedirectType              string `json:"redirectType"`
	Rdata                     string `json:"rdata" db:"uk"`
	AgentView                 string `json:"-" db:"ownby,uk"`
}

func (r *AgentRedirection) ToRR() RR {
	return RR{
		Name:  r.Name,
		TTL:   strconv.Itoa(int(r.Ttl)),
		Type:  r.RrType,
		Rdata: r.Rdata,
	}
}

func (r *AgentRedirection) Validate() error {
	rrType, err := g53.TypeFromString(r.RrType)
	if err != nil {
		return fmt.Errorf("redirection %s type %s invalid: %s", r.Name, r.RrType, err.Error())
	}

	rdata, err := g53.RdataFromString(rrType, r.Rdata)
	if err != nil {
		return fmt.Errorf("redirection %s rdata %s invalid: %s", r.Name, r.Rdata, err.Error())
	}

	r.Rdata = rdata.String()
	if r.RedirectType == LocalZoneType {
		return nil
	}

	name, err := g53.NameFromString(r.Name)
	if err != nil {
		return fmt.Errorf("redirection name %s is invalid: %s", r.Name, err.Error())
	}

	r.Name = name.String(false)
	return nil
}
