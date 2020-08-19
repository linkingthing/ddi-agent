package grpcservice

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/linkingthing/ddi-agent/pkg/db"

	"github.com/linkingthing/ddi-agent/pkg/boltdb"
	"github.com/zdnscloud/cement/shell"
)

type namedData struct {
	ConfigPath   string
	ACLNames     []string
	Views        []View
	DNS64s       []Dns64
	IPBlackHole  *ipBlackHole
	IsLogOpen    bool
	SortList     []string
	Concu        *recursiveConcurrent
	IsDnssecOpen bool
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
	IPs  []string
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

func (handler *DNSHandler) namedConfData() (*namedData, error) {
	var err error
	data := namedData{ConfigPath: handler.dnsConfPath}
	//TODO newdb innovation begin point--

	//get all the acl names.
	aclList, err := db.ListAcl()
	if err != nil {
		return nil, err
	}

	for _, acl := range aclList {
		if acl.ID != anyACL && acl.ID != noneACL {
			data.ACLNames = append(data.ACLNames, acl.Name)
		}
	}

	globalConfig, err := db.GetDnsGlobalConfig()
	if err != nil {
		return nil, err
	}
	//get log data
	data.IsLogOpen = globalConfig.IsLogEnable
	//get dnssec data
	data.IsDnssecOpen = globalConfig.IsDnssecEnable
	//get the data under the views
	viewList, err := db.ListView()
	if err != nil {
		return nil, err
	}
	for _, value := range viewList {
		//get the acls and name data
		var acls []ACL
		for _, aclValue := range value.Acls {
			acls = append(acls, ACL{Name: aclValue})
		}
		view := View{Name: value.Name, ACLs: acls}
		//get the redirect data
		view.Redirect = &Redirect{}
		redirections, err := db.ListRedirection(value.ID)
		if err != nil {
			return nil, err
		}
		for _, reValue := range redirections {
			var tmp RR
			tmp.Name = reValue.Name
			tmp.TTL = strconv.Itoa(int(reValue.Ttl))
			tmp.Type = reValue.DataType
			tmp.Value = reValue.Rdata
			view.Redirect.RRs = append(view.Redirect.RRs, tmp)
		}

		//at last append view to data.views
		data.Views = append(data.Views, view)
	}

	var kvs map[string][]byte
	kvs, err = boltdb.GetDB().GetTableKVs(viewsEndPath)
	viewid := string(kvs["next"])
	for viewid != "" {
		nameKvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, string(viewid)))
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsPath, string(viewid), aCLsEndPath)); err != nil {
			return nil, err
		}
		var aCLs []ACL
		for i := 0; i < len(tables); i++ {
			aCLids, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, string(viewid), aCLsEndPath, fmt.Sprintf("%d", i+1)))
			if err != nil {
				return nil, err
			}
			if len(aCLids) == 0 {
				return nil, fmt.Errorf("view %s' the %d's acl not exists", string(viewid), i+1)
			}
			aCL := ACL{Name: string(aCLids[0])}
			aCLs = append(aCLs, aCL)
		}
		view := View{Name: string(viewName), ACLs: aCLs}
		//get the redirect data
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsPath, string(viewid), redirectEndPath)); err != nil {
			return nil, err
		}
		if len(tables) > 0 {
			var tmp redierct
			view.Redirect = &tmp
		}
		for _, rrid := range tables {
			rrMap, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, string(viewid), redirectPath, rrid))
			if err != nil {
				return nil, err
			}
			var tmp RR
			tmp.Name = string(rrMap["name"])
			tmp.TTL = string(rrMap["ttl"])
			tmp.Type = string(rrMap["type"])
			tmp.Value = string(rrMap["value"])
			view.Redirect.RRs = append(view.Redirect.RRs, tmp)
		}
		//get the RPZ data
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsPath, string(viewid), rpzEndPath)); err != nil {
			return nil, err
		}
		if len(tables) > 0 {
			var tmp rpz
			view.RPZ = &tmp
		}
		for _, rrid := range tables {
			rrMap, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, string(viewid), rpzPath, rrid))
			if err != nil {
				return nil, err
			}
			var tmp RR
			tmp.Name = string(rrMap["name"])
			tmp.TTL = string(rrMap["ttl"])
			tmp.Type = string(rrMap["type"])
			tmp.Value = string(rrMap["value"])
			view.RPZ.RRs = append(view.RPZ.RRs, tmp)
		}
		//get the dns64s data
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsPath, string(viewid), dns64sEndPath)); err != nil {
			return nil, err
		}
		for _, dns64id := range tables {
			dns64Map, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, string(viewid), dns64sPath, dns64id))
			if err != nil {
				return nil, err
			}
			var tmp dns64
			tmp.Prefix = string(dns64Map["prefix"])
			aCLNames, err := boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, string(dns64Map["clientacl"])))
			if err != nil {
				return nil, err
			}
			tmp.ClientACLName = string(aCLNames["name"])
			aCLNames, err = boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, string(dns64Map["aaddress"])))
			if err != nil {
				return nil, err
			}
			tmp.AAddressACLName = string(aCLNames["name"])
			view.DNS64s = append(view.DNS64s, tmp)
		}
		//get the forward data under the zone
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsPath, string(viewid), zonesEndPath)); err != nil {
			return nil, err
		}
		for _, zoneid := range tables {
			zoneMap, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, string(viewid), zonesPath, zoneid))
			if err != nil {
				return nil, err
			}
			if string(zoneMap["zonetype"]) != forwardType {
				continue
			}
			var tmp Zone
			tmp.Name = string(zoneMap["name"])
			tmp.ForwardType = string(zoneMap["forwardtype"])
			forwardTables, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, string(viewid), zonesPath, zoneid, forwardsEndPath))
			if err != nil {
				return nil, err
			}
			for _, id := range forwardTables {
				ips, err := boltdb.GetDB().GetTableKVs(filepath.Join(forwardsPath, id))
				if err != nil {
					return nil, err
				}
				for ip, _ := range ips {
					tmp.IPs = append(tmp.IPs, ip)
				}
			}
			view.Zones = append(view.Zones, tmp)
		}
		//get the base64 of the view's Key
		keyMap, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, string(viewid)))
		if err != nil {
			return nil, err
		}
		encodeString := base64.StdEncoding.EncodeToString(keyMap["key"])
		view.Key = encodeString

		data.Views = append(data.Views, view)

		kvs, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid))
		if err != nil {
			return nil, err
		}
		viewid = string(kvs["next"])
		//TODO newdb innovation end point--

		//get the ip black hole data
		var tables []string
		tables, err = boltdb.GetDB().GetTables(ipBlackHoleEndPath)
		if err != nil {
			return nil, err
		}
		if len(tables) > 0 {
			var tmp ipBlackHole
			data.IPBlackHole = &tmp
		}
		for _, id := range tables {
			var blackholeKVs map[string][]byte
			blackholeKVs, err = boltdb.GetDB().GetTableKVs(filepath.Join(ipBlackHolePath, id))
			if err != nil {
				return nil, err
			}
			aclid := blackholeKVs["aclid"]
			var aclNameKVs map[string][]byte
			aclNameKVs, err = boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, string(aclid)))
			if err != nil {
				return nil, err
			}
			data.IPBlackHole.ACLNames = append(data.IPBlackHole.ACLNames, string(aclNameKVs["name"]))
		}
		//get recursive concurrency data
		var concuKVs map[string][]byte
		concuKVs, err = boltdb.GetDB().GetTableKVs(recurConcurEndPath)
		if err != nil {
			return nil, err
		}
		if len(concuKVs) > 0 {
			var tmp recursiveConcurrent
			data.Concu = &tmp
			if string(concuKVs["recursiveclients"]) != "" {
				var num int
				if num, err = strconv.Atoi(string(concuKVs["recursiveclients"])); err != nil {
					return nil, err
				}
				data.Concu.RecursiveClients = &num
			}
			if string(concuKVs["fetchesperzone"]) != "" {
				var num int
				if num, err = strconv.Atoi(string(concuKVs["fetchesperzone"])); err != nil {
					return nil, err
				}
				data.Concu.FetchesPerZone = &num
			}
		}

		//get the sortlist
		var sortkvs map[string][]byte
		sortkvs, err = boltdb.GetDB().GetTableKVs(sortListEndPath)
		if err != nil {
			return nil, err
		}
		aclid := string(sortkvs["next"])
		for {
			if aclid == "" {
				break
			}
			//get acl name
			var aclName map[string][]byte
			aclName, err = boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, aclid))
			if err != nil {
				return nil, err
			}
			data.SortList = append(data.SortList, string(aclName["name"]))
			var kvs map[string][]byte
			kvs, err = boltdb.GetDB().GetTableKVs(filepath.Join(sortListPath, aclid))
			if err != nil {
				return nil, err
			}
			aclid = string(kvs["next"])
		}

	}
	return &data, nil
}

