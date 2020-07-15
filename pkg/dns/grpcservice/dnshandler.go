package grpcservice

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/shell"
	"github.com/zdnscloud/g53"

	"github.com/linkingthing/ddi-agent/pkg/boltdb"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/zdnscloud/cement/randomdata"
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
	forwardsPath       = "/forwards/"
	forwardsEndPath    = "/forwards"
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
	logPath            = "/log"
	namedTpl           = "named.tpl"
	namedNoRPZTpl      = "named_norpz.tpl"
	zoneTpl            = "zone.tpl"
	aCLTpl             = "acl.tpl"
	nzfTpl             = "nzf.tpl"
	redirectTpl        = "redirect.tpl"
	rpzTpl             = "rpz.tpl"
	rndcPort           = "953"
	checkPeriod        = 5
	dnsServer          = "localhost:53"
	masterType         = "master"
	forwardType        = "forward"
	anyACL             = "any"
	noneACL            = "none"
	defaultView        = "default"
	aclSuffix          = ".acl"
	zoneSuffix         = ".zone"
	nzfSuffix          = ".nzf"
	localZoneType      = "localzone"
	nxDomain           = "nxdomain"
	cnameType          = "CNAME"
	ptrType            = "PTR"
	logStatus          = "isopen"
	orderFirst         = "1"
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
	instance := &DNSHandler{dnsConfPath: filepath.Join(dnsConfPath), dBPath: filepath.Join(agentPath), tplPath: filepath.Join(dnsConfPath, "templates")}
	var err error
	instance.tpl, err = template.ParseFiles(filepath.Join(instance.tplPath, namedTpl))
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(filepath.Join(instance.tplPath, namedNoRPZTpl))
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(filepath.Join(instance.tplPath, zoneTpl))
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(filepath.Join(instance.tplPath, aCLTpl))
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(filepath.Join(instance.tplPath, nzfTpl))
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(filepath.Join(instance.tplPath, redirectTpl))
	if err != nil {
		return nil, err
	}

	instance.tpl, err = instance.tpl.ParseFiles(filepath.Join(instance.tplPath, rpzTpl))
	if err != nil {
		return nil, err
	}

	instance.ticker = time.NewTicker(checkPeriod * time.Second)
	instance.quit = make(chan int)
	//check wether the default acl "any" and "none" is exist.if not add the any and none into the database.
	anykvs := map[string][]byte{}
	anykvs, err = boltdb.GetDB().GetTableKVs(filepath.Join(aCLsEndPath, anyACL))
	if err != nil {
		return nil, err
	}

	if len(anykvs) == 0 {
		anykvs["name"] = []byte(anyACL)
		if err := boltdb.GetDB().AddKVs(filepath.Join(aCLsPath, anyACL), anykvs); err != nil {
			return nil, err
		}
	}

	nonekvs := map[string][]byte{}
	nonekvs, err = boltdb.GetDB().GetTableKVs(filepath.Join(aCLsEndPath, noneACL))
	if err != nil {
		return nil, err
	}

	if len(nonekvs) == 0 {
		nonekvs["name"] = []byte(noneACL)
		if err := boltdb.GetDB().AddKVs(filepath.Join(aCLsEndPath, noneACL), nonekvs); err != nil {
			return nil, err
		}
	}

	viewkvs := map[string][]byte{}
	viewkvs, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsPath, defaultView))
	if err != nil {
		return nil, err
	}

	if len(viewkvs) == 0 {
		if err := boltdb.GetDB().AddKVs(viewsEndPath, map[string][]byte{"next": []byte(defaultView)}); err != nil {
			return nil, err
		}

		viewkvs["name"] = []byte(defaultView)
		viewkvs["key"] = []byte(randomdata.RandString(12))
		viewkvs["next"] = []byte("")
		if err := boltdb.GetDB().AddKVs(filepath.Join(viewsPath, defaultView), viewkvs); err != nil {
			return nil, err
		}
	}

	var acls []string
	acls, err = boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, defaultView, aCLsEndPath, orderFirst))
	if err != nil {
		return nil, err
	}

	if len(acls) == 0 {
		if _, err := boltdb.GetDB().CreateOrGetTable(filepath.Join(viewsPath, defaultView, aCLsEndPath, orderFirst, anyACL)); err != nil {
			return nil, err
		}
	}
	if err := instance.StartDNS(pb.DNSStartReq{}); err != nil {
		log.Errorf("start dns fail:%s", err.Error())
	}
	return instance, nil
}

