package db

import (
	"fmt"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"

	"github.com/linkingthing/ddi-controller/pkg/dns/resource"
)

const globalConfigid = "globalConfig"

func GetDnsGlobalConfig() (*resource.DnsGlobalConfig, error) {
	var globalConfigs []*resource.DnsGlobalConfig
	globalConfig, err := restdb.GetResourceWithID(GetDB(), globalConfigid, &globalConfigs)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("get globalConfig resource %s from db failed: %s", globalConfigid, err.Error()))
	}
	return globalConfig.(*resource.DnsGlobalConfig), nil
}

func ListAcl() ([]*resource.Acl, *resterror.APIError) {
	var acls []*resource.Acl
	if err := GetResources(map[string]interface{}{"orderby": "create_time"}, &acls); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("ListAcls acls from db failed: %s", err.Error()))
	}

	return acls, nil
}

func ListView() ([]*resource.View, *resterror.APIError) {
	var views []*resource.View
	if err := GetResources(map[string]interface{}{"orderby": "priority"}, &views); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("ListViews views from db failed: %s", err.Error()))
	}

	return views, nil
}

func ListRedirection(parentID string) ([]*resource.Redirection, *resterror.APIError) {
	var redirections []*resource.Redirection
	if err := GetResources(map[string]interface{}{"view": parentID, "orderby": "create_time"}, &redirections); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError, fmt.Sprintf("list Redirection from db failed: %s", err.Error()))
	}
	return redirections, nil
}