func (handler *DNSHandler) aCLsData() ([]ACL, error) {
	var err error
	var aCLsData []ACL
	var viewTables []string
	viewTables, err = boltdb.GetDB().GetTables(viewsEndPath)
	if err != nil {
		return nil, err
	}
	for _, viewid := range viewTables {
		var aCLs []ACL
		aCLTables, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, viewid, aCLsEndPath))
		if err != nil {
			return nil, err
		}
		for _, aCLid := range aCLTables {
			aCLNames, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid, aCLsPath, aCLid))
			if err != nil {
				return nil, err
			}
			aCLName := aCLNames["name"]
			ipsMap, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid, aCLsPath, aCLid, iPsEndPath))
			if err != nil {
				return nil, err
			}
			var ips []string
			for ip, _ := range ipsMap {
				ips = append(ips, ip)
			}
			aCL := ACL{Name: string(aCLName), IPs: ips}
			aCLs = append(aCLs, aCL)
		}
		aCLsData = append(aCLsData, aCLs[0:]...)
	}
	var aCLTables []string
	aCLTables, err = boltdb.GetDB().GetTables(aCLsEndPath)
	if err != nil {
		return nil, err
	}
	for _, aCLId := range aCLTables {
		names, err := boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, aCLId))
		if err != nil {
			return nil, err
		}
		aCLName := names["name"]
		ipsMap, err := boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, aCLId, iPsEndPath))
		if err != nil {
			return nil, err
		}
		var ips []string
		for ip, _ := range ipsMap {
			ips = append(ips, ip)
		}
		oneACL := ACL{Name: string(aCLName), IPs: ips}
		aCLsData = append(aCLsData, oneACL)
	}
	return aCLsData, nil
}