type namedData struct {
	ConfigPath  string
	ACLNames    []string
	Views       []View
	DNS64s      []dns64
	IPBlackHole *ipBlackHole
	IsLogOpen   bool
	SortList    []string
	Concu       *recursiveConcurrent
}

type forward struct {
	IPs []string
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
	IPs         []string
}

type zoneData struct {
	ViewName string
	Name     string
	ZoneFile string
	TTL      string
	RRs      []RR
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
	if err := handler.Start(req); err != nil {
		return err
	}
	go handler.keepDNSAlive()
	return nil

}
func (handler *DNSHandler) Start(req pb.DNSStartReq) error {
	if _, err := os.Stat(filepath.Join(handler.dnsConfPath, "named.pid")); err == nil {
		return nil
	}
	if err := handler.rewriteNamedFile(false); err != nil {
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

	if err := handler.rewriteRPZFile(true); err != nil {
		return err
	}

	var param string = "-c" + filepath.Join(handler.dnsConfPath, mainConfName)
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "named"), param); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) StopDNS() error {
	if _, err := os.Stat(filepath.Join(handler.dnsConfPath, "named.pid")); err != nil {
		return nil
	}
	var err error
	if _, err = shell.Shell(filepath.Join(handler.dnsConfPath, "rndc"), "stop"); err != nil {
		return err
	}
	handler.quit <- 1
	return nil
}

func (handler *DNSHandler) CreateACL(req pb.CreateACLReq) error {
	err := boltdb.GetDB().AddKVs(filepath.Join(aCLsEndPath, req.ID), map[string][]byte{"name": []byte(req.Name)})
	if err != nil {
		return err
	}
	values := map[string][]byte{}
	for _, ip := range req.IPs {
		values[ip] = []byte("")
	}
	if err := boltdb.GetDB().AddKVs(filepath.Join(aCLsEndPath, req.ID, iPsEndPath), values); err != nil {
		return err
	}
	aCLData := ACL{ID: req.ID, Name: req.Name, IPs: req.IPs}
	buffer := new(bytes.Buffer)
	if err = handler.tpl.ExecuteTemplate(buffer, aCLTpl, aCLData); err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, req.Name)+aclSuffix, buffer.Bytes(), 0644); err != nil {
		return err
	}
	if err := handler.rewriteNamedFile(false); err != nil {
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
	kvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(aCLsEndPath, req.ID))
	if err != nil {
		return err
	}
	if len(kvs) == 0 {
		return nil
	}
	name := kvs["name"]
	if err := os.Remove(filepath.Join(handler.dnsConfPath, string(name)) + aclSuffix); err != nil {
		return err
	}

	return boltdb.GetDB().DeleteTable(filepath.Join(aCLsEndPath, req.ID))
}

