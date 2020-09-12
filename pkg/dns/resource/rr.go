package resource

import (
	"strconv"

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

func (rr AgentRr) ToRRData() RRData {
	return RRData{
		Name:  rr.Name,
		Type:  rr.DataType,
		Value: rr.Rdata,
		TTL:   strconv.FormatUint(uint64(rr.Ttl), 10),
	}
}