func (handler *DNSHandler) nzfsData() ([]nzfData, error) {
	var data []nzfData
	viewIDs, err := boltdb.GetDB().GetTables(viewsEndPath)
	if err != nil {
		return nil, err
	}
	for _, viewid := range viewIDs {
		nameKvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid))
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		zoneTables, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, viewid, zonesEndPath))
		if err != nil {
			return nil, err
		}
		var zones []Zone
		for _, zoneId := range zoneTables {
			Names, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid, zonesPath, zoneId))
			if err != nil {
				return nil, err
			}
			zoneName := Names["name"]
			zoneFile := Names["zonefile"]
			if string(zoneFile) == "" { //the type of the forward zone has no zone file.
				continue
			}
			zone := Zone{Name: string(zoneName), ZoneFile: string(zoneFile)}
			zones = append(zones, zone)
		}
		oneNzfData := nzfData{ViewName: string(viewName), Zones: zones}
		data = append(data, oneNzfData)
	}
	return data, nil
}

func (handler *DNSHandler) redirectData() ([]redirectionData, error) {
	var data []redirectionData
	viewIDs, err := boltdb.GetDB().GetTables(viewsEndPath)
	if err != nil {
		return nil, err
	}
	for _, viewid := range viewIDs {
		var one redirectionData
		nameKvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid))
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		one.ViewName = string(viewName)
		rrsid, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, viewid, redirectEndPath))
		if err != nil {
			return nil, err
		}
		for _, rrID := range rrsid {
			rrs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid, redirectPath, rrID))
			if err != nil {
				return nil, err
			}
			name := rrs["name"]
			ttl := rrs["ttl"]
			dataType := rrs["type"]
			value := rrs["value"]
			rr := RR{Name: string(name), Type: string(dataType), TTL: string(ttl), Value: string(value)}
			one.RRs = append(one.RRs, rr)
		}
		if len(rrsid) > 0 {
			data = append(data, one)
		}
	}
	return data, nil
}

func (handler *DNSHandler) rpzData() ([]redirectionData, error) {
	var data []redirectionData
	viewIDs, err := boltdb.GetDB().GetTables(viewsEndPath)
	if err != nil {
		return nil, err
	}
	for _, viewid := range viewIDs {
		var one redirectionData
		nameKvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid))
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		one.ViewName = string(viewName)
		rrsid, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, viewid, rpzEndPath))
		if err != nil {
			return nil, err
		}
		for _, rrID := range rrsid {
			rrs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid, rpzPath, rrID))
			if err != nil {
				return nil, err
			}
			name := rrs["name"]
			ttl := rrs["ttl"]
			dataType := rrs["type"]
			value := rrs["value"]
			rr := RR{Name: string(name), Type: string(dataType), TTL: string(ttl), Value: string(value)}
			one.RRs = append(one.RRs, rr)
		}
		if len(rrsid) > 0 {
			data = append(data, one)
		}
	}
	return data, nil
}

