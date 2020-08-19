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
	mainConfName         = "named.conf"
	dBName               = "bind.db"
	viewsPath            = "/views/"
	viewsEndPath         = "/views"
	zonesPath            = "/zones/"
	zonesEndPath         = "/zones"
	aCLsPath             = "/acls/"
	aCLsEndPath          = "/acls"
	rRsEndPath           = "/rrs"
	rRsPath              = "/rrs/"
	iPsEndPath           = "/ips"
	forwardsPath         = "/forwards/"
	forwardsEndPath      = "/forwards"
	redirectPath         = "/redirect/"
	redirectEndPath      = "/redirect"
	rpzPath              = "/rpz/"
	rpzEndPath           = "/rpz"
	dns64sPath           = "/dns64s/"
	dns64sEndPath        = "/dns64s"
	ipBlackHolePath      = "/ipBlackHole/"
	ipBlackHoleEndPath   = "/ipBlackHole"
	recurConcurEndPath   = "/recurConcur"
	sortListPath         = "/sortList/"
	sortListEndPath      = "/sortList"
	urlRedirectsPath     = "/urlRedirects"
	namedTpl             = "named.tpl"
	namedNoRPZTpl        = "named_norpz.tpl"
	zoneTpl              = "zone.tpl"
	aCLTpl               = "acl.tpl"
	nzfTpl               = "nzf.tpl"
	redirectTpl          = "redirect.tpl"
	rpzTpl               = "rpz.tpl"
	nginxDefaultTpl      = "nginxdefault.tpl"
	rndcPort             = "953"
	checkPeriod          = 5
	dnsServer            = "localhost:53"
	masterType           = "master"
	forwardType          = "forward"
	anyACL               = "any"
	noneACL              = "none"
	defaultView          = "default"
	aclSuffix            = ".acl"
	zoneSuffix           = ".zone"
	nzfSuffix            = ".nzf"
	localZoneType        = "localzone"
	nxDomain             = "nxdomain"
	cnameType            = "CNAME"
	ptrType              = "PTR"
	orderFirst           = "1"
	domain               = "domain"
	url                  = "url"
	nginxDefaultConfFile = "default.conf"
)

type DNSHandler struct {
	tpl                 *template.Template
	dnsConfPath         string
	dBPath              string
	tplPath             string
	ticker              *time.Ticker
	quit                chan int
	nginxDefaultConfDir string
	localip             string
}

