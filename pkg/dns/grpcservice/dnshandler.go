package grpcservice

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/zdnscloud/cement/shell"
	"github.com/zdnscloud/g53"

	"github.com/linkingthing/ddi-agent/pkg/boltdb"
	"github.com/linkingthing/ddi-agent/pkg/pb"
	"github.com/linkingthing/ddi-metric/utils/random"
)

const (
	mainConfName       = "named.conf"
	dBName             = "bind.db"
	viewsPath          = "/views/"
	viewsEndPath       = "/views"
	zonesPath          = "/zones/"
	zonesEndPath       = "/zones"
	aCLsPath           = "/acls/"
	aCLsEndPath        = "/acls"
	rRsEndPath         = "/rrs"
	rRsPath            = "/rrs/"
	iPsEndPath         = "/ips"
	forwardPath        = "/forward"
	forwardEndPath     = "/forward"
	redirectPath       = "/redirect/"
	redirectEndPath    = "/redirect"
	rpzPath            = "/rpz/"
	rpzEndPath         = "/rpz"
	dns64sPath         = "/dns64s/"
	dns64sEndPath      = "/dns64s"
	ipBlackHolePath    = "/ipBlackHole/"
	ipBlackHoleEndPath = "/ipBlackHole"
	recurConcurEndPath = "/recurConcur"
	sortListPath       = "/sortList/"
	sortListEndPath    = "/sortList"
	namedTpl           = "named.tpl"
	zoneTpl            = "zone.tpl"
	aCLTpl             = "acl.tpl"
	nzfTpl             = "nzf.tpl"
	redirectTpl        = "redirect.tpl"
	rpzTpl             = "rpz.tpl"
	rndcPort           = "953"
	checkPeriod        = 5
	dnsServer          = "localhost:53"
)

type DNSHandler struct {
	tpl         *template.Template
	db          *boltdb.BoltDB
	dnsConfPath string
	dBPath      string
	tplPath     string
	ticker      *time.Ticker
	quit        chan int
}

func newDNSHandler(dnsConfPath string, agentPath string) (*DNSHandler, error) {
	var tmpDnsPath string
	if dnsConfPath[len(dnsConfPath)-1] != '/' {
		tmpDnsPath = dnsConfPath + "/"
	} else {
		tmpDnsPath = dnsConfPath
	}
	var tmpDBPath string
	if agentPath[len(agentPath)-1] != '/' {
		tmpDBPath = agentPath + "/"
	} else {
		tmpDBPath = agentPath
	}

	instance := &DNSHandler{dnsConfPath: tmpDnsPath, dBPath: tmpDBPath, tplPath: tmpDnsPath + "templates/"}
	var err error
	instance.tpl, err = template.ParseFiles(instance.tplPath + namedTpl)
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(instance.tplPath + zoneTpl)
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(instance.tplPath + aCLTpl)
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(instance.tplPath + nzfTpl)
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(instance.tplPath + redirectTpl)
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(instance.tplPath + rpzTpl)
	if err != nil {
		return nil, err
	}

	instance.ticker = time.NewTicker(checkPeriod * time.Second)
	instance.quit = make(chan int)
	//check wether the default acl "any" and "none" is exist.if not add the any and none into the database.
	anykvs := map[string][]byte{}
	anykvs, err = boltdb.GetDB().GetTableKVs(aCLsPath + "1")
	if err != nil {
		return nil, err
	}

	if len(anykvs) == 0 {
		anykvs["name"] = []byte("any")
		if err := boltdb.GetDB().AddKVs(aCLsPath+"1", anykvs); err != nil {
			return nil, err
		}
	}

	nonekvs := map[string][]byte{}
	nonekvs, err = boltdb.GetDB().GetTableKVs(aCLsPath + "2")
	if err != nil {
		return nil, err
	}

	if len(nonekvs) == 0 {
		nonekvs["name"] = []byte("none")
		if err := boltdb.GetDB().AddKVs(aCLsPath+"2", nonekvs); err != nil {
			return nil, err
		}
	}

	viewkvs := map[string][]byte{}
	viewkvs, err = boltdb.GetDB().GetTableKVs(viewsPath + "1000000")
	if err != nil {
		return nil, err
	}

	if len(viewkvs) == 0 {
		prikvs := map[string][]byte{}
		if prikvs, err = boltdb.GetDB().GetTableKVs(viewsEndPath); err != nil {
			return nil, err
		}

		addkvs := map[string][]byte{strconv.Itoa(len(prikvs) + 1): []byte("1000000")}
		if err := boltdb.GetDB().AddKVs(viewsEndPath, addkvs); err != nil {
			return nil, err
		}

		viewkvs["name"] = []byte("default")
		viewkvs["key"] = []byte(random.CreateRandomString(12))
		if err := boltdb.GetDB().AddKVs(viewsPath+"1000000", viewkvs); err != nil {
			return nil, err
		}
	}

	var acls []string
	acls, err = boltdb.GetDB().GetTables(viewsPath + "1000000" + aCLsEndPath)
	if err != nil {
		return nil, err
	}

	if len(acls) == 0 {
		if _, err := boltdb.GetDB().CreateOrGetTable(viewsPath + "1000000" + aCLsPath + "1"); err != nil {
			return nil, err
		}
	}
	req := pb.DNSStartReq{}
	//if err := instance.StartDNS(req); err != nil {
	//panic(err)  //can not exit the program cause other kafka cmd should be execuetd to fix the bind's configure.
	//}
	instance.StartDNS(req)
	return instance, nil
}

type namedData struct {
	ConfigPath  string
	ACLNames    []string
	Views       []View
	Forward     *forward
	DNS64s      []dns64
	IPBlackHole *ipBlackHole
	SortList    []string
	Concu       *recursiveConcurrent
}

type forward struct {
	ForwardType string
	IPs         []string
}

type dns64 struct {
	ID              string
	Prefix          string
	ClientACLName   string
	AAddressACLName string
}

type ipBlackHole struct {
	ACLNames []string
}

type recursiveConcurrent struct {
	RecursiveClients *int
	FetchesPerZone   *int
}

type View struct {
	Name     string
	ACLs     []ACL
	Zones    []Zone
	Redirect *redierct
	RPZ      *rpz
	DNS64s   []dns64
	Key      string
}

type redierct struct {
	RRs []RR
}

type nzfData struct {
	ViewName string
	Zones    []Zone
}

