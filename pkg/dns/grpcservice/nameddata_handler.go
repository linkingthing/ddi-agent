package grpcservice

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/zdnscloud/cement/shell"
	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/ddi-agent/pkg/db"
	"github.com/linkingthing/ddi-agent/pkg/dns/dbhandler"
	"github.com/linkingthing/ddi-agent/pkg/dns/resource"
)

type NamedData struct {
	ConfigPath       string
	NamedAclPath     string
	NamedViewPath    string
	NamedOptionsPath string
}

type NamedOption struct {
	ConfigPath   string
	DnssecEnable bool
	LogEnable    bool
	IPBlackHole  *ipBlackHole
	SortList     []string
	Concu        *recursiveConcurrent
}

type NamedViews struct {
	Views []View
}

type NamedAcl struct {
	ConfigPath string
	Acls       []ACL
}

type ipBlackHole struct {
	ACLNames []string
}

type View struct {
	Name     string
	ACLs     []ACL
	Zones    []Zone
	Redirect *Redirect
	RPZ      *Rpz
	DNS64s   []Dns64
	Key      string
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
	RRs []RR
}

type Redirect struct {
	RRs []RR
}

type RedirectionData struct {
	ViewName string
	RRs      []RR
}

type RR struct {
	Name  string
	Type  string
	Value string
	TTL   string
}

type nzfData struct {
	ViewName string
	Zones    []Zone
}

type Zone struct {
	Name        string
	ZoneFile    string
	ForwardType string
	IPs         []string
}

type ZoneFileData struct {
	ViewName string
	Name     string
	ZoneFile string
	TTL      string
	RRs      []RR
}

type recursiveConcurrent struct {
	RecursiveClients *uint32
	FetchesPerZone   *uint32
}

type nginxDefaultConf struct {
	URLRedirects []urlRedirect
}

type urlRedirect struct {
	Domain string
	URL    string
}

