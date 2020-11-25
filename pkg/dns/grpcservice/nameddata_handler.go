package grpcservice

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/ddi-agent/pkg/db"
	"github.com/linkingthing/ddi-agent/pkg/dns/dbhandler"
	"github.com/linkingthing/ddi-agent/pkg/dns/resource"
	"github.com/linkingthing/ddi-agent/pkg/grpcclient"
	monitorpb "github.com/linkingthing/ddi-monitor/pkg/proto"
)

type NamedData struct {
	ConfigPath       string
	NamedAclPath     string
	NamedViewPath    string
	NamedOptionsPath string
}

type NamedOption struct {
	ConfigPath       string
	DnssecEnable     bool
	LogEnable        bool
	BlackholeEnable  bool
	Blackholes       []string
	RecursionEnable  bool
	RecursiveClients uint32
}

type NamedViews struct {
	Views []View
}

type NamedAcl struct {
	ConfigPath string
	Acls       []ACL
}

type View struct {
	Name      string
	ACLs      []ACL
	Zones     []resource.ZoneData
	Redirect  *Redirect
	RPZ       *Rpz
	DNS64s    []Dns64
	Key       string
	DeniedIPs []string
}

type ACL struct {
	ID   string
	Name string
	Ips  []string
}

type Dns64 struct {
	ID              string
	Prefix          string
	ClientACLName   string
	AAddressACLName string
}

type Rpz struct {
	RRs []resource.RRData
}

type Redirect struct {
	RRs []resource.RRData
}

type RedirectionData struct {
	ViewName string
	RRs      []resource.RRData
}

type nzfData struct {
	ViewName string
	Zones    []resource.ZoneData
}

type nginxDefaultConf struct {
	URLRedirects []urlRedirect
}

type urlRedirect struct {
	Domain string
	URL    string
}

func (handler *DNSHandler) initFiles() error {
	handler.nginxConfPath = filepath.Join(handler.nginxDefaultConfDir, nginxDefaultConfFile)
	handler.namedViewPath = filepath.Join(handler.dnsConfPath, namedViewConfName)
	handler.namedOptionPath = filepath.Join(handler.dnsConfPath, namedOptionsConfName)
	handler.namedAclPath = filepath.Join(handler.dnsConfPath, namedAclConfName)
	if handler.nginxConfPath == "" || handler.namedViewPath == "" || handler.namedOptionPath == "" || handler.namedAclPath == "" {
		return fmt.Errorf("init path failed path is empty")
	}

	if err := createOneFolder(filepath.Join(handler.dnsConfPath, "redirection")); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := handler.initNamedConf(); err != nil {
			return fmt.Errorf("initNamedConf failed:%s", err.Error())
		}
		if err := handler.initNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("init rewriteNamedOptionsFile failed:%s", err.Error())
		}
		if err := handler.initNamedViewFile(tx); err != nil {
			return fmt.Errorf("initNamedViewFile failed:%s", err.Error())
		}
		if err := handler.initNamedAclFile(tx); err != nil {
			return fmt.Errorf("init rewriteNamedFile failed:%s", err.Error())
		}
		if err := handler.initZoneFiles(tx); err != nil {
			return fmt.Errorf("initZoneFiles failed:%s", err.Error())
		}
		if err := handler.rewriteNzfsFile(tx); err != nil {
			return fmt.Errorf("init rewriteNzfsFile failed:%s", err.Error())
		}
		if err := handler.initRPZFile(tx); err != nil {
			return fmt.Errorf("init rewriteRPZFile failed:%s", err.Error())
		}
		if err := handler.initRedirectFile(tx); err != nil {
			return fmt.Errorf("init rewriteRedirectFile failed:%s", err.Error())
		}
		if err := handler.rewriteNginxHttpFile(tx); err != nil {
			return fmt.Errorf("rewrite nginx config file error:%s", err.Error())
		}

		return nil
	})
}

