package resource

import (
	restdb "github.com/zdnscloud/gorest/db"
	"github.com/zdnscloud/gorest/resource"
)

var TableNginxProxy = restdb.ResourceDBType(&NginxProxy{})

type NginxProxy struct {
	resource.ResourceBase `json:",inline"`
	Domain                string `json:"domain" db:"uk"`
	Url                   string `json:"url"`
	IsHttps               bool   `json:"isHttps" db:"uk"`
}
