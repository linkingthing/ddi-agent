package resource

import (
	"strconv"

	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableRedirection = restdb.ResourceDBType(&AgentRedirection{})

type AgentRedirection struct {
	resource.ResourceBase `json:",inline"`
	Name                  string `json:"name" rest:"required=true,minLen=1,maxLen=254" db:"uk"`
	Ttl                   uint   `json:"ttl" rest:"required=true, min=0,max=86401"`
	DataType              string `json:"datatype" rest:"required=true,options=A|AAAA|CNAME" db:"uk"`
	RedirectType          string `json:"redirecttype" rest:"required=true,options=localzone|nxdomain"`
	Rdata                 string `json:"rdata" rest:"required=true,minLen=1,maxLen=40"`
	AgentView             string `db:"ownby,uk"`
}

type RRData struct {
	Name  string
	Type  string
	Value string
	TTL   string
}

func (redirection AgentRedirection) ToRRData() RRData {
	return RRData{
		Name:  redirection.Name,
		TTL:   strconv.Itoa(int(redirection.Ttl)),
		Type:  redirection.DataType,
		Value: redirection.Rdata,
	}
}
