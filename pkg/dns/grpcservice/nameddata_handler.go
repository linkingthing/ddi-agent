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

	"github.com/linkingthing/ddi-agent/pkg/db"
	restdb "github.com/zdnscloud/gorest/db"

	"github.com/zdnscloud/cement/shell"

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

func (handler *DNSHandler) initNamedConf() error {
	data := &NamedData{
		NamedAclPath:     filepath.Join(handler.dnsConfPath, namedAclConfName),
		NamedViewPath:    filepath.Join(handler.dnsConfPath, namedViewConfName),
		NamedOptionsPath: filepath.Join(handler.dnsConfPath, namedOptionsConfName),
	}
	return handler.FlushTemplateFiles(namedTpl, filepath.Join(handler.dnsConfPath, mainConfName), data)
}

func (handler *DNSHandler) rndcReconfig() error {
	paras := []string{
		"-c" + filepath.Join(handler.dnsConfPath, "rndc.conf"),
		"-s" + "localhost",
		"-p" + rndcPort,
		"reconfig",
	}

	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), paras...); err != nil {
		return fmt.Errorf("rndc reconfig error, %w", err)
	}
	return nil
}

func (handler *DNSHandler) rndcReload() error {
	paras := []string{
		"-c" + filepath.Join(handler.dnsConfPath, "rndc.conf"),
		"-s" + "localhost",
		"-p" + rndcPort,
		"reload",
	}

	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), paras...); err != nil {
		return fmt.Errorf("rndc reconfig error, %w", err)
	}
	return nil
}

func (handler *DNSHandler) rndcAddZone(name string, zoneFile string, viewName string) error {
	paras := []string{
		"-c" + filepath.Join(handler.dnsConfPath, "rndc.conf"),
		"-s" + "localhost",
		"-p" + rndcPort,
		"addzone " + name + " in " + viewName + " { type master; file \"" + zoneFile + "\";};",
	}
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), paras...); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rndcDelZone(name string, viewName string) error {
	paras := []string{
		"-c" + filepath.Join(handler.dnsConfPath, "rndc.conf"),
		"-s" + "localhost",
		"-p" + rndcPort,
		"delzone " + name + " in " + viewName,
	}
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), paras...); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rndcDumpJNLFile() error {
	paras := []string{
		"-c" + filepath.Join(handler.dnsConfPath, "rndc.conf"),
		"-s" + "localhost",
		"-p" + rndcPort,
		"sync",
		"-clean",
	}
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), paras...); err != nil {
		return fmt.Errorf("exec rndc sync -clean err:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) rewriteNginxFile() error {
	data, err := handler.GetNginxData()
	if err != nil {
		return fmt.Errorf("get nginx conf data error %s", err.Error())
	}
	buf := new(bytes.Buffer)
	if err = handler.tpl.ExecuteTemplate(buf, nginxDefaultTpl, data); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(handler.nginxDefaultConfDir, nginxDefaultConfFile), buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write file %s err:%s", filepath.Join(handler.nginxDefaultConfDir, nginxDefaultConfFile), err.Error())
	}
	return nil
}