func (handler *DNSHandler) CreateView(req pb.CreateViewReq) error {
	ids, err := boltdb.GetDB().GetTables(viewsEndPath)
	if err != nil {
		return err
	}
	for _, id := range ids {
		if id == req.ViewID {
			return nil
		}
	}
	if err := handler.addPriority(int(req.Priority), req.ViewID, true); err != nil {
		return err
	}
	//create table viewid and put name into the db.
	namekvs := map[string][]byte{"name": []byte(req.ViewName), "key": []byte(randomdata.RandString(12))}
	if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID), namekvs); err != nil {
		return err
	}
	//insert the dns64
	if req.DNS64 != "" {
		if err := handler.CreateDNS64(pb.CreateDNS64Req{ID: req.ViewID, ViewID: req.ViewID, Prefix: req.DNS64, ClientACL: anyACL, AAddress: anyACL}); err != nil {
			return err
		}
	}
	//insert aCLIDs into viewid table
	for i, id := range req.ACLs {
		if _, err := boltdb.GetDB().CreateOrGetTable(filepath.Join(viewsEndPath, req.ViewID, aCLsPath, fmt.Sprintf("%d", i+1), id)); err != nil {
			return err
		}
	}
	if err := handler.rewriteNamedFile(false); err != nil {
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
		return fmt.Errorf("update priority err:%s", err.Error())
	}
	//delete the old acls for view
	if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, aCLsEndPath)); err != nil {
		return fmt.Errorf("delete acls err:%s", err.Error())
	}

	//add new aclids for aCL
	for i, id := range req.NewACLs {
		if _, err := boltdb.GetDB().CreateOrGetTable(filepath.Join(viewsEndPath, req.ViewID, aCLsPath, fmt.Sprintf("%d", i+1), id)); err != nil {
			return fmt.Errorf("when update view, after delete acls,insert acls err:%s", err.Error())
		}
	}
	//update the dns64
	if req.DNS64 != "" {
		tables, err := boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, req.ViewID, dns64sPath))
		if err != nil {
			return fmt.Errorf("get dns64 tables fail:%s", err.Error())
		}
		if len(tables) > 0 {
			if err := handler.UpdateDNS64(pb.UpdateDNS64Req{ID: req.ViewID, ViewID: req.ViewID, Prefix: req.DNS64, ClientACL: anyACL, AAddress: anyACL}); err != nil {
				return fmt.Errorf("update dns64 fail:%s", err.Error())
			}
		} else {
			if err := handler.CreateDNS64(pb.CreateDNS64Req{ID: req.ViewID, ViewID: req.ViewID, Prefix: req.DNS64, ClientACL: anyACL, AAddress: anyACL}); err != nil {
				return fmt.Errorf("create dns64 fail:%s", err.Error())
			}
		}
	} else {
		tables, err := boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, req.ViewID, dns64sPath))
		if err != nil {
			return fmt.Errorf("get dns64 tables fail:%s", err.Error())
		}
		if len(tables) > 0 {
			if err := handler.DeleteDNS64(pb.DeleteDNS64Req{ID: req.ViewID, ViewID: req.ViewID}); err != nil {
				return fmt.Errorf("when update view, delete dns64 err:%s", err.Error())
			}
		}
	}
	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("rewrite named.conf fail:%s", err.Error())
	}
	if err := handler.rewriteACLsFile(); err != nil {
		return fmt.Errorf("rewrite *.acl files fail:%s", err.Error())
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("rndc reconfig fail:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) DeleteView(req pb.DeleteViewReq) error {
	handler.deletePriority(req.ViewID)
	//delete table
	if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID)); err != nil {
		return err
	}
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rewriteZonesFile(); err != nil {
		return err
	}
	if err := handler.rewriteACLsFile(); err != nil {
		return err
	}
	if err := handler.rewriteNzfsFile(); err != nil {
		return err
	}
	if err := handler.rewriteRPZFile(false); err != nil {
		return err
	}
	if err := handler.rewriteRedirectFile(); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) CreateZone(req pb.CreateZoneReq) error {
	//put the zone into db
	names := map[string][]byte{}
	names["name"] = []byte(req.ZoneName)
	names["zonefile"] = []byte(req.ZoneFileName)
	names["ttl"] = []byte(strconv.Itoa(int(req.TTL)))
	if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID), names); err != nil {
		return err
	}
	//update file
	if err := handler.rewriteZonesFile(); err != nil {
		return err
	}
	out, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID))
	if err != nil {
		return err
	}
	viewName := out["name"]
	if err := handler.rndcAddZone(req.ZoneName, req.ZoneFileName, string(viewName)); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) CreateForwardZone(req pb.CreateForwardZoneReq) error {
	//put the zone into db
	names := map[string][]byte{}
	names["name"] = []byte(req.ZoneName)
	names["forwardtype"] = []byte(req.ForwardType)
	names["zonetype"] = []byte(forwardType)
	if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID), names); err != nil {
		return err
	}
	for _, id := range req.Forwards {
		if _, err := boltdb.GetDB().CreateOrGetTable(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, forwardsPath, id)); err != nil {
			return err
		}
	}
	//update file
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateZone(req pb.UpdateZoneReq) error {
	//put the zone into db
	names := map[string][]byte{}
	names["ttl"] = []byte(strconv.Itoa(int(req.TTL)))
	if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID), names); err != nil {
		return err
	}
	//update file
	if err := handler.rewriteZonesFile(); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateForwardZone(req pb.UpdateForwardZoneReq) error {
	//put the zone into db
	names := map[string][]byte{}
	names["forwardtype"] = []byte(req.ForwardType)
	if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID), names); err != nil {
		return err
	}
	if err := handler.updateForward(req.Forwards, req.ZoneID, req.ViewID); err != nil {
		return err
	}
	//update file
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
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