func (handler *DNSHandler) initNamedConf() error {
	data := &NamedData{
		NamedAclPath:     filepath.Join(handler.dnsConfPath, namedAclConfName),
		NamedViewPath:    filepath.Join(handler.dnsConfPath, namedViewConfName),
		NamedOptionsPath: filepath.Join(handler.dnsConfPath, namedOptionsConfName),
	}
	return handler.flushTemplateFiles(namedTpl, filepath.Join(handler.dnsConfPath, mainConfName), data)
}

func (handler *DNSHandler) rndcReconfig() error {
	_, err := grpcclient.GetDDIMonitorGrpcClient().ReconfigDNS(context.Background(), &monitorpb.ReconfigDNSRequest{})
	return err
}

func (handler *DNSHandler) rndcReload() error {
	_, err := grpcclient.GetDDIMonitorGrpcClient().ReloadDNSConfig(context.Background(), &monitorpb.ReloadDNSConfigRequest{})
	return err
}

func (handler *DNSHandler) rndcAddZone(zoneName string, zoneFile string, viewName string) error {
	_, err := grpcclient.GetDDIMonitorGrpcClient().AddDNSZone(context.Background(), &monitorpb.AddDNSZoneRequest{
		ZoneName: zoneName,
		ViewName: viewName,
		ZoneFile: zoneFile,
	})
	return err
}

func (handler *DNSHandler) rndcModifyZone(zoneName string, zoneFile string, viewName string) error {
	_, err := grpcclient.GetDDIMonitorGrpcClient().UpdateDNSZone(context.Background(), &monitorpb.UpdateDNSZoneRequest{
		ZoneName: zoneName,
		ViewName: viewName,
		ZoneFile: zoneFile,
	})
	return err
}

func (handler *DNSHandler) rndcDeleteZone(zoneName string, viewName string) error {
	_, err := grpcclient.GetDDIMonitorGrpcClient().DeleteDNSZone(context.Background(), &monitorpb.DeleteDNSZoneRequest{
		ZoneName: zoneName,
		ViewName: viewName,
	})
	return err
}

func (handler *DNSHandler) rndcDumpJNLFile() error {
	_, err := grpcclient.GetDDIMonitorGrpcClient().DumpDNSAllZonesConfig(context.Background(), &monitorpb.DumpDNSAllZonesConfigRequest{})
	return err
}

func (handler *DNSHandler) rndcZoneDumpJNLFile(zoneName string, viewName string) error {
	_, err := grpcclient.GetDDIMonitorGrpcClient().DumpDNSZoneConfig(context.Background(), &monitorpb.DumpDNSZoneConfigRequest{
		ZoneName: zoneName,
		ViewName: viewName,
	})
	return err
}

func (handler *DNSHandler) rewriteNginxHttpFile(tx restdb.Transaction) error {
	data := nginxDefaultConf{}
	var urlRedirectList []*resource.AgentUrlRedirect
	if err := dbhandler.ListWithTx(&urlRedirectList, tx); err != nil {
		return err
	}

	for _, urlValue := range urlRedirectList {
		if !urlValue.IsHttps {
			data.URLRedirects = append(data.URLRedirects,
				urlRedirect{Domain: urlValue.Domain, URL: urlValue.Url})
		}
	}

	if err := handler.flushTemplateFiles(nginxDefaultTpl,
		handler.nginxConfPath, data); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) addNginxHttpsFile(key, crt []byte, urlRedirect *resource.AgentUrlRedirect) error {
	if err := createOneFolder(handler.nginxKeyDir); err != nil {
		return fmt.Errorf("create folder:%s  failed:%s", handler.nginxKeyDir, err.Error())
	}
	if err := ioutil.WriteFile(
		path.Join(handler.nginxKeyDir, urlRedirect.Domain+".key"), key, FilePermissions); err != nil {
		return fmt.Errorf("writeNginxSSLFile key failed:%s", err.Error())
	}
	if err := ioutil.WriteFile(
		path.Join(handler.nginxKeyDir, urlRedirect.Domain+".crt"), crt, FilePermissions); err != nil {
		return fmt.Errorf("writeNginxSSLFile crt failed:%s", err.Error())
	}

	return handler.flushTemplateFiles(nginxSslTpl,
		path.Join(handler.nginxDefaultConfDir, urlRedirect.Domain+".conf"), urlRedirect)
}