type rpz struct {
	RRs []RR
}

type Zone struct {
	Name        string
	ZoneFile    string
	ForwardType string
	Forwarder   *forwarder
}

type zoneData struct {
	ViewName  string
	Name      string
	ZoneFile  string
	RRs       []RR
	Forwarder *forwarder
}

type redirectionData struct {
	ViewName string
	RRs      []RR
}

type forwarder struct {
	IPs []string
}

type RR struct {
	Name  string
	Type  string
	Value string
	TTL   string
}

type ACL struct {
	ID   string
	Name string
	IPs  []string
}

func (handler *DNSHandler) StartDNS(req pb.DNSStartReq) error {
	handler.Start(req)
	go handler.keepDNSAlive()
	return nil

}
func (handler *DNSHandler) Start(req pb.DNSStartReq) error {
	if _, err := os.Stat(handler.dnsConfPath + "named.pid"); err == nil {
		return nil
	}
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	if err := handler.rewriteACLsFile(); err != nil {
		return err
	}

	if err := handler.rewriteZonesFile(); err != nil {
		return err
	}

	if err := handler.rewriteNzfsFile(); err != nil {
		return err
	}

	if err := handler.rewriteRedirectFile(); err != nil {
		return err
	}

	if err := handler.rewriteRPZFile(); err != nil {
		return err
	}

	var param string = "-c" + handler.dnsConfPath + mainConfName
	if _, err := shell.Shell("named", param); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) StopDNS() error {
	if _, err := os.Stat(handler.dnsConfPath + "named.pid"); err != nil {
		return nil
	}
	var err error
	if _, err = shell.Shell(handler.dnsConfPath+"rndc", "stop"); err != nil {
		return err
	}
	handler.quit <- 1
	return nil
}