func newDNSHandler(dnsConfPath string, agentPath string, nginxDefaultConfDir string, localip string) (*DNSHandler, error) {
	instance := &DNSHandler{
		dnsConfPath:         filepath.Join(dnsConfPath),
		dBPath:              filepath.Join(agentPath),
		tplPath:             filepath.Join(dnsConfPath, "templates"),
		nginxDefaultConfDir: nginxDefaultConfDir,
		localip:             localip,
	}
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

	instance.tpl, err = instance.tpl.ParseFiles(filepath.Join(instance.tplPath, nginxDefaultTpl))
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

type nginxDefaultConf struct {
	URLRedirects []urlRedirect
}

type urlRedirect struct {
	Domain string
	URL    string
}

type forward struct {
	IPs []string
}

type ipBlackHole struct {
	ACLNames []string
}

type recursiveConcurrent struct {
	RecursiveClients *int
	FetchesPerZone   *int
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

type redirectionData struct {
	ViewName string
	RRs      []RR
}

type forwarder struct {
	IPs []string
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

	if err := handler.rewriteNginxFile(); err != nil {
		return fmt.Errorf("rewrite nginx config file error:%s", err.Error())
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
		return fmt.Errorf("path:%s get kvs error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID), err.Error())
	}
	if req.TTL == "0" || req.TTL == "" {
		rrsMap["ttl"] = names["ttl"]
	}
	if err := boltdb.GetDB().AddKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID), rrsMap); err != nil {
		return fmt.Errorf("path:%s add kvs error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID), err.Error())
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID)); err != nil {
		return fmt.Errorf("path:%s GetTableKVs error:%s", filepath.Join(viewsEndPath, req.ViewID), err.Error())
	}
	key := viewsmap["key"]
	var data string
	data = req.Name + "." + string(names["name"]) + " " + string(rrsMap["ttl"]) + " IN " + req.Type + " " + req.RData
	if err := updateRR("key"+string(viewsmap["name"]), string(key), data, string(names["name"]), true); err != nil {
		return fmt.Errorf("updateRR %s error:%s", data, err.Error())
	}
	if err := handler.rndcDumpJNLFile(); err != nil {
		return fmt.Errorf("rndcDumpJNLFile error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateRR(req pb.UpdateRRReq) error {
	var rrsMap map[string][]byte
	var err error
	formatCnameValue(&req.RData, req.Type)
	if rrsMap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID)); err != nil {
		return fmt.Errorf("GetTableKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID), err.Error())
	}
	var names map[string][]byte
	if names, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID)); err != nil {
		return fmt.Errorf("GetTableKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID), err.Error())
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID)); err != nil {
		return fmt.Errorf("GetTableKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID), err.Error())
	}
	key := viewsmap["key"]
	oldData := string(rrsMap["name"]) + "." + string(names["name"]) + " " + string(rrsMap["ttl"]) + " IN " + string(rrsMap["type"]) + " " + string(rrsMap["value"])
	if err := updateRR("key"+string(viewsmap["name"]), string(key), oldData, string(names["name"]), false); err != nil {
		return fmt.Errorf("updateRR delete rrset:%s error:%s", oldData, err.Error())
	}
	if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID),
		map[string][]byte{"name": []byte(req.Name), "type": []byte(req.Type), "value": []byte(req.RData), "ttl": []byte(req.TTL)}); err != nil {
		return fmt.Errorf("UpdateKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID), err.Error())
	}
	//add all the rrset data by rrupdate cause the delete function of the rrupdate had deleted the rrset.
	var tables []string
	if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsEndPath)); err != nil {
		return fmt.Errorf("GetTables path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsEndPath), err.Error())
	}
	for _, t := range tables {
		var data map[string][]byte
		if data, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, t)); err != nil {
			return fmt.Errorf("GetTableKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, t), err.Error())
		}
		if req.Type != string(data["type"]) || req.Name != string(data["name"]) {
			continue
		}
		var updateData string
		updateData = string(data["name"]) + "." + string(names["name"]) + " " + string(data["ttl"]) + " IN " + string(data["type"]) + " " + string(data["value"])
		if err := updateRR("key"+string(viewsmap["name"]), string(key), updateData, string(names["name"]), true); err != nil {
			return fmt.Errorf("updateRR add rrset:%s error:%s", updateData, err.Error())
		}
	}

	if err := handler.rndcDumpJNLFile(); err != nil {
		return fmt.Errorf("rndcDumpJNLFile error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) DeleteRR(req pb.DeleteRRReq) error {
	var rrsMap map[string][]byte
	var err error
	if rrsMap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID)); err != nil {
		return fmt.Errorf("GetTableKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID), err.Error())
	}
	var names map[string][]byte
	if names, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID)); err != nil {
		return fmt.Errorf("GetTableKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID), err.Error())
	}
	var viewsmap map[string][]byte
	if viewsmap, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID)); err != nil {
		return fmt.Errorf("GetTableKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID), err.Error())
	}
	key := viewsmap["key"]
	rrData := string(rrsMap["name"]) + "." + string(names["name"]) + " " + string(rrsMap["ttl"]) + " IN " + string(rrsMap["type"]) + " " + string(rrsMap["value"])
	if err := updateRR("key"+string(viewsmap["name"]), string(key), rrData, string(names["name"]), false); err != nil { //string(rrData[:])
		return fmt.Errorf("updateRR delete rrset:%s error:%s", rrData, err.Error())
	}
	if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID)); err != nil {
		return fmt.Errorf("DeleteTable path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, req.RRID), err.Error())
	}
	//add the old data by rrupdate cause the delete function of the rrupdate had deleted all the rrs.
	var tables []string
	if tables, err = boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsEndPath)); err != nil {
		return fmt.Errorf("GetTables path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsEndPath), err.Error())
	}
	for _, t := range tables {
		var data map[string][]byte
		if data, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, t)); err != nil {
			return fmt.Errorf("GetTableKVs path:%s error:%s", filepath.Join(viewsEndPath, req.ViewID, zonesPath, req.ZoneID, rRsPath, t), err.Error())
		}
		if string(rrsMap["type"]) != string(data["type"]) {
			continue
		}
		var updateData string
		updateData = string(data["name"]) + "." + string(names["name"]) + " " + string(data["ttl"]) + " IN " + string(data["type"]) + " " + string(data["value"])
		if err := updateRR("key"+string(viewsmap["name"]), string(key), updateData, string(names["name"]), true); err != nil {
			return fmt.Errorf("updateRR add rrset:%s error:%s", updateData, err.Error())
		}
	}
	if err := handler.rndcDumpJNLFile(); err != nil {
		return fmt.Errorf("rndcDumpJNLFile error:%s", err.Error())
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

func (handler *DNSHandler) CreateUrlRedirect(req pb.CreateUrlRedirectReq) error {
	//input the data into view's urlRedirects's path store.
	if err := boltdb.GetDB().AddKVs(filepath.Join(viewsPath, req.ViewID, urlRedirectsPath, req.ID),
		map[string][]byte{domain: []byte(req.Domain), url: []byte(req.URL)}); err != nil {
		return fmt.Errorf("add kvs for path %s error:%s", filepath.Join(viewsPath, req.ViewID, urlRedirectsPath, req.ID), err.Error())
	}
	//add the localzone redirection.
	if err := handler.CreateRedirection(pb.CreateRedirectionReq{
		ID:           req.ID,
		ViewID:       req.ViewID,
		Name:         req.Domain,
		TTL:          "3600",
		DataType:     "A",
		Value:        handler.localip,
		RedirectType: localZoneType,
	}); err != nil {
		return fmt.Errorf("create redirection local zone for %s error:%s", req.Domain, err.Error())
	}
	//rewrite the nginx default config.
	if err := handler.rewriteNginxFile(); err != nil {
		return fmt.Errorf("rewrite nginx default config for %s and %s error:%s", req.Domain, req.URL, err.Error())
	}
	if err := handler.nginxReload(); err != nil {
		return fmt.Errorf("nginx reload error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateUrlRedirect(req pb.UpdateUrlRedirectReq) error {
	//update view's urlRedirects's data in the db.
	if err := boltdb.GetDB().UpdateKVs(filepath.Join(viewsPath, req.ViewID, urlRedirectsPath, req.ID),
		map[string][]byte{domain: []byte(req.Domain), url: []byte(req.URL)}); err != nil {
		return fmt.Errorf("update kvs for path %s error: %s", filepath.Join(viewsPath, req.ViewID, urlRedirectsPath, req.ID), err.Error())
	}
	//update the localzone redirection.
	if err := handler.UpdateRedirection(pb.UpdateRedirectionReq{
		ID:                    req.ID,
		ViewID:                req.ViewID,
		Name:                  req.Domain,
		TTL:                   "3600",
		DataType:              "A",
		Value:                 handler.localip,
		RedirectType:          localZoneType,
		IsRedirectTypeChanged: false,
	}); err != nil {
		return fmt.Errorf("create redirection local zone for %s error: %s", req.Domain, err.Error())
	}
	//rewrite the nginx default config.
	if err := handler.rewriteNginxFile(); err != nil {
		return fmt.Errorf("rewrite nginx default config for %s and %s error:%s", req.Domain, req.URL, err.Error())
	}
	if err := handler.nginxReload(); err != nil {
		return fmt.Errorf("nginx reload error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) DeleteUrlRedirect(req pb.DeleteUrlRedirectReq) error {
	//delete view's urlRedirects's data in the db.
	if err := boltdb.GetDB().DeleteTable(filepath.Join(viewsPath, req.ViewID, urlRedirectsPath, req.ID)); err != nil {
		return fmt.Errorf("delete tables for path %s error: %s", filepath.Join(viewsPath, req.ViewID, urlRedirectsPath, req.ID), err.Error())
	}
	//Delete the localzone redirection.
	if err := handler.DeleteRedirection(pb.DeleteRedirectionReq{
		ID:           req.ID,
		ViewID:       req.ViewID,
		RedirectType: localZoneType,
	}); err != nil {
		return fmt.Errorf("delete redirection local zone for %s error: %s", req.ID, err.Error())
	}
	//rewrite the nginx default config.
	if err := handler.rewriteNginxFile(); err != nil {
		return fmt.Errorf("rewrite nginx default config for %s error:%s", req.ID, err.Error())
	}
	if err := handler.nginxReload(); err != nil {
		return fmt.Errorf("nginx reload error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateTTL(req pb.UpdateTTLReq) error {
	viewids, err := boltdb.GetDB().GetTables(viewsEndPath)
	if err != nil {
		return fmt.Errorf("path %s GetTables error:%s", viewsEndPath, err.Error())
	}
	for _, viewid := range viewids {
		zoneids, err := boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, viewid, zonesEndPath))
		if err != nil {
			return fmt.Errorf("path %s GetTables error:%s", filepath.Join(viewsEndPath, viewid, zonesEndPath), err.Error())
		}
		var view map[string][]byte
		if view, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, viewid)); err != nil {
			return fmt.Errorf("path %s GetTableKVs error:%s", filepath.Join(viewsEndPath, viewid), err.Error())
		}
		for _, zoneid := range zoneids {
			rrids, err := boltdb.GetDB().GetTables(filepath.Join(viewsEndPath, viewid, zonesEndPath, zoneid, rRsEndPath))
			if err != nil {
				return fmt.Errorf("path %s GetTables error:%s", filepath.Join(viewsEndPath, viewid, zonesEndPath, zoneid, rRsEndPath), err.Error())
			}
			var zone map[string][]byte
			if zone, err = boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, viewid, zonesPath, zoneid)); err != nil {
				return fmt.Errorf("path %s GetTableKVs error:%s", filepath.Join(viewsEndPath, viewid, zonesPath, zoneid), err.Error())
			}
			type mydata struct {
				Name   string
				Mytype string
			}
			distinct := make(map[mydata]string)
			for _, rrid := range rrids {
				rrkvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, viewid, zonesEndPath, zoneid, rRsEndPath, rrid))
				if err != nil {
					return fmt.Errorf("path %s GetTableKVs error:%s", filepath.Join(viewsEndPath, viewid, zonesEndPath, zoneid, rRsEndPath, rrid), err.Error())
				}
				distinct[mydata{Name: string(rrkvs["name"]), Mytype: string(rrkvs["type"])}] = string(rrkvs["value"])
			}
			for k, v := range distinct {
				oldData := k.Name + "." + string(zone["name"]) + " 3600 IN " + k.Mytype + " " + v
				if err := updateRR("key"+string(view["name"]), string(view["key"]), oldData, string(zone["name"]), false); err != nil {
					return fmt.Errorf("delete rrset %s error:%s", oldData, err.Error())
				}
			}
			for _, rrid := range rrids {
				rrkvs, err := boltdb.GetDB().GetTableKVs(filepath.Join(viewsEndPath, viewid, zonesEndPath, zoneid, rRsEndPath, rrid))
				if err != nil {
					return fmt.Errorf("path %s GetTableKVs error:%s", filepath.Join(viewsEndPath, viewid, zonesEndPath, zoneid, rRsEndPath, rrid), err.Error())
				}
				newData := string(rrkvs["name"]) + "." + string(zone["name"]) + " " + strconv.Itoa(int(req.TTL)) + " IN " + string(rrkvs["type"]) + " " + string(rrkvs["value"])
				if err := updateRR("key"+string(view["name"]), string(view["key"]), newData, string(zone["name"]), true); err != nil {
					return fmt.Errorf("add new rrset %s error:%s", newData, err.Error())
				}
			}
		}
	}

	return nil
}

//TODO newdb innovation begin————

func (handler *DNSHandler) UpdateDnssec(req pb.UpdateDnssecReq) error {
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

func (handler *DNSHandler) CreateACL(req pb.CreateACLReq) error {
	aCLData := ACL{ID: req.ID, Name: req.Name, IPs: req.IPs}
	buffer := new(bytes.Buffer)
	if err := handler.tpl.ExecuteTemplate(buffer, aCLTpl, aCLData); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, req.ID)+aclSuffix, buffer.Bytes(), 0644); err != nil {
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
	if err := os.Remove(filepath.Join(handler.dnsConfPath, req.ID) + aclSuffix); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) CreateView(req pb.CreateViewReq) error {
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	//update bind
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) CreateRedirection(req pb.CreateRedirectionReq) error {
	formatCnameValue(&req.Value, req.DataType)
	formatDomain(&req.Name, req.DataType, req.RedirectType)
	if req.RedirectType == nxDomain {
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
	if req.RedirectType == nxDomain {
		if req.IsRedirectTypeChanged {
			if err := handler.rewriteRPZFile(false); err != nil {
				return err
			}
			if err := handler.rewriteNamedFile(false); err != nil {
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
			if err := handler.rewriteRedirectFile(); err != nil {
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