func (handler *DNSHandler) GetNginxData() (*nginxDefaultConf, error) {
	data := nginxDefaultConf{}
	var urlRedirectList []*resource.AgentUrlRedirect
	if err := dbhandler.List(&urlRedirectList); err != nil {
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

func (handler *DNSHandler) rewriteNamedViewFile(isInit bool, existRPZ bool) error {
	viewConfigData := &NamedViews{}
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var viewList []*resource.AgentView

		if err := dbhandler.ListByConditionWithTx(&viewList,
			map[string]interface{}{"orderby": "priority"}, tx); err != nil {
			return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
		}

		for _, value := range viewList {
			var acls []ACL
			for _, aclValue := range value.Acls {
				acls = append(acls, ACL{Name: aclValue})
			}
			view := View{Name: value.Name, ACLs: acls, Key: value.Key}

			var redirectionList []*resource.AgentRedirection
			if err := dbhandler.ListByConditionWithTx(&redirectionList,
				map[string]interface{}{"view": value.ID}, tx); err != nil {
				return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
			}

			if len(redirectionList) > 0 {
				var redirectRR []RR
				var rpzRR []RR
				for _, reValue := range redirectionList {
					var rr RR
					rr.Name = reValue.Name
					rr.TTL = strconv.Itoa(int(reValue.Ttl))
					rr.Type = reValue.DataType
					rr.Value = reValue.Rdata
					if reValue.RedirectType == localZoneType {
						rpzRR = append(rpzRR, rr)
					} else if reValue.RedirectType == nxDomain {
						redirectRR = append(redirectRR, rr)
					}
				}

				if len(redirectRR) > 0 {
					view.Redirect = &Redirect{redirectRR}
				}
				if !existRPZ && len(rpzRR) > 0 {
					view.RPZ = &Rpz{rpzRR}
				}
			}

			if value.Dns64 != "" {
				var dns64 Dns64
				dns64.Prefix = value.Dns64
				dns64.AAddressACLName = anyACL
				dns64.ClientACLName = anyACL
				view.DNS64s = append(view.DNS64s, dns64)
			}

			var forwardZoneList []*resource.AgentForwardZone
			if err := dbhandler.ListByCondition(&forwardZoneList,
				map[string]interface{}{"view": value.ID}); err != nil {
				return fmt.Errorf("rewriteViewFile failed:%s", err.Error())
			}

			for _, forwardZone := range forwardZoneList {
				var tmp Zone
				tmp.Name = forwardZone.Name
				tmp.ForwardType = forwardZone.ForwardType
				tmp.IPs = append(tmp.IPs, forwardZone.Ips...)
				view.Zones = append(view.Zones, tmp)
			}

			viewConfigData.Views = append(viewConfigData.Views, view)
		}

		return nil
	}); err != nil {
		return err
	}

	if err := handler.FlushTemplateFiles(namedViewTpl,
		filepath.Join(handler.dnsConfPath, namedViewConfName), viewConfigData); err != nil {
		return fmt.Errorf("FlushTemplateFiles failed :%s", err.Error())
	}

	if isInit {
		return nil
	}

	return handler.rndcReconfig()
}