func (handler *DNSHandler) CreateACL(req pb.CreateACLReq) error {
	err := boltdb.GetDB().AddKVs(aCLsPath+req.ID, map[string][]byte{"name": []byte(req.Name)})
	if err != nil {
		return err
	}
	values := map[string][]byte{}
	for _, ip := range req.IPs {
		values[ip] = []byte("")
	}
	if err := boltdb.GetDB().AddKVs(aCLsPath+req.ID+iPsEndPath, values); err != nil {
		return err
	}
	aCLData := ACL{ID: req.ID, Name: req.Name, IPs: req.IPs}
	buffer := new(bytes.Buffer)
	if err = handler.tpl.ExecuteTemplate(buffer, aCLTpl, aCLData); err != nil {
		return err
	}
	if err := ioutil.WriteFile(handler.dnsConfPath+req.Name+".conf", buffer.Bytes(), 0644); err != nil {
		return err
	}
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) UpdateACL(req pb.UpdateACLReq) error {
	reqDel := pb.DeleteACLReq{ID: req.ID}
	if err := handler.DeleteACL(reqDel); err != nil {
		return err
	}
	reqTmp := pb.CreateACLReq{Name: req.Name, ID: req.ID, IPs: req.NewIPs}
	if err := handler.CreateACL(reqTmp); err != nil {
		return err
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) DeleteACL(req pb.DeleteACLReq) error {
	kvs, err := boltdb.GetDB().GetTableKVs(aCLsPath + req.ID)
	if err != nil {
		return err
	}
	if len(kvs) == 0 {
		return nil
	}
	name := kvs["name"]
	if err := os.Remove(handler.dnsConfPath + string(name) + ".conf"); err != nil {
		return err
	}

	return boltdb.GetDB().DeleteTable(aCLsPath + req.ID)
}

func (handler *DNSHandler) CreateView(req pb.CreateViewReq) error {
	if err := handler.addPriority(int(req.Priority), req.ViewID); err != nil {
		return err
	}
	//create table viewid and put name into the db.
	namekvs := map[string][]byte{"name": []byte(req.ViewName)}
	namekvs["key"] = []byte(random.CreateRandomString(12))
	if err := boltdb.GetDB().AddKVs(viewsPath+req.ViewID, namekvs); err != nil {
		return err
	}
	//insert aCLIDs into viewid table
	for _, id := range req.ACLs {
		if _, err := boltdb.GetDB().GetTables(viewsPath + req.ViewID + aCLsPath + id); err != nil {
			return err
		}
	}
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateView(req pb.UpdateViewReq) error {
	if err := handler.updatePriority(int(req.Priority), req.ViewID); err != nil {
		return err
	}
	//delete aclids for aCL
	for _, id := range req.DeleteACLs {
		if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID + aCLsPath + id); err != nil {
			return err
		}
	}
	//add new aclids for aCL
	for _, id := range req.AddACLs {
		if _, err := boltdb.GetDB().CreateOrGetTable(viewsPath + req.ViewID + aCLsPath + id); err != nil {
			return err
		}
	}
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	if err := handler.rewriteACLsFile(); err != nil {
		return err
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) DeleteView(req pb.DeleteViewReq) error {
	handler.deletePriority(req.ViewID)
	//delete table
	if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID); err != nil {
		return err
	}
	if err := handler.rewriteNamedFile(); err != nil {
		return nil
	}
	if err := handler.rewriteZonesFile(); err != nil {
		return nil
	}
	if err := handler.rewriteACLsFile(); err != nil {
		return nil
	}
	if err := handler.rewriteNzfsFile(); err != nil {
		return nil
	}
	if err := handler.rndcReconfig(); err != nil {
		return nil
	}
	return nil
}

func (handler *DNSHandler) CreateZone(req pb.CreateZoneReq) error {
	//put the zone into db
	names := map[string][]byte{}
	names["name"] = []byte(req.ZoneName)
	names["zonefile"] = []byte(req.ZoneFileName)
	if err := boltdb.GetDB().AddKVs(viewsPath+req.ViewID+zonesPath+req.ZoneID, names); err != nil {
		return err
	}
	//update file
	if err := handler.rewriteZonesFile(); err != nil {
		return err
	}
	var err error
	out := map[string][]byte{}
	if out, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID); err != nil {
		return err
	}
	viewName := out["name"]
	if req.ZoneFileName == "" { //zone file name is "" stands for the forward zone
		return nil
	}
	if err := handler.rndcAddZone(req.ZoneName, req.ZoneFileName, string(viewName)); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) DeleteZone(req pb.DeleteZoneReq) error {
	var names map[string][]byte
	var err error
	if names, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID); err != nil {
		return err
	}
	zoneName := names["name"]
	zoneFile := names["zonefile"]
	var out map[string][]byte
	if out, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID); err != nil {
		return err
	}
	viewName := out["name"]
	if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID + zonesPath + req.ZoneID); err != nil {
		return nil
	}
	if string(zoneFile) == "" { // zone file equal "" stands for the forward zone.
		if err := handler.rewriteNamedFile(); err != nil {
			return err
		}
		return nil
	}
	if err := handler.rndcDelZone(string(zoneName), string(zoneFile), string(viewName)); err != nil {
		return err
	}

	if err := os.Remove(handler.dnsConfPath + string(zoneFile)); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) CreateRR(req pb.CreateRRReq) error {
	rrsMap := map[string][]byte{}
	rrsMap["name"] = []byte(req.Name)
	rrsMap["type"] = []byte(req.Type)
	rrsMap["value"] = []byte(req.Value)
	rrsMap["TTL"] = []byte(req.TTL)
	if err := boltdb.GetDB().AddKVs(viewsPath+req.ViewID+zonesPath+req.ZoneID+rRsPath+req.RRID, rrsMap); err != nil {
		return err
	}
	var names map[string][]byte
	var err error
	if names, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID); err != nil {
		return err
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID); err != nil {
		return err
	}
	key := viewsmap["key"]
	var data string
	data = req.Name + "." + string(names["name"]) + " " + req.TTL + " IN " + req.Type + " " + req.Value
	if err := updateRR("key"+string(viewsmap["name"]), string(key), data, string(names["name"]), true); err != nil {
		return err
	}
	if err := handler.rndcDumpJNLFile(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateRR(req pb.UpdateRRReq) error {
	var rrsMap map[string][]byte
	var err error
	if rrsMap, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID + rRsPath + req.RRID); err != nil {
		return err
	}
	var names map[string][]byte
	if names, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID); err != nil {
		return err
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID); err != nil {
		return err
	}
	key := viewsmap["key"]
	oldData := string(rrsMap["name"]) + "." + string(names["name"]) + " " + string(rrsMap["TTL"]) + " IN " + string(rrsMap["type"]) + " " + string(rrsMap["value"])
	newData := req.Name + "." + string(names["name"]) + " " + req.TTL + " IN " + req.Type + " " + req.Value
	if err := updateRR("key"+string(viewsmap["name"]), string(key), oldData, string(names["name"]), false); err != nil {
		return err
	}
	if err := updateRR("key"+string(viewsmap["name"]), string(key), newData, string(names["name"]), true); err != nil {
		return err
	}
	//add the old data by rrupdate cause the delete function of the rrupdate had deleted all the rrs.
	var tables []string
	if tables, err = boltdb.GetDB().GetTables(viewsPath + req.ViewID + zonesPath + req.ZoneID + rRsEndPath); err != nil {
		return err
	}
	for _, t := range tables {
		if t == req.RRID {
			continue
		}
		var data map[string][]byte
		if data, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID + rRsPath + t); err != nil {
			return err
		}
		if req.Type != string(data["type"]) {
			continue
		}
		var updateData string
		updateData = string(data["name"]) + "." + string(names["name"]) + " " + string(data["TTL"]) + " IN " + string(data["type"]) + " " + string(data["value"])
		if err := updateRR("key"+string(viewsmap["name"]), string(key), updateData, string(names["name"]), true); err != nil {
			return err
		}
	}
	rrsMap["name"] = []byte(req.Name)
	rrsMap["type"] = []byte(req.Type)
	rrsMap["value"] = []byte(req.Value)
	rrsMap["TTL"] = []byte(req.TTL)
	if err := boltdb.GetDB().UpdateKVs(viewsPath+req.ViewID+zonesPath+req.ZoneID+rRsPath+req.RRID, rrsMap); err != nil {
		return err
	}
	if err := handler.rndcDumpJNLFile(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) DeleteRR(req pb.DeleteRRReq) error {
	var rrsMap map[string][]byte
	var err error
	if rrsMap, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID + rRsPath + req.RRID); err != nil {
		return err
	}
	var names map[string][]byte
	if names, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID); err != nil {
		return err
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID); err != nil {
		return err
	}
	key := viewsmap["key"]
	rrData := string(rrsMap["name"]) + "." + string(names["name"]) + " " + string(rrsMap["TTL"]) + " IN " + string(rrsMap["type"]) + " " + string(rrsMap["value"])
	if err := updateRR("key"+string(viewsmap["name"]), string(key), rrData, string(names["name"]), false); err != nil { //string(rrData[:])
		return err
	}
	if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID + zonesPath + req.ZoneID + rRsPath + req.RRID); err != nil {
		return err
	}
	//add the old data by rrupdate cause the delete function of the rrupdate had deleted all the rrs.
	var tables []string
	if tables, err = boltdb.GetDB().GetTables(viewsPath + req.ViewID + zonesPath + req.ZoneID + rRsEndPath); err != nil {
		return err
	}
	for _, t := range tables {
		var data map[string][]byte
		if data, err = boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID + rRsPath + t); err != nil {
			return err
		}
		if string(rrsMap["type"]) != string(data["type"]) {
			continue
		}
		var updateData string
		updateData = string(data["name"]) + "." + string(names["name"]) + " " + string(data["TTL"]) + " IN " + string(data["type"]) + " " + string(data["value"])
		if err := updateRR("key"+string(viewsmap["name"]), string(key), updateData, string(names["name"]), true); err != nil {
			return err
		}
	}
	if err := handler.rndcDumpJNLFile(); err != nil {
		return err
	}

	return nil
}

func (h *DNSHandler) Close() {
	boltdb.GetDB().Close()
}

