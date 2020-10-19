package resource

import (
	"strconv"
	"strings"

	"github.com/zdnscloud/g53"

	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableRR = restdb.ResourceDBType(&AgentRr{})

type AgentRr struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name" rest:"required=true,minLen=1,maxLen=256" db:"uk"`
	DataType              string `json:"datatype" rest:"required=true,options=A|AAAA|CNAME|HINFO|MX|NS|NAPTR|PTR|SRV|TXT" db:"uk"`
	Ttl                   uint   `json:"ttl" rest:"required=true, min=0,max=3000000"`
	Rdata                 string `json:"rdata" rest:"required=true" db:"uk"`
	RdataBackup           string `json:"rdataBackup"`
	ActiveRdata           string `json:"activeRdata"`
	Zone                  string `json:"-" db:"uk"`
	AgentView             string `json:"-" db:"ownby,uk"`
}

func (rr AgentRr) ToRRData(zoneName string) (RRData, error) {
	rdata, err := rr.formatRData(zoneName)
	if err != nil {
		return RRData{}, err
	}

	return RRData{
		Name:  rr.Name,
		Type:  rr.DataType,
		Value: rdata,
		TTL:   strconv.FormatUint(uint64(rr.Ttl), 10),
	}, nil
}

func (rr AgentRr) formatRData(zoneName string) (string, error) {
	rrType, err := g53.TypeFromString(rr.DataType)
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(rr.Rdata, "."+zoneName) {
		rdata, err := g53.RdataFromString(rrType, rr.Rdata)
		if err != nil {
			return "", err
		}
		return rdata.String(), nil
	}

	rdata, err := g53.RdataFromString(rrType, strings.TrimSuffix(rr.Rdata, "."+zoneName))
	if err != nil {
		return "", err
	}

	return rdata.String(), nil
}