func (handler *DNSHandler) rewriteNamedOptionsFile(isInit bool) error {
	namedOptionData := &NamedOption{ConfigPath: handler.dnsConfPath}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		globalConfigRes, err := dbhandler.Get(defaultGlobalConfigID, &[]*resource.AgentDnsGlobalConfig{})
		if err != nil {
			return err
		}
		globalConfig := globalConfigRes.(*resource.AgentDnsGlobalConfig)
		namedOptionData.LogEnable = globalConfig.LogEnable
		namedOptionData.DnssecEnable = globalConfig.DnssecEnable

		var ipBlackHoleList []*resource.AgentIpBlackHole
		if err := dbhandler.List(&ipBlackHoleList); err != nil {
			return err
		}
		for _, ipBlack := range ipBlackHoleList {
			namedOptionData.IPBlackHole = &ipBlackHole{}
			namedOptionData.IPBlackHole.ACLNames = append(namedOptionData.IPBlackHole.ACLNames, ipBlack.Acl)
		}

		exist, err := dbhandler.Exist(resource.TableRecursiveConcurrent, defaultRecursiveConcurrentId)
		if err != nil {
			return err
		}
		if exist {
			recursiveConcurrentRes, err := dbhandler.Get(defaultRecursiveConcurrentId, &[]*resource.AgentRecursiveConcurrent{})
			if err != nil {
				return err
			}
			recursiveCon := recursiveConcurrentRes.(*resource.AgentRecursiveConcurrent)
			namedOptionData.Concu = &recursiveConcurrent{
				RecursiveClients: &recursiveCon.RecursiveClients,
				FetchesPerZone:   &recursiveCon.FetchesPerZone,
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if err := handler.FlushTemplateFiles(namedOptionsTpl,
		filepath.Join(handler.dnsConfPath, namedOptionsConfName), namedOptionData); err != nil {
		return fmt.Errorf("FlushTemplateFiles failed :%s", err.Error())
	}

	if isInit {
		return nil
	}

	return handler.rndcReconfig()

}

func (handler *DNSHandler) rewriteNamedAclFile(isInit bool) error {
	namedAcl := &NamedAcl{ConfigPath: handler.dnsConfPath}
	var aclList []*resource.AgentAcl
	if err := dbhandler.List(&aclList); err != nil {
		return fmt.Errorf("rewriteNamedAclFile failed:%s", err.Error())
	}

	for _, acl := range aclList {
		if acl.ID != anyACL && acl.ID != noneACL {
			oneAcl := ACL{Name: acl.Name, Ips: acl.Ips}
			namedAcl.Acls = append(namedAcl.Acls, oneAcl)
		}
	}

	if err := handler.FlushTemplateFiles(namedAclTpl,
		filepath.Join(handler.dnsConfPath, namedAclConfName), namedAcl); err != nil {
		return fmt.Errorf("FlushTemplateFiles failed :%s", err.Error())
	}

	if isInit {
		return nil
	}

	return handler.rndcReconfig()
}

func (handler *DNSHandler) rewriteZoneFile(zoneId string) error {
	var zonesData []ZoneFileData
	if zoneId != "" {
		zoneRes, err := dbhandler.Get(zoneId, &[]*resource.AgentZone{})
		if err != nil {
			return err
		}

		var rrList []*resource.AgentRr
		if err := dbhandler.ListByCondition(&rrList,
			map[string]interface{}{"zone": zoneId, "orderby": "name"}); err != nil {
			return err
		}
		zone := zoneRes.(*resource.AgentZone)

		oneZone := ZoneFileData{ViewName: zone.View, Name: zone.Name, ZoneFile: zone.ZoneFile, TTL: strconv.FormatUint(uint64(zone.Ttl), 10)}
		preName := ""
		for _, rr := range rrList {
			oneRR := RR{Type: rr.DataType, Value: rr.Rdata, TTL: strconv.FormatUint(uint64(rr.Ttl), 10)}
			if preName != rr.Name {
				oneRR.Name = rr.Name
			}
			preName = rr.Name
			oneZone.RRs = append(oneZone.RRs, oneRR)
		}
		zonesData = append(zonesData, oneZone)
	} else {
		var zoneList []*resource.AgentZone
		if err := dbhandler.List(&zoneList); err != nil {
			return err
		}

		for _, zone := range zoneList {
			var rrList []*resource.AgentRr
			if err := dbhandler.ListByCondition(&rrList,
				map[string]interface{}{"zone": zone.ID, "orderby": "name"}); err != nil {
				return err
			}
			oneZone := ZoneFileData{ViewName: zone.View, Name: zone.Name, ZoneFile: zone.ZoneFile, TTL: strconv.FormatUint(uint64(zone.Ttl), 10)}
			for _, rr := range rrList {
				oneRR := RR{Name: rr.Name, Type: rr.DataType, Value: rr.Rdata, TTL: strconv.FormatUint(uint64(rr.Ttl), 10)}
				oneZone.RRs = append(oneZone.RRs, oneRR)
			}
			zonesData = append(zonesData, oneZone)
		}

		if err := removeFiles(handler.dnsConfPath, "", zoneSuffix); err != nil {
			return fmt.Errorf("remvoe files for %s*.zone fail", handler.dnsConfPath)
		}
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

func (handler *DNSHandler) rewriteNzfsFile() error {
	nzfsData, err := handler.nzfsData()
	if err != nil {
		return err
	}
	if err := removeFiles(handler.dnsConfPath, "", nzfSuffix); err != nil {
		return fmt.Errorf("remvoe files for %s*.zone fail", handler.dnsConfPath)
	}
	for _, nzfData := range nzfsData {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, nzfTpl, nzfData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, nzfData.ViewName)+nzfSuffix, buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) nzfsData() ([]nzfData, error) {
	var data []nzfData
	var zoneList []*resource.AgentZone
	if err := dbhandler.List(&zoneList); err != nil {
		return nil, err
	}

	for _, zone := range zoneList {
		oneNzfData := nzfData{ViewName: zone.View}
		if zone.ZoneFile != "" {
			oneZone := Zone{Name: zone.Name, ZoneFile: zone.ZoneFile}
			oneNzfData.Zones = append(oneNzfData.Zones, oneZone)
		}
		data = append(data, oneNzfData)
	}

	return data, nil
}

func (handler *DNSHandler) rewriteRedirectFile(isStart bool, viewID string) error {
	redirectionList, err := handler.redirectData(viewID, nxDomain)
	if err != nil {
		return err
	}
	if viewID == "" {
		if err := removeFiles(filepath.Join(handler.dnsConfPath, "redirection"), "redirect_", ""); err != nil {
			return fmt.Errorf("delete all the rpz file in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}
	} else {
		path := filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+viewID)
		if err := PathExists(path); err == nil {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("delete %s file err: %s", path, err.Error())
			}
		}
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

	if err := handler.rewriteNamedViewFile(isStart, false); err != nil {
		return fmt.Errorf("rewriteRPZFile rewriteRedirectFile failed:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) rewriteRPZFile(isStart bool, viewID string) error {
	redirectionList, err := handler.redirectData(viewID, localZoneType)
	if err != nil {
		return err
	}

	if viewID == "" {
		if err := removeFiles(filepath.Join(handler.dnsConfPath, "redirection"), "rpz_", ""); err != nil {
			return fmt.Errorf("delete all the rpz file in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}
	} else {
		path := filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+viewID)
		if err := PathExists(path); err == nil {
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("delete %s file err: %s", path, err.Error())
			}
		}
	}

	if !isStart {
		if err := handler.rewriteNamedViewFile(false, true); err != nil {
			return fmt.Errorf("rewriteRPZFile rewriteNamedViewFile failed:%s", err.Error())
		}
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

	if err := handler.rewriteNamedViewFile(isStart, false); err != nil {
		return fmt.Errorf("rewriteRPZFile rewriteNamedViewFile failed:%s", err.Error())
	}

	return nil
}

func PathExists(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return err
		}
		return err
	}
	return nil
}

func (handler *DNSHandler) redirectData(viewID, redirectType string) ([]RedirectionData, error) {
	var data []RedirectionData
	if viewID == "" {
		var viewList []*resource.AgentView
		if err := dbhandler.List(&viewList); err != nil {
			return nil, err
		}
		for _, view := range viewList {
			cond := map[string]interface{}{
				"view":    view.ID,
				"orderby": "create_time",
			}
			if redirectType != "" {
				cond["redirect_type"] = redirectType
			}
			var redirectList []*resource.AgentRedirection
			if err := dbhandler.ListByCondition(&redirectList, cond); err != nil {
				return nil, err
			}
			if len(redirectList) == 0 {
				continue
			}
			oneRedirectionData := RedirectionData{ViewName: view.Name}
			for _, redirection := range redirectList {
				oneRR := RR{Name: redirection.Name, Type: redirection.DataType, Value: redirection.Rdata, TTL: strconv.FormatUint(uint64(redirection.Ttl), 10)}
				oneRedirectionData.RRs = append(oneRedirectionData.RRs, oneRR)
			}

			data = append(data, oneRedirectionData)
		}
	} else {
		viewRes, err := dbhandler.Get(viewID, &[]*resource.AgentView{})
		if err != nil {
			return nil, err
		}
		view := viewRes.(*resource.AgentView)
		cond := map[string]interface{}{
			"view":    view.ID,
			"orderby": "create_time",
		}
		if redirectType != "" {
			cond["redirect_type"] = redirectType
		}
		var redirectList []*resource.AgentRedirection
		if err := dbhandler.ListByCondition(&redirectList, cond); err != nil {
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
	}

	return data, nil
}

func (handler *DNSHandler) FlushTemplateFiles(tplName, tplConfName string, data interface{}) error {
	buffer := new(bytes.Buffer)
	if err := handler.tpl.ExecuteTemplate(buffer, tplName, data); err != nil {
		return fmt.Errorf("FlushTemplateFiles tplName:%s tplConfName:%s  failed:%s", tplName, tplConfName, err.Error())
	}

	if err := ioutil.WriteFile(tplConfName, buffer.Bytes(), 0644); err != nil {
		return fmt.Errorf("FlushTemplateFiles tplName:%s tplConfName:%s  WriteFile failed:%s", tplName, tplConfName, err.Error())
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