func (handler *DNSHandler) DeleteZone(req pb.DeleteZoneReq) error {
	var names map[string][]byte
	var err error
	if names, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID)); err != nil {
		return err
	}
	zoneName := names["name"]
	zoneFile := names["zonefile"]
	var out map[string][]byte
	if out, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID)); err != nil {
		return err
	}
	viewName := out["name"]
	if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID)); err != nil {
		return nil
	}
	if err := handler.rndcDelZone(string(zoneName), string(zoneFile), string(viewName)); err != nil {
		return err
	}

	if err := os.Remove(filepath.Join(handler.dnsConfPath, string(zoneFile))); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) DeleteForwardZone(req pb.DeleteForwardZoneReq) error {
	if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID)); err != nil {
		return nil
	}
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) CreateRR(req pb.CreateRRReq) error {
	formatCnameValue(&req.RData, req.Type)
	rrsMap := map[string][]byte{"name": []byte(req.Name), "type": []byte(req.Type), "value": []byte(req.RData), "ttl": []byte(req.TTL)}
	names, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID))
	if err != nil {
		return err
	}
	if req.TTL == "0" || req.TTL == "" {
		rrsMap["ttl"] = names["ttl"]
	}
	if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID), rrsMap); err != nil {
		return err
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID)); err != nil {
		return err
	}
	key := viewsmap["key"]
	var data string
	data = req.Name + "." + string(names["name"]) + " " + string(rrsMap["ttl"]) + " IN " + req.Type + " " + req.RData
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
	formatCnameValue(&req.RData, req.Type)
	if rrsMap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID)); err != nil {
		return err
	}
	var names map[string][]byte
	if names, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID)); err != nil {
		return err
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID)); err != nil {
		return err
	}
	key := viewsmap["key"]
	oldData := string(rrsMap["name"]) + "." + string(names["name"]) + " " + string(rrsMap["ttl"]) + " IN " + string(rrsMap["type"]) + " " + string(rrsMap["value"])
	newData := req.Name + "." + string(names["name"]) + " " + req.TTL + " IN " + req.Type + " " + req.RData
	if err := updateRR("key"+string(viewsmap["name"]), string(key), oldData, string(names["name"]), false); err != nil {
		return err
	}
	if err := updateRR("key"+string(viewsmap["name"]), string(key), newData, string(names["name"]), true); err != nil {
		return err
	}
	//add the old data by rrupdate cause the delete function of the rrupdate had deleted all the rrs.
	var tables []string
	if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsEndPath)); err != nil {
		return err
	}
	for _, t := range tables {
		if t == req.RRID {
			continue
		}
		var data map[string][]byte
		if data, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, t)); err != nil {
			return err
		}
		if req.Type != string(data["type"]) {
			continue
		}
		var updateData string
		updateData = string(data["name"]) + "." + string(names["name"]) + " " + string(data["ttl"]) + " IN " + string(data["type"]) + " " + string(data["value"])
		if err := updateRR("key"+string(viewsmap["name"]), string(key), updateData, string(names["name"]), true); err != nil {
			return err
		}
	}
	rrsMap["name"] = []byte(req.Name)
	rrsMap["type"] = []byte(req.Type)
	rrsMap["value"] = []byte(req.RData)
	rrsMap["ttl"] = []byte(req.TTL)
	if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID), rrsMap); err != nil {
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
	if rrsMap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID)); err != nil {
		return err
	}
	var names map[string][]byte
	if names, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID)); err != nil {
		return err
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID)); err != nil {
		return err
	}
	key := viewsmap["key"]
	rrData := string(rrsMap["name"]) + "." + string(names["name"]) + " " + string(rrsMap["ttl"]) + " IN " + string(rrsMap["type"]) + " " + string(rrsMap["value"])
	if err := updateRR("key"+string(viewsmap["name"]), string(key), rrData, string(names["name"]), false); err != nil { //string(rrData[:])
		return err
	}
	if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID)); err != nil {
		return err
	}
	//add the old data by rrupdate cause the delete function of the rrupdate had deleted all the rrs.
	var tables []string
	if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsEndPath)); err != nil {
		return err
	}
	for _, t := range tables {
		var data map[string][]byte
		if data, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, t)); err != nil {
			return err
		}
		if string(rrsMap["type"]) != string(data["type"]) {
			continue
		}
		var updateData string
		updateData = string(data["name"]) + "." + string(names["name"]) + " " + string(data["ttl"]) + " IN " + string(data["type"]) + " " + string(data["value"])
		if err := updateRR("key"+string(viewsmap["name"]), string(key), updateData, string(names["name"]), true); err != nil {
			return err
		}
	}
	if err := handler.rndcDumpJNLFile(); err != nil {
		return err
	}

	return nil
}