func (handler *DNSHandler) zonesData() ([]zoneData, error) {
	var zonesData []zoneData
	viewIDs, err := boltdb.GetDB().GetTables(viewsEndPath)
	if err != nil {
		return nil, err
	}
	for _, viewid := range viewIDs {
		nameKvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid))
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		zoneTables, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, viewid, zonesEndPath))
		if err != nil {
			return nil, err
		}
		for _, zoneID := range zoneTables {
			var rrs []RR
			names, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid, zonesPath, zoneID))
			if err != nil {
				return nil, err
			}
			zoneName := names["name"]
			zoneFile := names["zonefile"]
			ttl := names["ttl"]
			if string(zoneFile) == "" {
				continue
			}
			rrTables, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, viewid, zonesPath, zoneID, rRsEndPath))
			if err != nil {
				return nil, err
			}
			for _, rrID := range rrTables {
				datas, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid, zonesPath, zoneID, rRsPath, rrID))
				if err != nil {
					return nil, err
				}
				rr := RR{Name: string(datas["name"]), TTL: string(datas["ttl"]), Type: string(datas["type"]), Value: string(datas["value"])}
				rrs = append(rrs, rr)
			}
			one := zoneData{ViewName: string(viewName), Name: string(zoneName), ZoneFile: string(zoneFile), RRs: rrs, TTL: string(ttl)}
			zonesData = append(zonesData, one)
		}
	}
	return zonesData, nil
}

func (handler *DNSHandler) rewriteNamedFile(isExistRPZ bool) error {
	var namedConfData *namedData
	var err error
	if namedConfData, err = handler.namedConfData(); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	buffer := new(bytes.Buffer)
	if isExistRPZ {
		if err = handler.tpl.ExecuteTemplate(buffer, namedNoRPZTpl, namedConfData); err != nil {
			return err
		}
	} else {
		if err = handler.tpl.ExecuteTemplate(buffer, namedTpl, namedConfData); err != nil {
			return err
		}
	}
	if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, mainConfName), buffer.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rewriteZonesFile() error {
	zonesData, err := handler.zonesData()
	if err != nil {
		return err
	}
	if err := removeFiles(handler.dnsConfPath, "", zoneSuffix); err != nil {
		return fmt.Errorf("remvoe files for %s*.zone fail", handler.dnsConfPath)
	}
	for _, zoneData := range zonesData {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, zoneTpl, zoneData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, zoneData.ZoneFile), buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) rewriteACLsFile() error {
	aCLs, err := handler.aCLsData()
	if err != nil {
		return err
	}
	if err := removeFiles(handler.dnsConfPath, "", aclSuffix); err != nil {
		return fmt.Errorf("remvoe files for %s*.zone fail", handler.dnsConfPath)
	}
	for _, aCL := range aCLs {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, aCLTpl, aCL); err != nil {
			return err
		}
		if aCL.Name == "any" || aCL.Name == "none" {
			continue
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, aCL.Name)+aclSuffix, buf.Bytes(), 0644); err != nil {
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

func (handler *DNSHandler) rewriteRedirectFile() error {
	redirectionsData, err := handler.redirectData()
	if err != nil {
		return err
	}
	if err := removeFiles(filepath.Join(handler.dnsConfPath, "redirection"), "redirect_", ""); err != nil {
		return fmt.Errorf("delete all the rpz file in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
	}
	for _, redirectionData := range redirectionsData {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, redirectTpl, redirectionData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+redirectionData.ViewName), buf.Bytes(), 0644); err != nil {
			return err
		}
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

func (handler *DNSHandler) rewriteRPZFile(isStart bool) error {
	redirectionsData, err := handler.rpzData()
	if err != nil {
		return err
	}
	if !isStart {
		if err := removeFiles(filepath.Join(handler.dnsConfPath, "redirection"), "rpz_", ""); err != nil {
			return fmt.Errorf("delete all the rpz file in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}
		if err := handler.rewriteNamedFile(true); err != nil {
			return err
		}
		if err := handler.rndcReconfig(); err != nil {
			return err
		}
	}

	for _, redirectionData := range redirectionsData {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, rpzTpl, redirectionData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+redirectionData.ViewName), buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	if !isStart {
		if err := handler.rewriteNamedFile(false); err != nil {
			return err
		}
		if err := handler.rndcReconfig(); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) addPriority(pri int, viewid string, isCreateView bool) error {
	//if the viewid has already in the database then return nil
	viewskvs, err := boltdb.GetDB().GetTableKVs(viewsEndPath)
	if err != nil {
		return fmt.Errorf("get %s tablekvs err:%s", viewsEndPath, err.Error())
	}
	if pri == 1 {
		if err := boltdb.GetDB().UpdateKVs(viewsEndPath, map[string][]byte{"next": []byte(viewid)}); err != nil {
			return fmt.Errorf("update %s tablekvs err:%s", viewsEndPath, err.Error())
		}
		//add new node.
		if isCreateView {
			if err := boltdb.GetDB().AddKVs(filepath.Join(viewsPath, viewid), map[string][]byte{"next": viewskvs["next"]}); err != nil {
				return fmt.Errorf("add %s kvs err:%s", filepath.Join(viewsPath, viewid), err.Error())
			}
		} else {
			if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsPath, viewid), map[string][]byte{"next": viewskvs["next"]}); err != nil {
				return fmt.Errorf("update %s tablekvs err:%s", filepath.Join(viewsPath, viewid), err.Error())
			}
		}
	} else {
		previd := string(viewskvs["next"])
		if viewskvs, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, previd)); err != nil {
			return fmt.Errorf("get %s tablekvs err:%s", filepath.Join(viewsPath, previd), err.Error())
		}
		nextid := string(viewskvs["next"])

		var count int
		for count = 1; count < pri-1 && nextid != ""; count++ {
			if viewskvs, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, nextid)); err != nil {
				return fmt.Errorf("get %s tablekvs err:%s", filepath.Join(viewsPath, nextid), err.Error())
			}
			previd = nextid
			nextid = string(viewskvs["next"])
		}
		//update the previous one
		if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsPath, previd), map[string][]byte{"next": []byte(viewid)}); err != nil {
			return fmt.Errorf("update %s tablekvs err:%s", filepath.Join(viewsPath, previd), err.Error())
		}
		//add new node.
		if isCreateView {
			if err := boltdb.GetDB().AddKVs(filepath.Join(viewsPath, viewid), map[string][]byte{"next": []byte(nextid)}); err != nil {
				return fmt.Errorf("add %s tablekvs err:%s", filepath.Join(viewsPath, viewid), err.Error())
			}
		} else {
			if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsPath, viewid), map[string][]byte{"next": []byte(nextid)}); err != nil {
				return fmt.Errorf("update %s tablekvs err:%s", filepath.Join(viewsPath, viewid), err.Error())
			}
		}
	}
	return nil
}

