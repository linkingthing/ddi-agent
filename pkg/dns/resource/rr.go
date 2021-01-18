package resource

import (
	"fmt"
	"strconv"

	"github.com/zdnscloud/g53"
	restdb "github.com/zdnscloud/gorest/db"
	restresource "github.com/zdnscloud/gorest/resource"
)

var TableAgentAuthRR = restdb.ResourceDBType(&AgentAuthRr{})

type AgentAuthRr struct {
	restresource.ResourceBase `json:",inline"`
	Name                      string `json:"name" db:"uk"`
	RrType                    string `json:"rrType" db:"uk"`
	Ttl                       uint32 `json:"ttl"`
	Rdata                     string `json:"rdata" db:"uk"`
	Zone                      string `json:"-" db:"uk"`
	AgentView                 string `json:"-" db:"ownby,uk"`
}

type RR struct {
	Name  string
	Type  string
	TTL   string
	Rdata string
}

func (rr *AgentAuthRr) ToRR() (RR, error) {
	rdata, err := rr.formatRdata()
	if err != nil {
		return RR{}, err
	}

	return RR{
		Name:  rr.Name,
		Type:  rr.RrType,
		TTL:   strconv.FormatUint(uint64(rr.Ttl), 10),
		Rdata: rdata,
	}, nil
}

func (rr *AgentAuthRr) formatRdata() (string, error) {
	rrType, err := g53.TypeFromString(rr.RrType)
	if err != nil {
		return "", err
	}

	rdata, err := g53.RdataFromString(rrType, rr.Rdata)
	if err != nil {
		return "", err
	}

	return rdata.String(), nil
}

func (rr *AgentAuthRr) ToRRset() (*g53.RRset, error) {
	zoneName, err := g53.NameFromString(rr.Zone)
	if err != nil {
		return nil, fmt.Errorf("rr %s zone %s is invalid: %s", rr.Name, rr.Zone, err.Error())
	}
	rr.Zone = zoneName.String(true)

	name, err := g53.NameFromString(rr.Name)
	if err != nil {
		return nil, fmt.Errorf("zone %s rr name %s is invalid: %s", rr.Zone, rr.Name, err.Error())
	}

	var rrName *g53.Name

	if name.IsRoot() {
		rrName = zoneName
	} else {
		rr.Name = name.String(true)
		rrname, err := name.Concat(zoneName)
		if err != nil {
			return nil, fmt.Errorf("rr name.zone %s.%s is invalid: %s", rr.Name, rr.Zone, err.Error())
		} else {
			rrName = rrname
		}
	}

	rrType, err := g53.TypeFromString(rr.RrType)
	if err != nil {
		return nil, fmt.Errorf("zone %s rr type %s invalid: %s", rr.Zone, rr.RrType, err.Error())
	}

	rdata, err := g53.RdataFromString(rrType, rr.Rdata)
	if err != nil {
		return nil, fmt.Errorf("zone %s rr rdata %s invalid: %s", rr.Zone, rr.Rdata, err.Error())
	}

	return &g53.RRset{
		Name:   rrName,
		Type:   rrType,
		Class:  g53.CLASS_IN,
		Ttl:    g53.RRTTL(rr.Ttl),
		Rdatas: []g53.Rdata{rdata},
	}, nil
}