func (handler *DNSHandler) namedConfData() (namedData, error) {
	var err error
	data := namedData{ConfigPath: handler.dnsConfPath}
	//get all the acl names.
	var aclTables []string
	aclTables, err = boltdb.GetDB().GetTables(aCLsEndPath)
	if err != nil {
		return data, err
	}
	for _, aclid := range aclTables {
		var nameKVs map[string][]byte
		nameKVs, err = boltdb.GetDB().GetTableKVs(aCLsPath + aclid)
		if err != nil {
			return data, err
		}
		if aclid != "1" && aclid != "2" {
			data.ACLNames = append(data.ACLNames, string(nameKVs["name"]))
		}
	}
	//get the ip black hole data
	var tables []string
	tables, err = boltdb.GetDB().GetTables(ipBlackHoleEndPath)
	if err != nil {
		return data, err
	}
	if len(tables) > 0 {
		var tmp ipBlackHole
		data.IPBlackHole = &tmp
	}
	for _, id := range tables {
		var blackholeKVs map[string][]byte
		blackholeKVs, err = boltdb.GetDB().GetTableKVs(ipBlackHolePath + id)
		if err != nil {
			return data, err
		}
		aclid := blackholeKVs["aclid"]
		var aclNameKVs map[string][]byte
		aclNameKVs, err = boltdb.GetDB().GetTableKVs(aCLsPath + string(aclid))
		if err != nil {
			return data, err
		}
		data.IPBlackHole.ACLNames = append(data.IPBlackHole.ACLNames, string(aclNameKVs["name"]))
	}
	//get recursive concurrency data
	var concuKVs map[string][]byte
	concuKVs, err = boltdb.GetDB().GetTableKVs(recurConcurEndPath)
	if err != nil {
		return data, err
	}
	if len(concuKVs) > 0 {
		var tmp recursiveConcurrent
		data.Concu = &tmp
		if string(concuKVs["recursiveclients"]) != "" {
			var num int
			if num, err = strconv.Atoi(string(concuKVs["recursiveclients"])); err != nil {
				return data, err
			}
			data.Concu.RecursiveClients = &num
		}
		if string(concuKVs["fetchesperzone"]) != "" {
			var num int
			if num, err = strconv.Atoi(string(concuKVs["fetchesperzone"])); err != nil {
				return data, err
			}
			data.Concu.FetchesPerZone = &num
		}
	}
	//get the sortlist
	var sortkvs map[string][]byte
	sortkvs, err = boltdb.GetDB().GetTableKVs(sortListEndPath)
	if err != nil {
		return data, err
	}
	aclid := string(sortkvs["next"])
	for {
		if aclid == "" {
			break
		}
		//get acl name
		var aclName map[string][]byte
		aclName, err = boltdb.GetDB().GetTableKVs(aCLsPath + aclid)
		if err != nil {
			return data, err
		}
		data.SortList = append(data.SortList, string(aclName["name"]))
		var kvs map[string][]byte
		kvs, err = boltdb.GetDB().GetTableKVs(sortListPath + aclid)
		if err != nil {
			return data, err
		}
		aclid = string(kvs["next"])
	}
	//get the data under the views
	var kvs map[string][]byte
	kvs, err = boltdb.GetDB().GetTableKVs(viewsEndPath)
	if err != nil {
		return data, err
	}
	if len(kvs) == 0 {
		return data, nil
	}
	var keys []string
	for k := range kvs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, priority := range keys {
		viewid := kvs[priority]
		nameKvs, err := boltdb.GetDB().GetTableKVs(viewsPath + string(viewid))
		if err != nil {
			return data, err
		}
		viewName := nameKvs["name"]
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(viewsPath + string(viewid) + aCLsEndPath); err != nil {
			return data, err
		}
		var aCLs []ACL
		for _, aCLid := range tables {
			aCLNames, err := boltdb.GetDB().GetTableKVs(aCLsPath + aCLid)
			if err != nil {
			}
			aCLName := aCLNames["name"]
			ipsMap, err := boltdb.GetDB().GetTableKVs(aCLsPath + aCLid + iPsEndPath)
			if err != nil {
				return data, err
			}
			var ips []string
			for _, ip := range ipsMap {
				ips = append(ips, string(ip))
			}
			aCL := ACL{Name: string(aCLName)}
			aCLs = append(aCLs, aCL)
		}
		view := View{Name: string(viewName), ACLs: aCLs}
		//get the redirect data
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(viewsPath + string(viewid) + redirectEndPath); err != nil {
			return data, err
		}
		if len(tables) > 0 {
			var tmp redierct
			view.Redirect = &tmp
		}
		for _, rrid := range tables {
			rrMap, err := boltdb.GetDB().GetTableKVs(viewsPath + string(viewid) + redirectPath + rrid)
			if err != nil {
				return data, err
			}
			var tmp RR
			tmp.Name = string(rrMap["name"])
			tmp.TTL = string(rrMap["TTL"])
			tmp.Type = string(rrMap["type"])
			tmp.Value = string(rrMap["value"])
			view.Redirect.RRs = append(view.Redirect.RRs, tmp)
		}
		//get the RPZ data
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(viewsPath + string(viewid) + rpzEndPath); err != nil {
			return data, err
		}
		if len(tables) > 0 {
			var tmp rpz
			view.RPZ = &tmp
		}
		for _, rrid := range tables {
			rrMap, err := boltdb.GetDB().GetTableKVs(viewsPath + string(viewid) + rpzPath + rrid)
			if err != nil {
				return data, err
			}
			var tmp RR
			tmp.Name = string(rrMap["name"])
			tmp.TTL = string(rrMap["TTL"])
			tmp.Type = string(rrMap["type"])
			tmp.Value = string(rrMap["value"])
			view.RPZ.RRs = append(view.RPZ.RRs, tmp)
		}
		//get the dns64s data
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(viewsPath + string(viewid) + dns64sEndPath); err != nil {
			return data, err
		}
		for _, dns64id := range tables {
			dns64Map, err := boltdb.GetDB().GetTableKVs(viewsPath + string(viewid) + dns64sPath + dns64id)
			if err != nil {
				return data, err
			}
			var tmp dns64
			tmp.Prefix = string(dns64Map["prefix"])
			aCLNames, err := boltdb.GetDB().GetTableKVs(aCLsPath + string(dns64Map["clientacl"]))
			if err != nil {
				return data, err
			}
			tmp.ClientACLName = string(aCLNames["name"])
			aCLNames, err = boltdb.GetDB().GetTableKVs(aCLsPath + string(dns64Map["aaddress"]))
			if err != nil {
				return data, err
			}
			tmp.AAddressACLName = string(aCLNames["name"])
			view.DNS64s = append(view.DNS64s, tmp)
		}
		//get the forward data under the zone
		tables = tables[0:0]
		if tables, err = boltdb.GetDB().GetTables(viewsPath + string(viewid) + zonesEndPath); err != nil {
			return data, err
		}
		for _, zoneid := range tables {
			zoneMap, err := boltdb.GetDB().GetTableKVs(viewsPath + string(viewid) + zonesPath + zoneid)
			if err != nil {
				return data, err
			}
			forwardMap, err := boltdb.GetDB().GetTableKVs(viewsPath + string(viewid) + zonesPath + zoneid + forwardEndPath)
			if err != nil {
				return data, err
			}
			if string(forwardMap["isforward"]) == "1" {
				forwarderMap, err := boltdb.GetDB().GetTableKVs(viewsPath + string(viewid) + zonesPath + zoneid + forwardPath + iPsEndPath)
				if err != nil {
					return data, err
				}
				var tmp Zone
				tmp.Name = string(zoneMap["name"])
				tmp.ForwardType = string(forwardMap["type"])
				var one forwarder
				tmp.Forwarder = &one
				for ip, _ := range forwarderMap {
					tmp.Forwarder.IPs = append(tmp.Forwarder.IPs, ip)
				}
				view.Zones = append(view.Zones, tmp)
			}
		}
		//get the base64 of the view's Key
		keyMap, err := boltdb.GetDB().GetTableKVs(viewsPath + string(viewid))
		if err != nil {
			return data, err
		}
		encodeString := base64.StdEncoding.EncodeToString(keyMap["key"])
		view.Key = encodeString

		data.Views = append(data.Views, view)
	}
	return data, nil
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
		aCLTables, err := boltdb.GetDB().GetTables(viewsPath + viewid + aCLsEndPath)
		if err != nil {
			return nil, err
		}
		for _, aCLid := range aCLTables {
			aCLNames, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid + aCLsPath + aCLid)
			if err != nil {
				return nil, err
			}
			aCLName := aCLNames["name"]
			ipsMap, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid + aCLsPath + aCLid + iPsEndPath)
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
		names, err := boltdb.GetDB().GetTableKVs(aCLsPath + aCLId)
		if err != nil {
			return nil, err
		}
		aCLName := names["name"]
		ipsMap, err := boltdb.GetDB().GetTableKVs(aCLsPath + aCLId + iPsEndPath)
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
		nameKvs, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid)
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		zoneTables, err := boltdb.GetDB().GetTables(viewsPath + viewid + zonesEndPath)
		if err != nil {
			return nil, err
		}
		var zones []Zone
		for _, zoneId := range zoneTables {
			Names, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid + zonesPath + zoneId)
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
		nameKvs, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid)
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		one.ViewName = string(viewName)
		rrsid, err := boltdb.GetDB().GetTables(viewsPath + viewid + redirectEndPath)
		if err != nil {
			return nil, err
		}
		for _, rrID := range rrsid {
			rrs, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid + redirectPath + rrID)
			if err != nil {
				return nil, err
			}
			name := rrs["name"]
			ttl := rrs["TTL"]
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
		nameKvs, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid)
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		one.ViewName = string(viewName)
		rrsid, err := boltdb.GetDB().GetTables(viewsPath + viewid + rpzEndPath)
		if err != nil {
			return nil, err
		}
		for _, rrID := range rrsid {
			rrs, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid + rpzPath + rrID)
			if err != nil {
				return nil, err
			}
			name := rrs["name"]
			ttl := rrs["TTL"]
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
		nameKvs, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid)
		if err != nil {
			return nil, err
		}
		viewName := nameKvs["name"]
		zoneTables, err := boltdb.GetDB().GetTables(viewsPath + viewid + zonesEndPath)
		if err != nil {
			return nil, err
		}
		for _, zoneID := range zoneTables {
			var rrs []RR
			names, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid + zonesPath + zoneID)
			if err != nil {
				return nil, err
			}
			zoneName := names["name"]
			zoneFile := names["zonefile"]
			if string(zoneFile) == "" {
				continue
			}
			rrTables, err := boltdb.GetDB().GetTables(viewsPath + viewid + zonesPath + zoneID + rRsEndPath)
			if err != nil {
				return nil, err
			}
			for _, rrID := range rrTables {
				datas, err := boltdb.GetDB().GetTableKVs(viewsPath + viewid + zonesPath + zoneID + rRsPath + rrID)
				if err != nil {
					return nil, err
				}
				rr := RR{Name: string(datas["name"]), TTL: string(datas["TTL"]), Type: string(datas["type"]), Value: string(datas["value"])}
				rrs = append(rrs, rr)
			}
			one := zoneData{ViewName: string(viewName), Name: string(zoneName), ZoneFile: string(zoneFile), RRs: rrs}
			zonesData = append(zonesData, one)
		}
	}
	return zonesData, nil
}