func (handler *DNSHandler) updatePriority(pri int, viewid string) error {
	if err := handler.deletePriority(viewid); err != nil {
		return fmt.Errorf("delete priority err:%s", err.Error())
	}
	if err := handler.addPriority(pri, viewid, false); err != nil {
		return fmt.Errorf("add priority err:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) deletePriority(viewID string) error {
	// find the previd and nextid of the viewid.
	ids, err := boltdb.GetDB().GetTableKVs(viewsEndPath)
	if err != nil {
		return fmt.Errorf("get %s tablekvs err:%s", viewsEndPath, err.Error())
	}
	if string(ids["next"]) == viewID {
		ids, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewID))
		if err != nil {
			return fmt.Errorf("get %s tablekvs err:%s", filepath.Join(viewsPath, viewID), err.Error())
		}
		if err = boltdb.GetDB().UpdateKVs(viewsEndPath, map[string][]byte{"next": ids["next"]}); err != nil {
			return fmt.Errorf("update %s tablekvs err:%s", viewsEndPath, err.Error())
		}
	} else {
		previd := string(ids["next"])
		if ids, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, previd)); err != nil {
			return fmt.Errorf("get %s tablekvs err:%s", filepath.Join(viewsPath, previd), err.Error())
		}
		nextid := string(ids["next"])

		for nextid != "" && nextid != viewID {
			if ids, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, nextid)); err != nil {
				return fmt.Errorf("get %s tablekvs err:%s", filepath.Join(viewsPath, nextid), err.Error())
			}
			previd = nextid
			nextid = string(ids["next"])
		}
		if nextid == viewID {
			if ids, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, nextid)); err != nil {
				return fmt.Errorf("get %s tablekvs err:%s", filepath.Join(viewsPath, nextid), err.Error())
			}
			if err = boltdb.GetDB().UpdateKVs(filepath.Join(viewsPath, previd), map[string][]byte{"next": ids["next"]}); err != nil {
				return fmt.Errorf("update %s tablekvs err:%s", filepath.Join(viewsPath, previd), err.Error())
			}
		}
	}
	return nil
}

