package resource

import (
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
	View                  string `db:"uk"`
}