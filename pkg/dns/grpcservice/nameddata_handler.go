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

	"github.com/linkingthing/ddi-agent/pkg/dns/dbhandler"
	"github.com/linkingthing/ddi-agent/pkg/dns/resource"
)

type namedData struct {
	ConfigPath   string
	ACLNames     []string
	Views        []View
	DNS64s       []Dns64
	IPBlackHole  *ipBlackHole
	LogEnable    bool
	SortList     []string
	Concu        *recursiveConcurrent
	DnssecEnable bool
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

type RR struct {
	Name  string
	Type  string
	Value string
	TTL   string
}

type Zone struct {
	Name        string
	ZoneFile    string
	ForwardType string
	IPs         []string
}

type redirectionData struct {
	ViewName string
	RRs      []RR
}

type recursiveConcurrent struct {
	RecursiveClients *uint32
	FetchesPerZone   *uint32
}

type nzfData struct {
	ViewName string
	Zones    []Zone
}

type zoneData struct {
	ViewName string
	Name     string
	ZoneFile string
	TTL      string
	RRs      []RR
}

type nginxDefaultConf struct {
	URLRedirects []urlRedirect
}

type urlRedirect struct {
	Domain string
	URL    string
}

func (handler *DNSHandler) namedConfData() (*namedData, error) {
	var err error
	data := namedData{ConfigPath: handler.dnsConfPath}

	var aclList []*resource.AgentAcl
	if err := dbhandler.List(&aclList); err != nil {
		return nil, fmt.Errorf("aclListRes failed:%s", err.Error())
	}

	for _, acl := range aclList {
		if acl.ID != anyACL && acl.ID != noneACL {
			data.ACLNames = append(data.ACLNames, acl.Name)
		}
	}

	globalConfigRes, err := dbhandler.Get(defaultGlobalConfigID, &[]*resource.AgentDnsGlobalConfig{})
	if err != nil {
		return nil, err
	}
	globalConfig := globalConfigRes.(*resource.AgentDnsGlobalConfig)
	data.LogEnable = globalConfig.LogEnable
	data.DnssecEnable = globalConfig.DnssecEnable
	var viewList []*resource.AgentView
	if err := dbhandler.ListByCondition(&viewList,
		map[string]interface{}{"orderby": "priority"}); err != nil {
		return nil, err
	}

	for _, value := range viewList {
		var acls []ACL
		for _, aclValue := range value.Acls {
			acls = append(acls, ACL{Name: aclValue})
		}
		view := View{Name: value.Name, ACLs: acls, Key: value.Key}

		if value.Name == defaultView {
			data.Views = append(data.Views, view)
			continue
		}

		var redirectionList []*resource.AgentRedirection
		if err := dbhandler.ListByCondition(&redirectionList,
			map[string]interface{}{"view": value.ID}); err != nil {
			return nil, err
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
			if len(rpzRR) > 0 {
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
			return nil, err
		}

		for _, forwardZone := range forwardZoneList {
			var tmp Zone
			tmp.Name = forwardZone.Name
			tmp.ForwardType = forwardZone.ForwardType
			tmp.IPs = append(tmp.IPs, forwardZone.Ips...)
			view.Zones = append(view.Zones, tmp)
		}

		data.Views = append(data.Views, view)
	}

	var ipBlackHoleList []*resource.AgentIpBlackHole
	if err := dbhandler.List(&ipBlackHoleList); err != nil {
		return nil, err
	}
	for _, ipBlack := range ipBlackHoleList {
		data.IPBlackHole = &ipBlackHole{}
		data.IPBlackHole.ACLNames = append(data.IPBlackHole.ACLNames, ipBlack.Acl)
	}

	exist, err := dbhandler.Exist(resource.TableRecursiveConcurrent, defaultRecursiveConcurrentId)
	if err != nil {
		return nil, err
	}
	if exist {
		recursiveConcurrentRes, err := dbhandler.Get(defaultRecursiveConcurrentId, &[]*resource.AgentRecursiveConcurrent{})
		if err != nil {
			return nil, err
		}
		recursiveCon := recursiveConcurrentRes.(*resource.AgentRecursiveConcurrent)
		data.Concu = &recursiveConcurrent{
			RecursiveClients: &recursiveCon.RecursiveClients,
			FetchesPerZone:   &recursiveCon.FetchesPerZone,
		}
	}

	return &data, nil
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

func (handler *DNSHandler) redirectData(viewID, redirectType string) ([]redirectionData, error) {
	var data []redirectionData
	if viewID == "" {
		var viewList []*resource.AgentView
		if err := dbhandler.List(&viewList); err != nil {
			return nil, err
		}
		for _, view := range viewList {
			if view.Name == defaultView {
				continue
			}
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
			oneRedirectionData := redirectionData{ViewName: view.Name}
			for _, redirection := range redirectList {
				formatCnameValue(&redirection.Rdata, redirection.DataType)
				formatDomain(&redirection.Name, redirection.DataType, redirection.RedirectType)
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
		oneRedirectionData := redirectionData{ViewName: view.Name}
		for _, redirection := range redirectList {
			formatCnameValue(&redirection.Rdata, redirection.DataType)
			formatDomain(&redirection.Name, redirection.DataType, redirection.RedirectType)

			oneRR := RR{Name: redirection.Name, Type: redirection.DataType, Value: redirection.Rdata, TTL: strconv.FormatUint(uint64(redirection.Ttl), 10)}
			oneRedirectionData.RRs = append(oneRedirectionData.RRs, oneRR)
		}

		data = append(data, oneRedirectionData)
	}

	return data, nil
}

func (handler *DNSHandler) zonesData(zoneId string) ([]zoneData, error) {
	var zonesData []zoneData
	if zoneId != "" {
		zoneRes, err := dbhandler.Get(zoneId, &[]*resource.AgentZone{})
		if err != nil {
			return nil, err
		}

		var rrList []*resource.AgentRr
		if err := dbhandler.ListByCondition(&rrList,
			map[string]interface{}{"zone": zoneId, "orderby": "create_time"}); err != nil {
			return nil, err
		}
		zone := zoneRes.(*resource.AgentZone)

		oneZone := zoneData{ViewName: zone.View, Name: zone.Name, ZoneFile: zone.ZoneFile, TTL: strconv.FormatUint(uint64(zone.Ttl), 10)}
		for _, rr := range rrList {
			oneRR := RR{Name: rr.Name, Type: rr.DataType, Value: rr.Rdata, TTL: strconv.FormatUint(uint64(rr.Ttl), 10)}
			oneZone.RRs = append(oneZone.RRs, oneRR)
		}
		zonesData = append(zonesData, oneZone)
	} else {
		var zoneList []*resource.AgentZone
		if err := dbhandler.List(&zoneList); err != nil {
			return nil, err
		}

		for _, zone := range zoneList {
			var rrList []*resource.AgentRr
			if err := dbhandler.ListByCondition(&rrList,
				map[string]interface{}{"zone": zone.ID, "orderby": "create_time"}); err != nil {
				return nil, err
			}
			oneZone := zoneData{ViewName: zone.View, Name: zone.Name, ZoneFile: zone.ZoneFile, TTL: strconv.FormatUint(uint64(zone.Ttl), 10)}
			for _, rr := range rrList {
				oneRR := RR{Name: rr.Name, Type: rr.DataType, Value: rr.Rdata, TTL: strconv.FormatUint(uint64(rr.Ttl), 10)}
				oneZone.RRs = append(oneZone.RRs, oneRR)
			}
			zonesData = append(zonesData, oneZone)
		}
	}

	return zonesData, nil
}

func (handler *DNSHandler) rewriteNamedFile(isExistRPZ bool) (err error) {
	ndata, err := handler.namedConfData()
	if err != nil {
		return err
	}

	buffer := new(bytes.Buffer)
	if isExistRPZ {
		if err = handler.tpl.ExecuteTemplate(buffer, namedNoRPZTpl, ndata); err != nil {
			return err
		}
	} else {
		if err = handler.tpl.ExecuteTemplate(buffer, namedTpl, ndata); err != nil {
			return err
		}
	}
	if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, mainConfName), buffer.Bytes(), 0644); err != nil {
		return err
	}

	return
}

func (handler *DNSHandler) rewriteZonesFile(zoneId string) error {
	zonesData, err := handler.zonesData(zoneId)
	if err != nil {
		return err
	}

	if zoneId == "" {
		if err := removeFiles(handler.dnsConfPath, "", zoneSuffix); err != nil {
			return fmt.Errorf("remvoe files for %s*.zone fail", handler.dnsConfPath)
		}
	}

	buf := new(bytes.Buffer)
	for _, zoneData := range zonesData {
		buf.Reset()
		if err = handler.tpl.ExecuteTemplate(buf, zoneTpl, zoneData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, zoneData.ZoneFile), buf.Bytes(), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (handler *DNSHandler) aclData() ([]ACL, error) {
	var aclData []ACL
	var aclList []*resource.AgentAcl
	if err := dbhandler.List(&aclList); err != nil {
		return nil, err
	}

	for _, acl := range aclList {
		oneAcl := ACL{ID: acl.ID, Name: acl.Name}
		oneAcl.Ips = append(oneAcl.Ips, acl.Ips...)
		aclData = append(aclData, oneAcl)
	}

	return aclData, nil
}

func (handler *DNSHandler) rewriteACLsFile() error {
	aclList, err := handler.aclData()
	if err != nil {
		return err
	}

	if err := removeFiles(handler.dnsConfPath, "", aclSuffix); err != nil {
		return fmt.Errorf("remvoe files for %s*.acl fail", handler.dnsConfPath)
	}

	buf := new(bytes.Buffer)
	for _, aCL := range aclList {
		if aCL.Name == "any" || aCL.Name == "none" {
			continue
		}

		buf.Reset()
		if err := handler.tpl.ExecuteTemplate(buf, aCLTpl, aCL); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, aCL.ID)+aclSuffix, buf.Bytes(), 0644); err != nil {
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

func (handler *DNSHandler) rewriteRedirectFile(viewID string) error {
	redirectionList, err := handler.redirectData(viewID, nxDomain)
	if err != nil {
		return err
	}
	if viewID == "" {
		if err := removeFiles(filepath.Join(handler.dnsConfPath, "redirection"), "redirect_", ""); err != nil {
			return fmt.Errorf("delete all the rpz file in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}
	}

	if len(redirectionList) == 0 {
		if viewID != "" {
			path := filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+viewID)
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("delete redirect file in %s err: %s", path, err.Error())
			}
		}
		return nil
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

func (handler *DNSHandler) rewriteRPZFile(isStart bool, viewID string) error {
	redirectionList, err := handler.redirectData(viewID, localZoneType)
	if err != nil {
		return err
	}

	if !isStart && viewID == "" {
		if err := removeFiles(filepath.Join(handler.dnsConfPath, "redirection"), "rpz_", ""); err != nil {
			return fmt.Errorf("delete all the rpz file in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}

		if err := handler.rewriteNamedFile(true); err != nil {
			return err
		}
	}

	if len(redirectionList) == 0 {
		if viewID != "" {
			path := filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+viewID)
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("delete rpz file in %s err: %s", path, err.Error())
			}
		}
		return nil
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

func (handler *DNSHandler) rndcReconfig() error {
	var para1 = "-c" + filepath.Join(handler.dnsConfPath, "rndc.conf")
	var para2 = "-s" + "localhost"
	var para3 = "-p" + rndcPort
	var para4 = "reconfig"
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), para1, para2, para3, para4); err != nil {
		return fmt.Errorf("rndc reconfig error, %w", err)
	}
	return nil
}

func (handler *DNSHandler) rndcAddZone(name string, zoneFile string, viewName string) error {
	var para1 = "-c" + filepath.Join(handler.dnsConfPath, "rndc.conf")
	var para2 = "-s" + "localhost"
	var para3 = "-p" + rndcPort
	var para4 = "addzone " + name + " in " + viewName + " { type master; file \"" + zoneFile + "\";};"
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), para1, para2, para3, para4); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rndcDelZone(name string, viewName string) error {
	var para1 = "-c" + filepath.Join(handler.dnsConfPath, "rndc.conf")
	var para2 = "-s" + "localhost"
	var para3 = "-p" + rndcPort
	var para4 = "delzone " + name + " in " + viewName
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), para1, para2, para3, para4); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rndcDumpJNLFile() error {
	var para1 = "-c" + filepath.Join(handler.dnsConfPath, "rndc.conf")
	var para2 = "-s" + "localhost"
	var para3 = "-p" + rndcPort
	var para4 = "sync"
	var para5 = "-clean"
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), para1, para2, para3, para4, para5); err != nil {
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