func (handler *DNSHandler) aCLsFromTopPath(aCLids []string) ([]ACL, error) {
	var aCLs []ACL
	for _, aCLid := range aCLids {
		names, err := boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, aCLid))
		if err != nil {
			return nil, err
		}
		name := names["name"]
		var ipsmap map[string][]byte
		ipsmap, err = boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, aCLid, iPsEndPath))
		if err != nil {
			return nil, err
		}
		var ips []string
		for ip := range ipsmap {
			ips = append(ips, ip)
		}
		aCL := ACL{Name: string(name), IPs: ips, ID: aCLid}
		aCLs = append(aCLs, aCL)
	}
	return aCLs, nil
}

func (handler *DNSHandler) rndcReconfig() error {
	//update bind
	var para1 string = "-c" + filepath.Join(handler.dnsConfPath, "rndc.conf")
	var para2 string = "-s" + "localhost"
	var para3 string = "-p" + rndcPort
	var para4 string = "reconfig"
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), para1, para2, para3, para4); err != nil {
		return fmt.Errorf("rndc reconfig error, %w", err)
	}
	return nil
}
func (handler *DNSHandler) rndcAddZone(name string, zoneFile string, viewName string) error {
	//update bind
	var para1 string = "-c" + filepath.Join(handler.dnsConfPath, "rndc.conf")
	var para2 string = "-s" + "localhost"
	var para3 string = "-p" + rndcPort
	var para4 string = "addzone " + name + " in " + viewName + " { type master; file \"" + zoneFile + "\";};"
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), para1, para2, para3, para4); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rndcDelZone(name string, zoneFile string, viewName string) error {
	//update bind
	var para1 string = "-c" + filepath.Join(handler.dnsConfPath, "rndc.conf")
	var para2 string = "-s" + "localhost"
	var para3 string = "-p" + rndcPort
	var para4 string = "delzone " + name + " in " + viewName
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), para1, para2, para3, para4); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rndcDumpJNLFile() error {
	//update bind
	var para1 string = "-c" + filepath.Join(handler.dnsConfPath, "rndc.conf")
	var para2 string = "-s" + "localhost"
	var para3 string = "-p" + rndcPort
	var para4 string = "sync"
	var para5 string = "-clean"
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), para1, para2, para3, para4, para5); err != nil {
		return fmt.Errorf("exec rndc sync -clean err:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) updateForward(forwards []string, zoneid string, viewid string) error {
	//delete the ole forwardids
	tables, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, viewid, zonesPath, zoneid, forwardsEndPath))
	if err != nil {
		return err
	}
	for _, id := range tables {
		if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsPath, viewid, zonesPath, zoneid, forwardsPath, id)); err != nil {
			return err
		}
	}
	//add the forwardids
	for _, id := range forwards {
		if _, err := boltdb.GetDB().CreateOrGetTable(filepath.Join(viewsPath, viewid, zonesPath, zoneid, forwardsPath, id)); err != nil {
			return err
		}
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
		return fmt.Errorf("write file %s err", filepath.Join(handler.nginxDefaultConfDir, nginxDefaultConfFile), err.Error())
	}
	return nil
}

func (handler *DNSHandler) GetNginxData() (*nginxDefaultConf, error) {
	data := nginxDefaultConf{}
	kvs, err := boltdb.GetDB().GetTableKVs(viewsEndPath)
	if err != nil {
		return nil, err
	}
	viewid := string(kvs["next"])
	for viewid != "" {
		tables, err := boltdb.GetDB().GetTables(filepath.Join(viewsPath, viewid, urlRedirectsPath))
		if err != nil {
			return nil, err
		}
		for _, t := range tables {
			urlRedirectkvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, string(viewid), urlRedirectsPath, t))
			if err != nil {
				return nil, fmt.Errorf("get path %s kvs error", filepath.Join(viewsPath, string(viewid), urlRedirectsPath, t))
			}
			data.URLRedirects = append(data.URLRedirects, urlRedirect{Domain: string(urlRedirectkvs[domain]), URL: string(urlRedirectkvs[url])})
		}

		kvs, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, viewid))
		if err != nil {
			return nil, err
		}
		viewid = string(kvs["next"])
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