func (handler *DNSHandler) initFiles() error {
	handler.rndcConfPath = filepath.Join(handler.dnsConfPath, "rndc.conf")
	handler.rndcPath = filepath.Join(handler.dnsConfPath, "rndc")
	handler.nginxConfPath = filepath.Join(handler.nginxDefaultConfDir, nginxDefaultConfFile)
	handler.namedViewPath = filepath.Join(handler.dnsConfPath, namedViewConfName)
	handler.namedOptionPath = filepath.Join(handler.dnsConfPath, namedOptionsConfName)
	handler.namedAclPath = filepath.Join(handler.dnsConfPath, namedAclConfName)
	if handler.rndcConfPath == "" && handler.rndcPath == "" && handler.nginxConfPath == "" &&
		handler.namedViewPath == "" && handler.namedOptionPath == "" && handler.namedAclPath == "" {
		return fmt.Errorf("init path failed path is empty")
	}

	if err := createOneFile(filepath.Join(handler.dnsConfPath, "redirection")); err != nil {
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
		if err := handler.rewriteNginxFile(tx); err != nil {
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
	paras := []string{
		"-c" + handler.rndcConfPath,
		"-s" + "localhost",
		"-p" + rndcPort,
		"reconfig",
	}

	if _, err := shell.Shell(handler.rndcPath, paras...); err != nil {
		return fmt.Errorf("rndc reconfig error, %w", err)
	}
	return nil
}

func (handler *DNSHandler) rndcReload() error {
	paras := []string{
		"-c" + handler.rndcConfPath,
		"-s" + "localhost",
		"-p" + rndcPort,
		"reload",
	}

	if _, err := shell.Shell(handler.rndcPath, paras...); err != nil {
		return fmt.Errorf("rndc reconfig error, %w", err)
	}
	return nil
}

func (handler *DNSHandler) rndcAddZone(name string, zoneFile string, viewName string) error {
	paras := []string{
		"-c" + handler.rndcConfPath,
		"-s" + "localhost",
		"-p" + rndcPort,
		"addzone " + name + " in " + viewName + " { type master; file \"" + zoneFile + "\";};",
	}
	if out, err := shell.Shell(handler.rndcPath, paras...); err != nil {
		return fmt.Errorf("rndcAddZone error:%s cmd:%s error:%s", out, paras, err.Error())
	}
	return nil
}

func (handler *DNSHandler) rndcModifyZone(name string, zoneFile string, viewName string) error {
	paras := []string{
		"-c" + handler.rndcConfPath,
		"-s" + "localhost",
		"-p" + rndcPort,
		"modzone " + name + " in " + viewName + " { type master; file \"" + zoneFile + "\";};",
	}
	if out, err := shell.Shell(handler.rndcPath, paras...); err != nil {
		return fmt.Errorf("rndcModifyZone error:%s cmd:%s error:%s", out, paras, err.Error())
	}
	return nil
}

func (handler *DNSHandler) rndcDeleteZone(name string, viewName string) error {
	paras := []string{
		"-c" + handler.rndcConfPath,
		"-s" + "localhost",
		"-p" + rndcPort,
		"delzone",
		"-clean",
		name + " in " + viewName,
	}
	if out, err := shell.Shell(handler.rndcPath, paras...); err != nil {
		return fmt.Errorf("rndcDeleteZone error:%s cmd:%s error:%s", out, paras, err.Error())
	}
	return nil
}

func (handler *DNSHandler) rndcDumpJNLFile() error {
	paras := []string{
		"-c" + handler.rndcConfPath,
		"-s" + "localhost",
		"-p" + rndcPort,
		"sync",
		"-clean",
	}
	if _, err := shell.Shell(handler.rndcPath, paras...); err != nil {
		return fmt.Errorf("exec rndc sync -clean err:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) rndcZoneDumpJNLFile(zoneName string, viewName string) error {
	paras := []string{
		"-c" + handler.rndcConfPath,
		"-s" + "localhost",
		"-p" + rndcPort,
		"sync",
		"-clean",
		zoneName + " in " + viewName,
	}

	if _, err := shell.Shell(handler.rndcPath, paras...); err != nil {
		return fmt.Errorf("exec rndc sync -clean err:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) rewriteNginxFile(tx restdb.Transaction) error {
	data, err := handler.GetNginxData(tx)
	if err != nil {
		return fmt.Errorf("get nginx conf data error %s", err.Error())
	}
	buf := new(bytes.Buffer)
	if err = handler.tpl.ExecuteTemplate(buf, nginxDefaultTpl, data); err != nil {
		return err
	}
	if err := ioutil.WriteFile(handler.nginxConfPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file %s err:%s", handler.nginxConfPath, err.Error())
	}
	return nil
}

func (handler *DNSHandler) GetNginxData(tx restdb.Transaction) (*nginxDefaultConf, error) {
	data := nginxDefaultConf{}
	var urlRedirectList []*resource.AgentUrlRedirect
	if err := dbhandler.ListWithTx(&urlRedirectList, tx); err != nil {
		return nil, err
	}
	for _, urlValue := range urlRedirectList {
		data.URLRedirects = append(data.URLRedirects, urlRedirect{Domain: urlValue.Domain, URL: urlValue.Url})
	}

	return &data, nil
}

func (handler *DNSHandler) nginxReload() error {
	command := "docker exec -i ddi-nginx nginx -s reload"
	cmd := exec.Command("/bin/bash", "-c", command)
	if _, err := cmd.Output(); err != nil {
		return fmt.Errorf("exec docker nginx reload error: %s", err.Error())
	}
	return nil
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
	if err := dbhandler.ListByCondition(&forwardZoneList,
		map[string]interface{}{}); err != nil {
		return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
	}

	for _, value := range viewList {
		var acls []ACL
		for _, aclValue := range value.Acls {
			acls = append(acls, ACL{Name: aclValue})
		}
		view := View{Name: value.Name, ACLs: acls, Key: value.Key}

		for _, reValue := range redirectionList {
			if reValue.AgentView == value.ID {
				var redirectRR []RR
				var rpzRR []RR
				rr := RR{}
				rr.Name = reValue.Name
				rr.TTL = strconv.Itoa(int(reValue.Ttl))
				rr.Type = reValue.DataType
				rr.Value = reValue.Rdata
				if reValue.RedirectType == localZoneType {
					rpzRR = append(rpzRR, rr)
				} else if reValue.RedirectType == nxDomain {
					redirectRR = append(redirectRR, rr)
				}

				if len(redirectRR) > 0 {
					view.Redirect = &Redirect{redirectRR}
				}
				if len(rpzRR) > 0 {
					view.RPZ = &Rpz{rpzRR}
				}
			}
		}

		if value.Dns64 != "" {
			var dns64 Dns64
			dns64.Prefix = value.Dns64
			dns64.AAddressACLName = anyACL
			dns64.ClientACLName = anyACL
			view.DNS64s = append(view.DNS64s, dns64)
		}

		for _, forwardZone := range forwardZoneList {
			if forwardZone.AgentView == value.ID {
				var tmp Zone
				tmp.Name = forwardZone.Name
				tmp.ForwardType = forwardZone.ForwardType
				tmp.IPs = append(tmp.IPs, forwardZone.Ips...)
				view.Zones = append(view.Zones, tmp)
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
			acls = append(acls, ACL{Name: aclValue})
		}
		view := View{Name: value.Name, ACLs: acls, Key: value.Key}

		for _, reValue := range redirectionList {
			if reValue.AgentView == value.ID {
				var redirectRR []RR
				var rpzRR []RR
				rr := RR{}
				rr.Name = reValue.Name
				rr.TTL = strconv.Itoa(int(reValue.Ttl))
				rr.Type = reValue.DataType
				rr.Value = reValue.Rdata
				if reValue.RedirectType == localZoneType {
					rpzRR = append(rpzRR, rr)
				} else if reValue.RedirectType == nxDomain {
					redirectRR = append(redirectRR, rr)
				}

				if len(redirectRR) > 0 {
					view.Redirect = &Redirect{redirectRR}
				}
				if !existRPZ && len(rpzRR) > 0 {
					view.RPZ = &Rpz{rpzRR}
				}
			}
		}

		if value.Dns64 != "" {
			var dns64 Dns64
			dns64.Prefix = value.Dns64
			dns64.AAddressACLName = anyACL
			dns64.ClientACLName = anyACL
			view.DNS64s = append(view.DNS64s, dns64)
		}

		for _, forwardZone := range forwardZoneList {
			if forwardZone.AgentView == value.ID {
				var tmp Zone
				tmp.Name = forwardZone.Name
				tmp.ForwardType = forwardZone.ForwardType
				tmp.IPs = append(tmp.IPs, forwardZone.Ips...)
				view.Zones = append(view.Zones, tmp)
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
	globalConfigRes, err := dbhandler.GetWithTx(defaultGlobalConfigID, &[]*resource.AgentDnsGlobalConfig{}, tx)
	if err != nil {
		return err
	}
	globalConfig := globalConfigRes.(*resource.AgentDnsGlobalConfig)
	namedOptionData.LogEnable = globalConfig.LogEnable
	namedOptionData.DnssecEnable = globalConfig.DnssecEnable

	var ipBlackHoleList []*resource.AgentIpBlackHole
	if err := dbhandler.ListWithTx(&ipBlackHoleList, tx); err != nil {
		return err
	}
	for _, ipBlack := range ipBlackHoleList {
		namedOptionData.IPBlackHole = &ipBlackHole{}
		namedOptionData.IPBlackHole.ACLNames = append(namedOptionData.IPBlackHole.ACLNames, ipBlack.Acl)
	}

	exist, err := dbhandler.ExistWithTx(resource.TableRecursiveConcurrent, defaultRecursiveConcurrentId, tx)
	if err != nil {
		return err
	}
	if exist {
		recursiveConcurrentRes, err := dbhandler.GetWithTx(defaultRecursiveConcurrentId, &[]*resource.AgentRecursiveConcurrent{}, tx)
		if err != nil {
			return err
		}
		recursiveCon := recursiveConcurrentRes.(*resource.AgentRecursiveConcurrent)
		namedOptionData.Concu = &recursiveConcurrent{
			RecursiveClients: &recursiveCon.RecursiveClients,
			FetchesPerZone:   &recursiveCon.FetchesPerZone,
		}
	}

	if err := handler.flushTemplateFiles(namedOptionsTpl,
		handler.namedOptionPath, namedOptionData); err != nil {
		return fmt.Errorf("flushTemplateFiles failed :%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) rewriteNamedOptionsFile(tx restdb.Transaction) error {
	namedOptionData := &NamedOption{ConfigPath: handler.dnsConfPath}

	globalConfigRes, err := dbhandler.GetWithTx(defaultGlobalConfigID, &[]*resource.AgentDnsGlobalConfig{}, tx)
	if err != nil {
		return err
	}
	globalConfig := globalConfigRes.(*resource.AgentDnsGlobalConfig)
	namedOptionData.LogEnable = globalConfig.LogEnable
	namedOptionData.DnssecEnable = globalConfig.DnssecEnable

	var ipBlackHoleList []*resource.AgentIpBlackHole
	if err := dbhandler.ListWithTx(&ipBlackHoleList, tx); err != nil {
		return err
	}
	for _, ipBlack := range ipBlackHoleList {
		namedOptionData.IPBlackHole = &ipBlackHole{}
		namedOptionData.IPBlackHole.ACLNames = append(namedOptionData.IPBlackHole.ACLNames, ipBlack.Acl)
	}

	exist, err := dbhandler.ExistWithTx(resource.TableRecursiveConcurrent, defaultRecursiveConcurrentId, tx)
	if err != nil {
		return err
	}
	if exist {
		recursiveConcurrentRes, err := dbhandler.GetWithTx(defaultRecursiveConcurrentId, &[]*resource.AgentRecursiveConcurrent{}, tx)
		if err != nil {
			return err
		}
		recursiveCon := recursiveConcurrentRes.(*resource.AgentRecursiveConcurrent)
		namedOptionData.Concu = &recursiveConcurrent{
			RecursiveClients: &recursiveCon.RecursiveClients,
			FetchesPerZone:   &recursiveCon.FetchesPerZone,
		}
	}

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
			oneAcl := ACL{Name: acl.Name, Ips: acl.Ips}
			namedAcl.Acls = append(namedAcl.Acls, oneAcl)
		}
	}

	if err := handler.flushTemplateFiles(namedAclTpl,
		handler.namedAclPath, namedAcl); err != nil {
		return fmt.Errorf("flushTemplateFiles failed :%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) rewriteNamedAclFile(isInit bool, tx restdb.Transaction) error {
	namedAcl := &NamedAcl{ConfigPath: handler.dnsConfPath}
	var aclList []*resource.AgentAcl
	if err := dbhandler.ListWithTx(&aclList, tx); err != nil {
		return fmt.Errorf("rewriteNamedAclFile failed:%s", err.Error())
	}

	for _, acl := range aclList {
		if acl.ID != anyACL && acl.ID != noneACL {
			oneAcl := ACL{Name: acl.Name, Ips: acl.Ips}
			namedAcl.Acls = append(namedAcl.Acls, oneAcl)
		}
	}

	if err := handler.flushTemplateFiles(namedAclTpl,
		handler.namedAclPath, namedAcl); err != nil {
		return fmt.Errorf("flushTemplateFiles failed :%s", err.Error())
	}

	if isInit {
		return nil
	}

	return handler.rndcReconfig()
}

func (handler *DNSHandler) initZoneFiles(tx restdb.Transaction) error {
	var zonesData []ZoneFileData
	var zoneList []*resource.AgentZone
	if err := dbhandler.ListWithTx(&zoneList, tx); err != nil {
		return err
	}

	var rrList []*resource.AgentRr
	if err := dbhandler.ListByCondition(&rrList,
		map[string]interface{}{"orderby": "name"}); err != nil {
		return err
	}
	for _, zone := range zoneList {
		oneZone := ZoneFileData{ViewName: zone.AgentView, Name: zone.Name, ZoneFile: zone.ZoneFile, TTL: strconv.FormatUint(uint64(zone.Ttl), 10)}
		for _, rr := range rrList {
			if rr.Zone == zone.ID {
				oneRR := RR{Name: rr.Name, Type: rr.DataType, Value: rr.Rdata, TTL: strconv.FormatUint(uint64(rr.Ttl), 10)}
				oneZone.RRs = append(oneZone.RRs, oneRR)
			}
		}
		zonesData = append(zonesData, oneZone)
	}

	if err := removeFiles(handler.dnsConfPath, "", zoneSuffix); err != nil {
		return fmt.Errorf("remvoe files for %s*.zone fail", handler.dnsConfPath)
	}

	buf := new(bytes.Buffer)
	for _, z := range zonesData {
		buf.Reset()
		if err := handler.tpl.ExecuteTemplate(buf, zoneTpl, z); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, z.ZoneFile), buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (handler *DNSHandler) createZoneFile(zone *resource.AgentZone) error {
	oneZone := ZoneFileData{ViewName: zone.AgentView, Name: zone.Name,
		ZoneFile: zone.ZoneFile, TTL: strconv.FormatUint(uint64(zone.Ttl), 10)}

	buf := new(bytes.Buffer)
	if err := handler.tpl.ExecuteTemplate(buf, zoneTpl, oneZone); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, zone.ZoneFile), buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
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
	oneZone := ZoneFileData{ViewName: zone.AgentView, Name: zone.Name,
		ZoneFile: zone.ZoneFile, TTL: strconv.FormatUint(uint64(zone.Ttl), 10)}
	for _, rr := range rrList {
		oneRR := RR{Name: rr.Name, Type: rr.DataType, Value: rr.Rdata, TTL: strconv.FormatUint(uint64(rr.Ttl), 10)}
		oneZone.RRs = append(oneZone.RRs, oneRR)
	}

	if err := removeOneFile(filepath.Join(handler.dnsConfPath, zoneFile)); err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	if err := handler.tpl.ExecuteTemplate(buf, zoneTpl, oneZone); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, zoneFile), buf.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rewriteNzfsFile(tx restdb.Transaction) error {
	var nzfsData []nzfData
	var zoneList []*resource.AgentZone
	if err := dbhandler.ListWithTx(&zoneList, tx); err != nil {
		return err
	}

	for _, zone := range zoneList {
		oneNzfData := nzfData{ViewName: zone.AgentView}
		if zone.ZoneFile != "" {
			oneZone := Zone{Name: zone.Name, ZoneFile: zone.ZoneFile}
			oneNzfData.Zones = append(oneNzfData.Zones, oneZone)
		}
		nzfsData = append(nzfsData, oneNzfData)
	}

	if err := removeFiles(handler.dnsConfPath, "", nzfSuffix); err != nil {
		return fmt.Errorf("remvoe files for %s*.zone fail", handler.dnsConfPath)
	}
	for _, nzfData := range nzfsData {
		buf := new(bytes.Buffer)
		if err := handler.tpl.ExecuteTemplate(buf, nzfTpl, nzfData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, nzfData.ViewName)+nzfSuffix, buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) nzfsData(tx restdb.Transaction) ([]nzfData, error) {
	var data []nzfData
	var zoneList []*resource.AgentZone
	if err := dbhandler.ListWithTx(&zoneList, tx); err != nil {
		return nil, err
	}

	for _, zone := range zoneList {
		oneNzfData := nzfData{ViewName: zone.AgentView}
		if zone.ZoneFile != "" {
			oneZone := Zone{Name: zone.Name, ZoneFile: zone.ZoneFile}
			oneNzfData.Zones = append(oneNzfData.Zones, oneZone)
		}
		data = append(data, oneNzfData)
	}

	return data, nil
}

func (handler *DNSHandler) initRedirectFile(tx restdb.Transaction) error {
	redirectionList, err := handler.getAllRedirectData(nxDomain, tx)
	if err != nil {
		return err
	}
	if err := removeFiles(filepath.Join(handler.dnsConfPath, "redirection"), "redirect_", ""); err != nil {
		return fmt.Errorf("delete all the rpz file in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
	}

	buf := new(bytes.Buffer)
	for _, redirectionData := range redirectionList {
		buf.Reset()
		if err = handler.tpl.ExecuteTemplate(buf, redirectTpl, redirectionData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+redirectionData.ViewName), buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (handler *DNSHandler) rewriteOneRedirectFile(viewID string, tx restdb.Transaction) error {
	if err := removeOneFile(filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+viewID)); err != nil {
		return err
	}

	redirectionList, err := handler.getOneRedirectData(viewID, nxDomain, tx)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	for _, redirectionData := range redirectionList {
		buf.Reset()
		if err = handler.tpl.ExecuteTemplate(buf, redirectTpl, redirectionData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+redirectionData.ViewName), buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	if err := handler.rewriteNamedViewFile(false, tx); err != nil {
		return fmt.Errorf("rewriteRedirectFile rewriteRedirectFile failed:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) initRPZFile(tx restdb.Transaction) error {
	redirectionList, err := handler.getAllRedirectData(localZoneType, tx)
	if err != nil {
		return err
	}

	if err := removeFiles(filepath.Join(handler.dnsConfPath, "redirection"), "rpz_", ""); err != nil {
		return fmt.Errorf("delete all the rpz file in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
	}

	buf := new(bytes.Buffer)
	for _, redirectionData := range redirectionList {
		buf.Reset()
		if err = handler.tpl.ExecuteTemplate(buf, rpzTpl, redirectionData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+redirectionData.ViewName), buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) rewriteOneRPZFile(viewID string, tx restdb.Transaction) error {
	if err := removeOneFile(filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+viewID)); err != nil {
		return err
	}

	redirectionList, err := handler.getOneRedirectData(viewID, localZoneType, tx)
	if err != nil {
		return err
	}

	if err := handler.rewriteNamedViewFile(true, tx); err != nil {
		return fmt.Errorf("rewriteRPZFile rewriteNamedViewFile failed:%s", err.Error())
	}

	buf := new(bytes.Buffer)
	for _, redirectionData := range redirectionList {
		buf.Reset()
		if err = handler.tpl.ExecuteTemplate(buf, rpzTpl, redirectionData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+redirectionData.ViewName), buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	if err := handler.rewriteNamedViewFile(false, tx); err != nil {
		return fmt.Errorf("rewriteRPZFile rewriteNamedViewFile failed:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) getAllRedirectData(redirectType string, tx restdb.Transaction) ([]RedirectionData, error) {
	var data []RedirectionData
	var viewList []*resource.AgentView
	if err := dbhandler.ListWithTx(&viewList, tx); err != nil {
		return nil, err
	}

	cond := map[string]interface{}{
		"orderby": "create_time",
	}
	if redirectType != "" {
		cond["redirect_type"] = redirectType
	}
	var redirectList []*resource.AgentRedirection
	if err := dbhandler.ListByConditionWithTx(&redirectList, cond, tx); err != nil {
		return nil, err
	}
	for _, view := range viewList {
		oneRedirectionData := RedirectionData{ViewName: view.Name}
		for _, redirection := range redirectList {
			if redirection.AgentView == view.ID {
				oneRR := RR{Name: redirection.Name, Type: redirection.DataType, Value: redirection.Rdata, TTL: strconv.FormatUint(uint64(redirection.Ttl), 10)}
				oneRedirectionData.RRs = append(oneRedirectionData.RRs, oneRR)
			}
		}
		if len(oneRedirectionData.RRs) > 0 {
			data = append(data, oneRedirectionData)
		}
	}

	return data, nil
}

func (handler *DNSHandler) getOneRedirectData(viewID, redirectType string, tx restdb.Transaction) ([]RedirectionData, error) {
	var data []RedirectionData
	viewRes, err := dbhandler.GetWithTx(viewID, &[]*resource.AgentView{}, tx)
	if err != nil {
		return nil, err
	}
	view := viewRes.(*resource.AgentView)
	cond := map[string]interface{}{
		"agent_view": view.ID,
		"orderby":    "create_time",
	}
	if redirectType != "" {
		cond["redirect_type"] = redirectType
	}
	var redirectList []*resource.AgentRedirection
	if err := dbhandler.ListByConditionWithTx(&redirectList, cond, tx); err != nil {
		return nil, err
	}
	if len(redirectList) == 0 {
		return data, nil
	}
	oneRedirectionData := RedirectionData{ViewName: view.Name}
	for _, redirection := range redirectList {
		oneRR := RR{Name: redirection.Name, Type: redirection.DataType, Value: redirection.Rdata, TTL: strconv.FormatUint(uint64(redirection.Ttl), 10)}
		oneRedirectionData.RRs = append(oneRedirectionData.RRs, oneRR)
	}
	data = append(data, oneRedirectionData)

	return data, nil
}

func (handler *DNSHandler) flushTemplateFiles(tplName, tplConfName string, data interface{}) error {
	buffer := new(bytes.Buffer)
	if err := handler.tpl.ExecuteTemplate(buffer, tplName, data); err != nil {
		return fmt.Errorf("flushTemplateFiles tplName:%s tplConfName:%s  failed:%s", tplName, tplConfName, err.Error())
	}

	if err := ioutil.WriteFile(tplConfName, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("flushTemplateFiles tplName:%s tplConfName:%s  WriteFile failed:%s", tplName, tplConfName, err.Error())
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

func removeOneFile(path string) error {
	_, err := os.Stat(path)
	if !os.IsNotExist(err) {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("Remove %s failed:%s ", path, err.Error())
		}
	}
	return nil
}

func createOneFile(path string) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("create %s fail:%s", path, err.Error())
		}
	}
	return nil
}