func formatCnameValue(rr *string, datatype string) {
	if datatype != cnameType {
		return
	}
	if ret := strings.HasSuffix(*rr, "."); !ret {
		*rr += "."
	}
	return
}

func formatDomain(name *string, datatype string, redirectType string) {
	if datatype == ptrType || redirectType == localZoneType {
		return
	}
	if ret := strings.HasSuffix(*name, "."); !ret {
		*name += "."
	}
	return
}

func (h *DNSHandler) Close() {
	boltdb.GetDB().Close()
}

func (handler *DNSHandler) namedConfData() (*namedData, error) {
	var err error
	data := namedData{ConfigPath: handler.dnsConfPath}
	//get all the acl names.
	var aclTables []string
	aclTables, err = boltdb.GetDB().GetTables(aCLsEndPath)
	if err != nil {
		return nil, err
	}
	for _, aclid := range aclTables {
		var nameKVs map[string][]byte
		nameKVs, err = boltdb.GetDB().GetTableKVs(filepath.Join(aCLsPath, aclid))
		if err != nil {
			return nil, err
		}
		if aclid != anyACL && aclid != noneACL {
			data.ACLNames = append(data.ACLNames, string(nameKVs["name"]))
		}
	}
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
	//get log data
	logKVs, err := boltdb.GetDB().GetTableKVs(logPath)
	if err != nil {
		return nil, err
	}
	if len(logKVs) > 0 {
		if logKVs[logStatus][0] == 1 {
			data.IsLogOpen = true
		} else {
			data.IsLogOpen = false
		}
	} else {
		data.IsLogOpen = true
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
	//get the data under the views
	var kvs map[string][]byte
	kvs, err = boltdb.GetDB().GetTableKVs(viewsEndPath)
	if err != nil {
		return nil, err
	}
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

func (handler *DNSHandler) keepDNSAlive() {
	defer handler.ticker.Stop()
	for {
		select {
		case <-handler.ticker.C:
			if _, err := os.Stat(filepath.Join(handler.dnsConfPath, "named.pid")); err == nil {
				continue
			}
			req := pb.DNSStartReq{}
			handler.Start(req)
		case <-handler.quit:
			return
		}
	}
}

func (handler *DNSHandler) CreateForward(req pb.CreateForwardReq) error {
	values := map[string][]byte{}
	for _, ip := range req.IPs {
		values[ip] = []byte("")
	}
	if err := boltdb.GetDB().AddKVs(filepath.Join(forwardsPath, req.ID), values); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateForward(req pb.UpdateForwardReq) error {
	//delete the old data
	deleteReq := pb.DeleteForwardReq{ID: req.ID}
	if err := handler.DeleteForward(deleteReq); err != nil {
		return err
	}
	createReq := pb.CreateForwardReq{ID: req.ID, IPs: req.IPs}
	if err := handler.CreateForward(createReq); err != nil {
		return err
	}
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) DeleteForward(req pb.DeleteForwardReq) error {
	//delete the old data
	if err := boltdb.GetDB().DeleteTable(filepath.Join(forwardsPath, req.ID)); err != nil {
		return err
	}
	return nil
}
func (handler *DNSHandler) CreateRedirection(req pb.CreateRedirectionReq) error {
	formatCnameValue(&req.Value, req.DataType)
	formatDomain(&req.Name, req.DataType, req.RedirectType)
	rrMap := map[string][]byte{}
	rrMap["name"] = []byte(req.Name)
	rrMap["type"] = []byte(req.DataType)
	rrMap["value"] = []byte(req.Value)
	rrMap["ttl"] = []byte(req.TTL)
	if req.RedirectType == nxDomain {
		//input the data into the database.
		if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, redirectPath, req.ID), rrMap); err != nil {
			return err
		}
		//create the redirection/redirect_$viewname file,use the template.
		if err := handler.rewriteRedirectFile(); err != nil {
			return err
		}
		//reform the named.conf file
		if err := handler.rewriteNamedFile(false); err != nil {
			return err
		}
		//update bind
		if err := handler.rndcReconfig(); err != nil {
			return err
		}
	} else if req.RedirectType == localZoneType {
		//input the data into the database.
		if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, rpzPath, req.ID), rrMap); err != nil {
			return err
		}
		//create the redirection/rpz_$viewname file
		if err := handler.rewriteRPZFile(false); err != nil {
			return err
		}
	}
	return nil
}
func (handler *DNSHandler) UpdateRedirection(req pb.UpdateRedirectionReq) error {
	formatCnameValue(&req.Value, req.DataType)
	formatDomain(&req.Name, req.DataType, req.RedirectType)
	//update the data in the database
	rrMap := map[string][]byte{}
	rrMap["name"] = []byte(req.Name)
	rrMap["type"] = []byte(req.DataType)
	rrMap["value"] = []byte(req.Value)
	rrMap["ttl"] = []byte(req.TTL)
	if req.RedirectType == nxDomain {
		if req.IsRedirectTypeChanged {
			if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, rpzPath, req.ID)); err != nil {
				return err
			}
			if err := handler.rewriteRPZFile(false); err != nil {
				return err
			}
			//input the data into the database.
			if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, redirectPath, req.ID), rrMap); err != nil {
				return err
			}
			tables, err := boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, req.ViewID, redirectPath))
			if err != nil {
				return err
			}
			if len(tables) == 1 {
				if err := handler.rewriteNamedFile(false); err != nil {
					return err
				}
			}
		} else {
			//update the data into the database.
			if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsEndPath, req.ViewID, redirectPath, req.ID), rrMap); err != nil {
				return err
			}
		}
		//rewrite the redirection/redirect_$viewname file,use the template.
		if err := handler.rewriteRedirectFile(); err != nil {
			return err
		}
		//update bind
		if err := handler.rndcReconfig(); err != nil {
			return err
		}
	} else if req.RedirectType == localZoneType {
		if req.IsRedirectTypeChanged {
			if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, redirectPath, req.ID)); err != nil {
				return err
			}
			if err := handler.rewriteRedirectFile(); err != nil {
				return err
			}
			//input the data into the database.
			if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, rpzPath, req.ID), rrMap); err != nil {
				return err
			}
		} else {
			//update the data into the database.
			if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsEndPath, req.ViewID, rpzPath, req.ID), rrMap); err != nil {
				return err
			}
		}
		//rewrite the redirection/rpz_$viewname file
		if err := handler.rewriteRPZFile(false); err != nil {
			return err
		}
	}
	return nil
}
func (handler *DNSHandler) DeleteRedirection(req pb.DeleteRedirectionReq) error {
	//delete the data in the database
	if req.RedirectType == nxDomain {
		//input the data into the database.
		if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, redirectPath, req.ID)); err != nil {
			return err
		}
		//rewrite the redirection/redirect_$viewname file,use the template.
		if err := handler.rewriteRedirectFile(); err != nil {
			return err
		}
		//rewrite the named.conf file
		if err := handler.rewriteNamedFile(false); err != nil {
			return err
		}
		//update bind
		if err := handler.rndcReconfig(); err != nil {
			return err
		}
	} else if req.RedirectType == localZoneType {
		//input the data into the database.
		if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, rpzPath, req.ID)); err != nil {
			return err
		}
		//rewrite the redirection/rpz_$viewname file
		if err := handler.rewriteRPZFile(false); err != nil {
			return err
		}
	}
	return nil
}

