package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type SortList struct {
	resource.ResourceBase `json:",inline"`
	Acls                  []string `json:"acls" rest:"required=true"`
}
