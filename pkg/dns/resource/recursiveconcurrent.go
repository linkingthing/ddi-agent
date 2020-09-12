package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableRecursiveConcurrent = restdb.ResourceDBType(&AgentRecursiveConcurrent{})

type AgentRecursiveConcurrent struct {
	resource.ResourceBase `json:",inline"`
	RecursiveClients      uint32 `json:"recursiveclients" rest:"required=true"`
	FetchesPerZone        uint32 `json:"fetchesperzone" rest:"required=true"`
}