func (handler *DNSHandler) CreateDNS64(req pb.CreateDNS64Req) error {
	//input the data into the data base.
	kvs := map[string][]byte{}
	kvs["prefix"] = []byte(req.Prefix)
	kvs["clientacl"] = []byte(req.ClientACL)
	kvs["aaddress"] = []byte(req.AAddress)
	if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, dns64sPath, req.ID), kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
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
	oldkvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, dns64sPath, req.ID))
	if err != nil {
		return err
	}
	if len(oldkvs) == 0 {
		kvs := map[string][]byte{"prefix": []byte(req.Prefix), "clientacl": []byte(req.ClientACL), "aaddress": []byte(req.AAddress)}
		if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, dns64sPath, req.ID), kvs); err != nil {
			return err
		}
	} else {
		kvs := map[string][]byte{"prefix": []byte(req.Prefix), "clientacl": []byte(req.ClientACL), "aaddress": []byte(req.AAddress)}
		if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsEndPath, req.ViewID, dns64sPath, req.ID), kvs); err != nil {
			return err
		}
	}

	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
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
	if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, dns64sPath, req.ID)); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
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
	if err := boltdb.GetDB().AddKVs(filepath.Join(ipBlackHolePath, req.ID), kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
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
	if err := boltdb.GetDB().UpdateKVs(filepath.Join(ipBlackHolePath, req.ID), kvs); err != nil {
		return err
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
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
	boltdb.GetDB().DeleteTable(filepath.Join(ipBlackHolePath, req.ID))
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
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
	if err := handler.rewriteNamedFile(false); err != nil {
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
		if err := boltdb.GetDB().AddKVs(filepath.Join(sortListPath, v), kvs); err != nil {
			return err
		}
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
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
		return fmt.Errorf("delete sortlist err:%s", err.Error())
	}
	for _, v := range acls {
		boltdb.GetDB().DeleteTable(filepath.Join(sortListPath, v))
	}
	boltdb.GetDB().DeleteTable(sortListEndPath)
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	//update bind.
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateLog(req pb.UpdateLogReq) error {
	//update the data in the database;
	logKVs, err := boltdb.GetDB().GetTableKVs(logPath)
	if err != nil {
		return err
	}
	kvs := map[string][]byte{}
	if req.IsOpen {
		kvs[logStatus] = []byte{byte(1)}
	} else {
		kvs[logStatus] = []byte{byte(0)}
	}
	if len(logKVs) == 0 {
		if err := boltdb.GetDB().AddKVs(logPath, kvs); err != nil {
			return err
		}
	} else {
		if err := boltdb.GetDB().UpdateKVs(logPath, kvs); err != nil {
			return err
		}
	}
	//rewrite the named.conf file.
	if err := handler.rewriteNamedFile(false); err != nil {
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