func (handler *DNSHandler) rewriteNamedFile() error {
	var namedConfData namedData
	var err error
	if namedConfData, err = handler.namedConfData(); err != nil {
		return err
	}
	if err != nil {
		return err
	}
	buffer := new(bytes.Buffer)
	if err = handler.tpl.ExecuteTemplate(buffer, namedTpl, namedConfData); err != nil {
		return err
	}
	if err := ioutil.WriteFile(handler.dnsConfPath+mainConfName, buffer.Bytes(), 0644); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rewriteZonesFile() error {
	zonesData, err := handler.zonesData()
	if err != nil {
		return err
	}
	for _, zoneData := range zonesData {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, zoneTpl, zoneData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(handler.dnsConfPath+zoneData.ZoneFile, buf.Bytes(), 0644); err != nil {
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
	for _, aCL := range aCLs {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, aCLTpl, aCL); err != nil {
			return err
		}
		if aCL.Name == "any" || aCL.Name == "none" {
			continue
		}
		if err := ioutil.WriteFile(handler.dnsConfPath+aCL.Name+".conf", buf.Bytes(), 0644); err != nil {
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
	for _, nzfData := range nzfsData {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, nzfTpl, nzfData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(handler.dnsConfPath+nzfData.ViewName+".nzf", buf.Bytes(), 0644); err != nil {
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
	for _, redirectionData := range redirectionsData {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, redirectTpl, redirectionData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(handler.dnsConfPath+"redirection/redirect_"+redirectionData.ViewName, buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) rewriteRPZFile() error {
	redirectionsData, err := handler.rpzData()
	if err != nil {
		return err
	}
	for _, redirectionData := range redirectionsData {
		buf := new(bytes.Buffer)
		if err = handler.tpl.ExecuteTemplate(buf, rpzTpl, redirectionData); err != nil {
			return err
		}
		if err := ioutil.WriteFile(handler.dnsConfPath+"redirection/rpz_"+redirectionData.ViewName, buf.Bytes(), 0644); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) addPriority(pri int, viewid string) error {
	var kvs map[string][]byte
	var err error
	if kvs, err = boltdb.GetDB().GetTableKVs(viewsEndPath); err != nil {
		return err
	}
	if pri > len(kvs)+1 {
		pri = len(kvs)
	} else if pri < 1 {
		pri = 1
	}
	i := len(kvs)
	for i >= pri {
		kvs[strconv.Itoa(i+1)] = kvs[strconv.Itoa(i)]
		i--
	}
	kvs[strconv.Itoa(pri)] = []byte(viewid)
	addKVs := map[string][]byte{strconv.Itoa(len(kvs)): kvs[strconv.Itoa(len(kvs))]}
	if err := boltdb.GetDB().AddKVs(viewsEndPath, addKVs); err != nil {
		return err
	}
	delete(kvs, strconv.Itoa(len(kvs)))
	if err := boltdb.GetDB().UpdateKVs(viewsEndPath, kvs); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) updatePriority(pri int, viewid string) error {
	var kvs map[string][]byte
	var err error
	if kvs, err = boltdb.GetDB().GetTableKVs(viewsEndPath); err != nil {
		return err
	}
	if pri > len(kvs) {
		pri = len(kvs)
	} else if pri < 1 {
		pri = 1
	}
	var oriIndex string
	var v []byte
	for oriIndex, v = range kvs {
		if string(v) == viewid {
			break
		}
	}
	key := strconv.Itoa(pri)
	if oriIndex != key {
		tmp := kvs[key]
		kvs[key] = kvs[oriIndex]
		kvs[oriIndex] = tmp
		if err := boltdb.GetDB().UpdateKVs(viewsEndPath, kvs); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) deletePriority(viewID string) error {
	//query priority
	kvs, err := boltdb.GetDB().GetTableKVs(viewsEndPath)
	if err != nil {
		return err
	}
	//delete priority
	var k string
	var v []byte
	for k, v = range kvs {
		if string(v) == viewID {
			break
		}
	}
	if err = boltdb.GetDB().DeleteKVs(viewsEndPath, []string{strconv.Itoa(len(kvs))}); err != nil {
		return err
	}
	delete(kvs, k)
	var keys []string
	for k := range kvs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	i := 1
	updateKVs := map[string][]byte{}
	for _, pri := range keys {
		updateKVs[strconv.Itoa(i)] = kvs[pri]
		i++
	}
	if err = boltdb.GetDB().UpdateKVs(viewsEndPath, updateKVs); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) aCLsFromTopPath(aCLids []string) ([]ACL, error) {
	var aCLs []ACL
	for _, aCLid := range aCLids {
		names, err := boltdb.GetDB().GetTableKVs(aCLsPath + aCLid)
		if err != nil {
			return nil, err
		}
		name := names["name"]
		var ipsmap map[string][]byte
		ipsmap, err = boltdb.GetDB().GetTableKVs(aCLsPath + aCLid + iPsEndPath)
		if err != nil {
			return nil, err
		}
		var ips []string
		for ip, _ := range ipsmap {
			ips = append(ips, ip)
		}
		aCL := ACL{Name: string(name), IPs: ips, ID: aCLid}
		aCLs = append(aCLs, aCL)
	}
	return aCLs, nil
}

func (handler *DNSHandler) rndcReconfig() error {
	//update bind
	var para1 string = "-c" + handler.dnsConfPath + "rndc.conf"
	var para2 string = "-s" + "localhost"
	var para3 string = "-p" + rndcPort
	var para4 string = "reconfig"
	if _, err := shell.Shell(handler.dnsConfPath+"rndc", para1, para2, para3, para4); err != nil {
		return fmt.Errorf("rndc reconfig error, %w", err)
	}
	return nil
}
func (handler *DNSHandler) rndcAddZone(name string, zoneFile string, viewName string) error {
	//update bind
	var para1 string = "-c" + handler.dnsConfPath + "rndc.conf"
	var para2 string = "-s" + "localhost"
	var para3 string = "-p" + rndcPort
	var para4 string = "addzone " + name + " in " + viewName + " { type master; file \"" + zoneFile + "\";};"
	if _, err := shell.Shell(handler.dnsConfPath+"rndc", para1, para2, para3, para4); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rndcDelZone(name string, zoneFile string, viewName string) error {
	//update bind
	var para1 string = "-c" + handler.dnsConfPath + "rndc.conf"
	var para2 string = "-s" + "localhost"
	var para3 string = "-p" + rndcPort
	var para4 string = "delzone " + name + " in " + viewName
	if _, err := shell.Shell(handler.dnsConfPath+"rndc", para1, para2, para3, para4); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) rndcDumpJNLFile() error {
	//update bind
	var para1 string = "-c" + handler.dnsConfPath + "rndc.conf"
	var para2 string = "-s" + "localhost"
	var para3 string = "-p" + rndcPort
	var para4 string = "sync"
	var para5 string = "-clean"
	if _, err := shell.Shell(handler.dnsConfPath+"rndc", para1, para2, para3, para4, para5); err != nil {
		panic(err)
		return err
	}
	return nil
}

func (handler *DNSHandler) keepDNSAlive() {
	for {
		select {
		case <-handler.ticker.C:
			if _, err := os.Stat("/root/bindtest/" + "named.pid"); err == nil {
				continue
			}
			req := pb.DNSStartReq{}
			handler.Start(req)
		case <-handler.quit:
			return
		}
	}
}
func (handler *DNSHandler) UpdateDefaultForward(req pb.UpdateDefaultForwardReq) error {
	//delete the old data
	_, err := boltdb.GetDB().GetTableKVs(forwardPath + iPsEndPath)
	if err != nil {
		return err
	}
	if err := boltdb.GetDB().DeleteTable(forwardPath + iPsEndPath); err != nil {
		return err
	}
	//input the new data
	values := map[string][]byte{}
	kvs, err := boltdb.GetDB().GetTableKVs(forwardEndPath)
	if err != nil {
		return err
	}
	values["type"] = []byte(req.Type)
	values["isforward"] = []byte("1")
	if len(kvs) == 0 {
		if err := boltdb.GetDB().AddKVs(forwardEndPath, values); err != nil {
			return err
		}
	} else {
		if err := boltdb.GetDB().UpdateKVs(forwardEndPath, values); err != nil {
			return err
		}
	}
	ipsKVs := map[string][]byte{}
	for _, ip := range req.IPs {
		ipsKVs[ip] = []byte("")
	}
	if err := boltdb.GetDB().AddKVs(forwardPath+iPsEndPath, ipsKVs); err != nil {
		return err
	}
	//reform the named.conf file
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//involve rndc reconfig, update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) DeleteDefaultForward(req pb.DeleteDefaultForwardReq) error {
	//delete the old data
	if err := boltdb.GetDB().DeleteTable(forwardPath + iPsEndPath); err != nil {
		return err
	}
	//update the isforward be 0
	values := map[string][]byte{}
	values["isforward"] = []byte("0")
	if err := boltdb.GetDB().UpdateKVs(forwardEndPath, values); err != nil {
		return err
	}
	//reform the named.conf file
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) UpdateForward(req pb.UpdateForwardReq) error {
	//delete the old data
	_, err := boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID + forwardPath + iPsEndPath)
	if err != nil {
		return err
	}
	if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID + zonesPath + req.ZoneID + forwardPath + iPsEndPath); err != nil {
		return err
	}
	//input the new data
	values := map[string][]byte{}
	kvs, err := boltdb.GetDB().GetTableKVs(viewsPath + req.ViewID + zonesPath + req.ZoneID + forwardEndPath)
	if err != nil {
		return err
	}
	values["type"] = []byte(req.Type)
	values["isforward"] = []byte("1")
	if len(kvs) == 0 {
		if err := boltdb.GetDB().AddKVs(viewsPath+req.ViewID+zonesPath+req.ZoneID+forwardEndPath, values); err != nil {
			return err
		}
	} else {
		if err := boltdb.GetDB().UpdateKVs(viewsPath+req.ViewID+zonesPath+req.ZoneID+forwardEndPath, values); err != nil {
			return err
		}
	}
	ipsKVs := map[string][]byte{}
	for _, ip := range req.IPs {
		ipsKVs[ip] = []byte("")
	}
	if err := boltdb.GetDB().AddKVs(viewsPath+req.ViewID+zonesPath+req.ZoneID+forwardPath+iPsEndPath, ipsKVs); err != nil {
		return err
	}
	//reform the named.conf file
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//involve rndc reconfig, update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	return nil
}
func (handler *DNSHandler) DeleteForward(req pb.DeleteForwardReq) error {
	//delete the old data
	if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID + zonesPath + req.ZoneID + forwardPath + iPsEndPath); err != nil {
		return err
	}
	//update the isforward be 0
	values := map[string][]byte{}
	values["isforward"] = []byte("0")
	if err := boltdb.GetDB().UpdateKVs(viewsPath+req.ViewID+zonesPath+req.ZoneID+forwardEndPath, values); err != nil {
		return err
	}
	//reform the named.conf file
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) CreateRedirection(req pb.CreateRedirectionReq) error {
	rrMap := map[string][]byte{}
	rrMap["name"] = []byte(req.Name)
	rrMap["type"] = []byte(req.DataType)
	rrMap["value"] = []byte(req.Value)
	rrMap["TTL"] = []byte(req.TTL)
	if req.RedirectType == "redirect" {
		//input the data into the database.
		if err := boltdb.GetDB().AddKVs(viewsPath+req.ViewID+redirectPath+req.ID, rrMap); err != nil {
			return err
		}
		//create the redirection/redirect_$viewname file,use the template.
		if err := handler.rewriteRedirectFile(); err != nil {
			return err
		}
	} else if req.RedirectType == "rpz" {
		//input the data into the database.
		if err := boltdb.GetDB().AddKVs(viewsPath+req.ViewID+rpzPath+req.ID, rrMap); err != nil {
			return err
		}
		//create the redirection/rpz_$viewname file
		if err := handler.rewriteRPZFile(); err != nil {
			return err
		}
	}
	//reform the named.conf file
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) UpdateRedirection(req pb.UpdateRedirectionReq) error {
	//update the data in the database
	rrMap := map[string][]byte{}
	rrMap["name"] = []byte(req.Name)
	rrMap["type"] = []byte(req.DataType)
	rrMap["value"] = []byte(req.Value)
	rrMap["TTL"] = []byte(req.TTL)
	if req.RedirectType == "redirect" {
		//input the data into the database.
		if err := boltdb.GetDB().UpdateKVs(viewsPath+req.ViewID+redirectPath+req.ID, rrMap); err != nil {
			return err
		}
		//rewrite the redirection/redirect_$viewname file,use the template.
		if err := handler.rewriteRedirectFile(); err != nil {
			return err
		}
	} else if req.RedirectType == "rpz" {
		//input the data into the database.
		if err := boltdb.GetDB().UpdateKVs(viewsPath+req.ViewID+rpzPath+req.ID, rrMap); err != nil {
			return err
		}
		//rewrite the redirection/rpz_$viewname file
		if err := handler.rewriteRPZFile(); err != nil {
			return err
		}
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) DeleteRedirection(req pb.DeleteRedirectionReq) error {
	//delete the data in the database
	if req.RedirectType == "redirect" {
		//input the data into the database.
		if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID + redirectPath + req.ID); err != nil {
			return err
		}
		//rewrite the redirection/redirect_$viewname file,use the template.
		if err := handler.rewriteRedirectFile(); err != nil {
			return err
		}
	} else if req.RedirectType == "rpz" {
		//input the data into the database.
		if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID + rpzPath + req.ID); err != nil {
			return err
		}
		//rewrite the redirection/rpz_$viewname file
		if err := handler.rewriteRPZFile(); err != nil {
			return err
		}
	}
	//reform the named.conf file
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) CreateDefaultDNS64(req pb.CreateDefaultDNS64Req) error {
	//input the data into the data base.
	kvs := map[string][]byte{}
	kvs["prefix"] = []byte(req.Prefix)
	kvs["clientacl"] = []byte(req.ClientACL)
	kvs["aaddress"] = []byte(req.AAddress)
	if err := boltdb.GetDB().AddKVs(dns64sPath+req.ID, kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) UpdateDefaultDNS64(req pb.UpdateDefaultDNS64Req) error {
	//input the data into the data base.
	kvs := map[string][]byte{}
	kvs["prefix"] = []byte(req.Prefix)
	kvs["clientacl"] = []byte(req.ClientACL)
	kvs["aaddress"] = []byte(req.AAddress)
	if err := boltdb.GetDB().UpdateKVs(dns64sPath+req.ID, kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) DeleteDefaultDNS64(req pb.DeleteDefaultDNS64Req) error {
	//delete the data in the data base.drop the leaf table.
	if err := boltdb.GetDB().DeleteTable(dns64sPath + req.ID); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) CreateDNS64(req pb.CreateDNS64Req) error {
	//input the data into the data base.
	kvs := map[string][]byte{}
	kvs["prefix"] = []byte(req.Prefix)
	kvs["clientacl"] = []byte(req.ClientACL)
	kvs["aaddress"] = []byte(req.AAddress)
	if err := boltdb.GetDB().AddKVs(viewsPath+req.ViewID+dns64sPath+req.ID, kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) UpdateDNS64(req pb.UpdateDNS64Req) error {
	//input the data into the data base.
	kvs := map[string][]byte{}
	kvs["prefix"] = []byte(req.Prefix)
	kvs["clientacl"] = []byte(req.ClientACL)
	kvs["aaddress"] = []byte(req.AAddress)
	if err := boltdb.GetDB().UpdateKVs(viewsPath+req.ViewID+dns64sPath+req.ID, kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	return nil
}
func (handler *DNSHandler) DeleteDNS64(req pb.DeleteDNS64Req) error {
	//delete the data in the data base.drop the leaf table.
	if err := boltdb.GetDB().DeleteTable(viewsPath + req.ViewID + dns64sPath + req.ID); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) CreateIPBlackHole(req pb.CreateIPBlackHoleReq) error {
	//input the data into the data base.
	kvs := map[string][]byte{}
	kvs["aclid"] = []byte(req.ACL)
	if err := boltdb.GetDB().AddKVs(ipBlackHolePath+req.ID, kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) UpdateIPBlackHole(req pb.UpdateIPBlackHoleReq) error {
	//update the data into the data base.
	kvs := map[string][]byte{}
	kvs["aclid"] = []byte(req.ACL)
	if err := boltdb.GetDB().UpdateKVs(ipBlackHolePath+req.ID, kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) DeleteIPBlackHole(req pb.DeleteIPBlackHoleReq) error {
	//delete the data into the data base.
	boltdb.GetDB().DeleteTable(ipBlackHolePath + req.ID)
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) UpdateRecursiveConcurrent(req pb.UpdateRecurConcuReq) error {
	//update the data in the database;
	conKVs, err := boltdb.GetDB().GetTableKVs(recurConcurEndPath)
	if err != nil {
		return err
	}
	kvs := map[string][]byte{}
	kvs["recursiveclients"] = []byte(req.RecursiveClients)
	kvs["fetchesperzone"] = []byte(req.FetchesPerZone)
	if len(conKVs) == 0 {
		if err := boltdb.GetDB().AddKVs(recurConcurEndPath, kvs); err != nil {
			return err
		}
	} else {
		if err := boltdb.GetDB().UpdateKVs(recurConcurEndPath, kvs); err != nil {
			return err
		}
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	return nil
}
func (handler *DNSHandler) CreateSortList(req pb.CreateSortListReq) error {
	//input the data into the data base.
	kvs := map[string][]byte{}
	if len(req.ACLs) == 0 {
		return nil
	}
	kvs["next"] = []byte(req.ACLs[0])
	if err := boltdb.GetDB().AddKVs(sortListEndPath, kvs); err != nil {
		return err
	}
	for k, v := range req.ACLs {
		kvs := map[string][]byte{}
		if k == 0 {
			kvs["prev"] = []byte("")
		} else {
			kvs["prev"] = []byte(req.ACLs[k-1])
		}
		if k == len(req.ACLs)-1 {
			kvs["next"] = []byte("")
		} else {
			kvs["next"] = []byte(req.ACLs[k+1])
		}
		if err := boltdb.GetDB().AddKVs(sortListPath+v, kvs); err != nil {
			return err
		}
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) UpdateSortList(req pb.UpdateSortListReq) error {
	//update the data into the data base.
	delReq := pb.DeleteSortListReq{ACLs: req.ACLs}
	handler.DeleteSortList(delReq)
	createReq := pb.CreateSortListReq{ACLs: req.ACLs}
	handler.CreateSortList(createReq)
	return nil
}
func (handler *DNSHandler) DeleteSortList(req pb.DeleteSortListReq) error {
	//delete the data in the data base.
	var acls []string
	var err error
	acls, err = boltdb.GetDB().GetTables(sortListEndPath)
	if err != nil {
		panic(err)
	}
	for _, v := range acls {
		boltdb.GetDB().DeleteTable(sortListPath + v)
	}
	boltdb.GetDB().DeleteTable(sortListEndPath)
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func updateRR(key string, secret string, rr string, zone string, isAdd bool) error {
	if len(rr) >= 2 {
		if rr[0] == '@' && rr[1] == '.' {
			rr = rr[2:]
		}
	}
	serverAddr, err := net.ResolveUDPAddr("udp", dnsServer)
	if err != nil {
		return err
	}

	conn, err := net.DialUDP("udp", nil, serverAddr)
	if err != nil {
		return err
	}
	defer conn.Close()

	zone_, err := g53.NewName(zone, false)
	if err != nil {
		return err
	}

	rrset, err := g53.RRsetFromString(rr)
	if err != nil {
		return err
	}

	msg := g53.MakeUpdate(zone_)
	if isAdd {
		msg.UpdateAddRRset(rrset)
	} else {
		msg.UpdateRemoveRRset(rrset)
	}
	msg.Header.Id = 1200

	secret = base64.StdEncoding.EncodeToString([]byte(secret))
	tsig, err := g53.NewTSIG(key, secret, "hmac-md5")
	if err != nil {
		return err
	}
	msg.SetTSIG(tsig)
	msg.RecalculateSectionRRCount()

	render := g53.NewMsgRender()
	msg.Rend(render)
	conn.Write(render.Data())

	answerBuffer := make([]byte, 1024)
	_, _, err = conn.ReadFromUDP(answerBuffer)
	if err != nil {
		return err
	}
	return nil
}