func (handler *DNSHandler) updateNginxHttpsFile(urlRedirect *resource.AgentUrlRedirect) error {
	domainConf := urlRedirect.Domain + ".conf"
	if err := removeOneFile(path.Join(handler.nginxDefaultConfDir, domainConf)); err != nil {
		return fmt.Errorf("updateNginxHttpsFile  remove file:%s  failed:%s", domainConf, err.Error())
	}

	return handler.flushTemplateFiles(nginxSslTpl, path.Join(handler.nginxDefaultConfDir, domainConf), urlRedirect)
}

func (handler *DNSHandler) removeNginxHttpsFile(domain string) error {
	domainConf := domain + ".conf"
	if err := removeOneFile(path.Join(handler.nginxDefaultConfDir, domainConf)); err != nil {
		return fmt.Errorf("removeNginxHttpsFile file:%s  failed:%s", domainConf, err.Error())
	}
	if err := removeOneFile(path.Join(handler.nginxKeyDir, domain+".key")); err != nil {
		return fmt.Errorf("removeNginxHttpsFile file:%s  failed:%s", handler.nginxKeyDir, err.Error())
	}
	if err := removeOneFile(path.Join(handler.nginxKeyDir, domain+".crt")); err != nil {
		return fmt.Errorf("removeNginxHttpsFile file:%s  failed:%s", handler.nginxKeyDir, err.Error())
	}

	return nil
}

func (handler *DNSHandler) nginxReload() error {
	_, err := grpcclient.GetDDIMonitorGrpcClient().ReloadNginxConfig(context.Background(),
		&monitorpb.ReloadNginxConfigRequest{})
	return err
}

func (handler *DNSHandler) initNamedViewFile(tx restdb.Transaction) error {
	viewConfigData := &NamedViews{}
	var viewList []*resource.AgentView
	if err := dbhandler.ListByConditionWithTx(&viewList,
		map[string]interface{}{"orderby": "priority"}, tx); err != nil {
		return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
	}

	var redirectionList []*resource.AgentRedirection
	if err := dbhandler.ListByConditionWithTx(&redirectionList,
		map[string]interface{}{}, tx); err != nil {
		return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
	}

	var forwardZoneList []*resource.AgentForwardZone
	if err := dbhandler.ListByConditionWithTx(&forwardZoneList,
		map[string]interface{}{}, tx); err != nil {
		return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
	}

	for _, value := range viewList {
		var acls []ACL
		for _, aclValue := range value.Acls {
			if aclValue != "none" {
				acls = append(acls, ACL{Name: aclValue})
			}
		}
		view := View{Name: value.Name, Key: value.Key}
		if len(handler.interfaceIPs) > 0 {
			view.DeniedIPs = handler.interfaceIPs
		}
		if len(acls) > 0 {
			view.ACLs = acls
		}

		for _, reValue := range redirectionList {
			if reValue.AgentView == value.ID {
				if reValue.RedirectType == localZoneType {
					view.RPZ = &Rpz{[]resource.RRData{reValue.ToRRData()}}
				} else if reValue.RedirectType == nxDomain {
					view.Redirect = &Redirect{[]resource.RRData{reValue.ToRRData()}}
				}
			}
		}

		if value.Dns64 != "" {
			view.DNS64s = append(view.DNS64s, Dns64{
				Prefix:          value.Dns64,
				AAddressACLName: anyACL,
				ClientACLName:   anyACL})
		}

		for _, forwardZone := range forwardZoneList {
			if forwardZone.AgentView == value.ID {
				view.Zones = append(view.Zones, forwardZone.ToZoneData())
			}
		}

		viewConfigData.Views = append(viewConfigData.Views, view)
	}

	if err := handler.flushTemplateFiles(namedViewTpl,
		handler.namedViewPath, viewConfigData); err != nil {
		return fmt.Errorf("flushTemplateFiles failed :%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) rewriteNamedViewFile(existRPZ bool, tx restdb.Transaction) error {
	viewConfigData := &NamedViews{}
	var viewList []*resource.AgentView

	if err := dbhandler.ListByConditionWithTx(&viewList,
		map[string]interface{}{"orderby": "priority"}, tx); err != nil {
		return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
	}

	var redirectionList []*resource.AgentRedirection
	if err := dbhandler.ListByConditionWithTx(&redirectionList,
		map[string]interface{}{}, tx); err != nil {
		return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
	}

	var forwardZoneList []*resource.AgentForwardZone
	if err := dbhandler.ListByConditionWithTx(&forwardZoneList,
		map[string]interface{}{}, tx); err != nil {
		return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
	}

	for _, value := range viewList {
		var acls []ACL
		for _, aclValue := range value.Acls {
			if aclValue != "none" {
				acls = append(acls, ACL{Name: aclValue})
			}
		}
		view := View{Name: value.Name, Key: value.Key}
		if len(handler.interfaceIPs) > 0 {
			view.DeniedIPs = handler.interfaceIPs
		}
		if len(acls) > 0 {
			view.ACLs = acls
		}

		for _, reValue := range redirectionList {
			if reValue.AgentView == value.ID {
				if reValue.RedirectType == localZoneType {
					if !existRPZ {
						view.RPZ = &Rpz{[]resource.RRData{reValue.ToRRData()}}
					}
				} else if reValue.RedirectType == nxDomain {
					view.Redirect = &Redirect{[]resource.RRData{reValue.ToRRData()}}
				}
			}
		}

		if value.Dns64 != "" {
			view.DNS64s = append(view.DNS64s, Dns64{
				Prefix:          value.Dns64,
				AAddressACLName: anyACL,
				ClientACLName:   anyACL})
		}

		for _, forwardZone := range forwardZoneList {
			if forwardZone.AgentView == value.ID {
				view.Zones = append(view.Zones, forwardZone.ToZoneData())
			}
		}

		viewConfigData.Views = append(viewConfigData.Views, view)
	}

	if err := handler.flushTemplateFiles(namedViewTpl,
		handler.namedViewPath, viewConfigData); err != nil {
		return fmt.Errorf("flushTemplateFiles failed :%s", err.Error())
	}

	return handler.rndcReconfig()
}

func (handler *DNSHandler) initNamedOptionsFile(tx restdb.Transaction) error {
	namedOptionData := &NamedOption{ConfigPath: handler.dnsConfPath}
	globalConfigRes, err := dbhandler.GetWithTx(defaultGlobalConfigID,
		&[]*resource.AgentDnsGlobalConfig{}, tx)
	if err != nil {
		return err
	}
	globalConfig := globalConfigRes.(*resource.AgentDnsGlobalConfig)
	namedOptionData.LogEnable = globalConfig.LogEnable
	namedOptionData.DnssecEnable = globalConfig.DnssecEnable
	namedOptionData.BlackholeEnable = globalConfig.BlackholeEnable
	namedOptionData.Blackholes = globalConfig.Blackholes
	namedOptionData.RecursionEnable = globalConfig.RecursionEnable
	namedOptionData.RecursiveClients = globalConfig.RecursiveClients

	if err := handler.flushTemplateFiles(namedOptionsTpl,
		handler.namedOptionPath, namedOptionData); err != nil {
		return fmt.Errorf("flushTemplateFiles failed :%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) rewriteNamedOptionsFile(tx restdb.Transaction) error {
	namedOptionData := &NamedOption{ConfigPath: handler.dnsConfPath}

	globalConfigRes, err := dbhandler.GetWithTx(defaultGlobalConfigID,
		&[]*resource.AgentDnsGlobalConfig{}, tx)
	if err != nil {
		return err
	}

	globalConfig := globalConfigRes.(*resource.AgentDnsGlobalConfig)
	namedOptionData.LogEnable = globalConfig.LogEnable
	namedOptionData.DnssecEnable = globalConfig.DnssecEnable
	namedOptionData.BlackholeEnable = globalConfig.BlackholeEnable
	if len(globalConfig.Blackholes) == 0 {
		namedOptionData.BlackholeEnable = false
	}
	namedOptionData.Blackholes = globalConfig.Blackholes
	namedOptionData.RecursionEnable = globalConfig.RecursionEnable
	namedOptionData.RecursiveClients = globalConfig.RecursiveClients

	if err := handler.flushTemplateFiles(namedOptionsTpl,
		handler.namedOptionPath, namedOptionData); err != nil {
		return fmt.Errorf("flushTemplateFiles failed :%s", err.Error())
	}

	return handler.rndcReconfig()
}

func (handler *DNSHandler) initNamedAclFile(tx restdb.Transaction) error {
	namedAcl := &NamedAcl{ConfigPath: handler.dnsConfPath}
	var aclList []*resource.AgentAcl
	if err := dbhandler.ListWithTx(&aclList, tx); err != nil {
		return fmt.Errorf("initNamedAclFile failed:%s", err.Error())
	}

	for _, acl := range aclList {
		if acl.ID != anyACL && acl.ID != noneACL {
			namedAcl.Acls = append(namedAcl.Acls, ACL{Name: acl.Name, Ips: acl.Ips})
		}
	}

	return handler.flushTemplateFiles(namedAclTpl,
		handler.namedAclPath, namedAcl)
}

func (handler *DNSHandler) rewriteNamedAclFile(tx restdb.Transaction) error {
	namedAcl := &NamedAcl{ConfigPath: handler.dnsConfPath}
	var aclList []*resource.AgentAcl
	if err := dbhandler.ListWithTx(&aclList, tx); err != nil {
		return fmt.Errorf("rewriteNamedAclFile failed:%s", err.Error())
	}

	for _, acl := range aclList {
		if acl.ID != anyACL && acl.ID != noneACL {
			namedAcl.Acls = append(namedAcl.Acls, ACL{Name: acl.Name, Ips: acl.Ips})
		}
	}

	if err := handler.flushTemplateFiles(namedAclTpl,
		handler.namedAclPath, namedAcl); err != nil {
		return fmt.Errorf("flushTemplateFiles failed :%s", err.Error())
	}

	return handler.rndcReconfig()
}

func (handler *DNSHandler) initZoneFiles(tx restdb.Transaction) error {
	if err := removeFiles(handler.dnsConfPath, "", zoneSuffix); err != nil {
		return fmt.Errorf("remvoe files for %s*.zone fail", handler.dnsConfPath)
	}

	var zoneList []*resource.AgentZone
	if err := dbhandler.ListWithTx(&zoneList, tx); err != nil {
		return err
	}

	var rrList []*resource.AgentRr
	if err := dbhandler.ListByConditionWithTx(&rrList,
		map[string]interface{}{"orderby": "name"}, tx); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	for _, zone := range zoneList {
		oneZone := zone.ToZoneFileData()
		for _, rr := range rrList {
			if rr.Zone == zone.ID {
				rdata, err := rr.ToRRData(zone.Name)
				if err != nil {
					return err
				}
				oneZone.RRs = append(oneZone.RRs, rdata)
			}
		}

		if err := handler.rewriteFiles(zoneTpl,
			filepath.Join(handler.dnsConfPath, oneZone.ZoneFile), oneZone, buf); err != nil {
			return err
		}
	}

	return nil
}

func (handler *DNSHandler) createZoneFile(zone *resource.AgentZone) error {
	return handler.flushTemplateFiles(zoneTpl,
		filepath.Join(handler.dnsConfPath, zone.ZoneFile), zone.ToZoneFileData())
}

func (handler *DNSHandler) rewriteOneZoneFile(zoneId, zoneFile string, tx restdb.Transaction) error {
	zoneRes, err := dbhandler.GetWithTx(zoneId, &[]*resource.AgentZone{}, tx)
	if err != nil {
		return err
	}
	var rrList []*resource.AgentRr
	if err := dbhandler.ListByConditionWithTx(&rrList,
		map[string]interface{}{"zone": zoneId, "orderby": "name"}, tx); err != nil {
		return err
	}
	zone := zoneRes.(*resource.AgentZone)
	oneZone := zone.ToZoneFileData()
	for _, rr := range rrList {
		rdata, err := rr.ToRRData(zone.Name)
		if err != nil {
			return err
		}
		oneZone.RRs = append(oneZone.RRs, rdata)
	}

	if err := removeOneFile(filepath.Join(handler.dnsConfPath, zoneFile)); err != nil {
		return err
	}

	return handler.flushTemplateFiles(zoneTpl,
		filepath.Join(handler.dnsConfPath, zoneFile), oneZone)
}

func (handler *DNSHandler) rewriteNzfsFile(tx restdb.Transaction) error {
	var zoneList []*resource.AgentZone
	if err := dbhandler.ListByConditionWithTx(&zoneList,
		map[string]interface{}{"orderby": "agent_view"}, tx); err != nil {
		return err
	}

	if err := removeFiles(handler.dnsConfPath, "", nzfSuffix); err != nil {
		return fmt.Errorf("remove files for %s*.zone fail", handler.dnsConfPath)
	}

	oneNzfMap := make(map[string][]resource.ZoneData)
	for _, zone := range zoneList {
		oneNzfMap[zone.AgentView] = append(oneNzfMap[zone.AgentView], zone.ToZoneData())
	}

	buf := new(bytes.Buffer)
	for view, zones := range oneNzfMap {
		if err := handler.rewriteFiles(nzfTpl,
			filepath.Join(handler.dnsConfPath, view)+nzfSuffix,
			nzfData{
				ViewName: view,
				Zones:    zones,
			}, buf); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) initRedirectFile(tx restdb.Transaction) error {
	if err := removeFiles(
		filepath.Join(handler.dnsConfPath, "redirection"), "redirect_", ""); err != nil {
		return fmt.Errorf("delete all the rpz file in %s err: %s",
			filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
	}

	return handler.getAllRedirectData(nxDomain, tx)
}

func (handler *DNSHandler) rewriteOneRedirectFile(viewID string, tx restdb.Transaction) error {
	if err := removeOneFile(
		filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+viewID)); err != nil {
		return err
	}

	if err := handler.getOneRedirectData(viewID, nxDomain, tx); err != nil {
		return err
	}

	if err := handler.rewriteNamedViewFile(false, tx); err != nil {
		return fmt.Errorf("rewriteRedirectFile rewriteNamedViewFile failed:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) initRPZFile(tx restdb.Transaction) error {
	if err := removeFiles(
		filepath.Join(handler.dnsConfPath, "redirection"), "rpz_", ""); err != nil {
		return fmt.Errorf("delete all the rpz file in %s err: %s",
			filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
	}

	return handler.getAllRedirectData(localZoneType, tx)
}

func (handler *DNSHandler) rewriteOneRPZFile(viewID string, tx restdb.Transaction) error {
	if err := removeOneFile(
		filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+viewID)); err != nil {
		return err
	}

	if err := handler.rewriteNamedViewFile(true, tx); err != nil {
		return fmt.Errorf("rewriteRPZFile rewriteNamedViewFile failed:%s", err.Error())
	}

	if err := handler.getOneRedirectData(viewID, localZoneType, tx); err != nil {
		return err
	}

	if err := handler.rewriteNamedViewFile(false, tx); err != nil {
		return fmt.Errorf("rewriteRPZFile rewriteNamedViewFile failed:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) getAllRedirectData(redirectType string, tx restdb.Transaction) error {
	var viewList []*resource.AgentView
	if err := dbhandler.ListWithTx(&viewList, tx); err != nil {
		return err
	}

	var redirectList []*resource.AgentRedirection
	if err := dbhandler.ListByConditionWithTx(&redirectList, map[string]interface{}{
		"orderby":       "create_time",
		"redirect_type": redirectType,
	}, tx); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	for _, view := range viewList {
		oneRedirectionData := RedirectionData{ViewName: view.Name}
		for _, redirection := range redirectList {
			if redirection.AgentView == view.ID {
				oneRedirectionData.RRs = append(oneRedirectionData.RRs, redirection.ToRRData())
			}
		}

		if len(oneRedirectionData.RRs) > 0 {
			if redirectType == localZoneType {
				if err := handler.flushTemplateFiles(rpzTpl,
					filepath.Join(handler.dnsConfPath,
						"redirection", "rpz_"+oneRedirectionData.ViewName),
					oneRedirectionData); err != nil {
					return err
				}
			} else {
				if err := handler.rewriteFiles(redirectTpl,
					filepath.Join(handler.dnsConfPath,
						"redirection", "redirect_"+oneRedirectionData.ViewName),
					oneRedirectionData, buf); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (handler *DNSHandler) getOneRedirectData(viewID, redirectType string, tx restdb.Transaction) error {
	viewRes, err := dbhandler.GetWithTx(viewID, &[]*resource.AgentView{}, tx)
	if err != nil {
		return err
	}
	view := viewRes.(*resource.AgentView)
	var redirectList []*resource.AgentRedirection
	if err := dbhandler.ListByConditionWithTx(&redirectList, map[string]interface{}{
		"agent_view":    view.ID,
		"orderby":       "create_time",
		"redirect_type": redirectType,
	}, tx); err != nil {
		return err
	}
	if len(redirectList) == 0 {
		return nil
	}

	oneRedirectionData := RedirectionData{ViewName: view.Name}
	for _, redirection := range redirectList {
		oneRedirectionData.RRs = append(oneRedirectionData.RRs, redirection.ToRRData())
	}

	if redirectType == localZoneType {
		return handler.flushTemplateFiles(rpzTpl,
			filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+oneRedirectionData.ViewName),
			oneRedirectionData)
	}

	return handler.flushTemplateFiles(redirectTpl,
		filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+oneRedirectionData.ViewName),
		oneRedirectionData)
}

func (handler *DNSHandler) flushTemplateFiles(tplName, tplConfName string, data interface{}) error {
	buffer := new(bytes.Buffer)
	return handler.rewriteFiles(tplName, tplConfName, data, buffer)
}

func (handler *DNSHandler) rewriteFiles(tplName, tplConfName string, data interface{}, buffer *bytes.Buffer) error {
	buffer.Reset()
	if err := handler.tpl.ExecuteTemplate(buffer, tplName, data); err != nil {
		return fmt.Errorf("flushTemplateFiles tplName:%s tplConfName:%s  failed:%s",
			tplName, tplConfName, err.Error())
	}

	if err := ioutil.WriteFile(tplConfName, buffer.Bytes(), FilePermissions); err != nil {
		return fmt.Errorf("flushTemplateFiles tplName:%s tplConfName:%s  WriteFile failed:%s",
			tplName, tplConfName, err.Error())
	}

	return nil
}

func removeFiles(dir string, prefix string, suffix string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		if prefix != "" && strings.HasPrefix(name, prefix) {
			err = os.RemoveAll(filepath.Join(dir, name))
			if err != nil {
				return err
			}
		}
		if suffix != "" && strings.HasSuffix(name, suffix) {
			err = os.RemoveAll(filepath.Join(dir, name))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func removeFolder(path string) error {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("removeFolder %s failed:%s ", path, err.Error())
		}
	}
	return nil
}

func removeOneFile(path string) error {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("removeOneFile %s failed:%s ", path, err.Error())
		}
	}
	return nil
}

func createOneFolder(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.Mkdir(path, FilePermissions); err != nil {
			return fmt.Errorf("create %s fail:%s", path, err.Error())
		}
	}
	return nil
}
