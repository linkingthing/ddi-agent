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
	uuid2 "github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/g53"
	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/ddi-agent/pkg/db"
	"github.com/linkingthing/ddi-agent/pkg/dns/dbhandler"
	"github.com/linkingthing/ddi-agent/pkg/dns/resource"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

const (
	mainConfName                 = "named.conf"
	dBName                       = "bind.db"
	namedTpl                     = "named.tpl"
	namedNoRPZTpl                = "named_norpz.tpl"
	zoneTpl                      = "zone.tpl"
	aCLTpl                       = "acl.tpl"
	nzfTpl                       = "nzf.tpl"
	redirectTpl                  = "redirect.tpl"
	rpzTpl                       = "rpz.tpl"
	nginxDefaultTpl              = "nginxdefault.tpl"
	rndcPort                     = "953"
	checkPeriod                  = 5
	dnsServer                    = "localhost:53"
	masterType                   = "master"
	forwardType                  = "forward"
	anyACL                       = "any"
	noneACL                      = "none"
	defaultView                  = "default"
	aclSuffix                    = ".acl"
	zoneSuffix                   = ".zone"
	nzfSuffix                    = ".nzf"
	localZoneType                = "localzone"
	nxDomain                     = "nxdomain"
	cnameType                    = "CNAME"
	ptrType                      = "PTR"
	orderFirst                   = "1"
	domain                       = "domain"
	url                          = "url"
	nginxDefaultConfFile         = "default.conf"
	RoleMain                     = "main"
	RoleBackup                   = "backup"
	defaultGlobalConfigID        = "globalConfig"
	defaultRecursiveConcurrentId = "1"

	updateZonesTTLSQL       = "update gr_agent_zone set ttl = $1"
	updateRRsTTLSQL         = "update gr_agent_rr set ttl = $1"
	updateRedirectionTtlSQL = "update gr_agent_redirection set ttl = $1"
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

func newDNSHandler(dnsConfPath string, agentPath string, nginxDefaultConfDir string, localIP string) (*DNSHandler, error) {
	instance := &DNSHandler{
		dnsConfPath:         filepath.Join(dnsConfPath),
		dBPath:              filepath.Join(agentPath),
		tplPath:             filepath.Join(dnsConfPath, "templates"),
		nginxDefaultConfDir: nginxDefaultConfDir,
		localip:             localIP,
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

	exist, err := dbhandler.Exist(resource.TableAcl, noneACL)
	if err != nil {
		return nil, err
	}
	if !exist {
		acl := &resource.AgentAcl{Name: noneACL}
		acl.SetID(noneACL)
		if err = dbhandler.Insert(acl); err != nil {
			return nil, err
		}
	}

	exist, err = dbhandler.Exist(resource.TableAcl, anyACL)
	if err != nil {
		return nil, err
	}
	if !exist {
		acl := &resource.AgentAcl{Name: anyACL}
		acl.SetID(anyACL)
		if err = dbhandler.Insert(acl); err != nil {
			return nil, err
		}
	}

	exist, err = dbhandler.Exist(resource.TableView, defaultView)
	if err != nil {
		return nil, err
	}
	if !exist {
		view := &resource.AgentView{Name: defaultView, Priority: 1}
		view.SetID(defaultView)
		view.Acls = append(view.Acls, anyACL)
		key, _ := uuid2.Gen()
		view.Key = base64.StdEncoding.EncodeToString([]byte(key))
		if err = dbhandler.Insert(view); err != nil {
			return nil, err
		}
	}

	exist, err = dbhandler.Exist(resource.TableDnsGlobalConfig, defaultGlobalConfigID)
	if err != nil {
		return nil, err
	}
	if !exist {
		dnsGlobalConfig := &resource.AgentDnsGlobalConfig{
			LogEnable: true, Ttl: 3600, DnssecEnable: false,
		}
		dnsGlobalConfig.SetID(defaultGlobalConfigID)
		if err = dbhandler.Insert(dnsGlobalConfig); err != nil {
			return nil, err
		}
	}

	if err := instance.StartDNS(&pb.DNSStartReq{}); err != nil {
		log.Errorf("start dns fail:%s", err.Error())
	}
	return instance, nil
}

func (handler *DNSHandler) StartDNS(req *pb.DNSStartReq) error {
	if err := handler.Start(); err != nil {
		return err
	}

	go handler.keepDNSAlive()
	return nil
}

func (handler *DNSHandler) Start() error {
	if _, err := os.Stat(filepath.Join(handler.dnsConfPath, "named.pid")); err == nil {
		return nil
	}
	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("init rewriteNamedFile failed:%s", err.Error())
	}
	if err := handler.rewriteACLsFile(); err != nil {
		return fmt.Errorf("init rewriteACLsFile failed:%s", err.Error())
	}

	if err := handler.rewriteZonesFile(""); err != nil {
		return fmt.Errorf("init rewriteZonesFile failed:%s", err.Error())
	}

	if err := handler.rewriteNzfsFile(); err != nil {
		return fmt.Errorf("init rewriteNzfsFile failed:%s", err.Error())
	}

	if err := handler.rewriteRedirectFile(""); err != nil {
		return fmt.Errorf("init rewriteRedirectFile failed:%s", err.Error())
	}

	if err := handler.rewriteRPZFile(true, ""); err != nil {
		return fmt.Errorf("init rewriteRPZFile failed:%s", err.Error())
	}

	if err := handler.rewriteNginxFile(); err != nil {
		return fmt.Errorf("rewrite nginx config file error:%s", err.Error())
	}

	var param = "-c" + filepath.Join(handler.dnsConfPath, mainConfName)
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "named"), param); err != nil {
		return fmt.Errorf("exec named -c  failed:%s", err.Error())
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

func (handler *DNSHandler) Close() {}

func (handler *DNSHandler) keepDNSAlive() {
	defer handler.ticker.Stop()
	for {
		select {
		case <-handler.ticker.C:
			if _, err := os.Stat(filepath.Join(handler.dnsConfPath, "named.pid")); err == nil {
				continue
			}
			handler.Start()
		case <-handler.quit:
			return
		}
	}
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

	//secret = base64.StdEncoding.EncodeToString([]byte(secret))
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

func (handler *DNSHandler) CreateACL(req *pb.CreateACLReq) error {
	acl := &resource.AgentAcl{Name: req.Name, Ips: req.Ips}
	acl.SetID(req.Id)
	if err := dbhandler.Insert(acl); err != nil {
		return fmt.Errorf("insert acl db id:%s failed: %s ", req.Id, err.Error())
	}

	buffer := new(bytes.Buffer)
	if err := handler.tpl.ExecuteTemplate(buffer, aCLTpl, acl); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, req.Id)+aclSuffix, buffer.Bytes(), 0644); err != nil {
		return err
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) UpdateACL(req *pb.UpdateACLReq) error {
	aclRes, err := dbhandler.Get(req.Id, &[]*resource.AgentAcl{})
	if err != nil {
		return err
	}

	acl := aclRes.(*resource.AgentAcl)
	acl.Ips = req.Ips
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableAcl,
			map[string]interface{}{"ips": req.Ips},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("UpdateACL failed:%s", err.Error())
	}

	buffer := new(bytes.Buffer)
	if err := handler.tpl.ExecuteTemplate(buffer, aCLTpl, acl); err != nil {
		return err
	}

	if err := ioutil.WriteFile(filepath.Join(handler.dnsConfPath, req.Id)+aclSuffix, buffer.Bytes(), 0644); err != nil {
		return err
	}

	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) DeleteACL(req *pb.DeleteACLReq) error {
	if err := dbhandler.Delete(req.Id, resource.TableAcl); err != nil {
		return fmt.Errorf("delete acl id:%s from db failed: %s", req.Id, err.Error())
	}

	return os.Remove(filepath.Join(handler.dnsConfPath, req.Id) + aclSuffix)
}

func (handler *DNSHandler) CreateView(req *pb.CreateViewReq) error {
	view := &resource.AgentView{
		Name:     req.Name,
		Priority: uint(req.Priority),
		Acls:     req.Acls,
		Dns64:    req.Dns64,
	}
	view.SetID(req.Id)
	key, _ := uuid2.Gen()
	view.Key = base64.StdEncoding.EncodeToString([]byte(key))

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := adjustPriority(view, tx, false); err != nil {
			return err
		}

		if _, err := tx.Insert(view); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("CreateView failed:%s", err.Error())
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) UpdateView(req *pb.UpdateViewReq) error {
	viewRes, err := dbhandler.Get(req.Id, &[]*resource.AgentView{})
	if err != nil {
		return err
	}
	view := viewRes.(*resource.AgentView)
	view.Priority = uint(req.Priority)
	view.Acls = req.Acls
	view.Dns64 = req.Dns64

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := adjustPriority(view, tx, false); err != nil {
			return err
		}
		if _, err := tx.Update(
			resource.TableView,
			map[string]interface{}{
				"priority": req.Priority,
				"acls":     req.Acls,
				"dns64":    req.Dns64,
			},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("UpdateView failed:%s", err.Error())
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("rewrite named.conf fail:%s", err.Error())
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("rndc reconfig fail:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) DeleteView(req *pb.DeleteViewReq) error {
	viewRes, err := dbhandler.Get(req.Id, &[]*resource.AgentView{})
	if err != nil {
		return err
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := adjustPriority(viewRes.(*resource.AgentView), tx, true); err != nil {
			return fmt.Errorf("adjust priority when delete view failed:%s", err.Error())
		}
		c, err := tx.Delete(resource.TableView, map[string]interface{}{
			restdb.IDField: viewRes.GetID(),
		})
		if err != nil {
			return fmt.Errorf("delete view %s from db failed: %s", viewRes.GetID(), err.Error())
		}
		if c == 0 {
			return nil
		}

		return nil
	}); err != nil {
		return fmt.Errorf("DeleteView failed:%s", err.Error())
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rewriteZonesFile(""); err != nil {
		return err
	}
	if err := handler.rewriteNzfsFile(); err != nil {
		return err
	}
	if err := handler.rewriteRPZFile(false, ""); err != nil {
		return err
	}
	if err := handler.rewriteRedirectFile(""); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) CreateRedirection(req *pb.CreateRedirectionReq) error {
	redirect := &resource.AgentRedirection{
		Name:         req.Name,
		Ttl:          uint(req.Ttl),
		DataType:     req.DataType,
		RedirectType: req.RedirectType,
		Rdata:        req.RData,
		View:         req.ViewId,
	}
	redirect.SetID(req.Id)

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(redirect); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("insert redirection id:%s to db failed:%s ", req.Id, err.Error())
	}

	formatCnameValue(&redirect.Rdata, redirect.DataType)
	formatDomain(&redirect.Name, redirect.DataType, redirect.RedirectType)
	if redirect.RedirectType == nxDomain {
		if err := handler.rewriteRedirectFile(req.ViewId); err != nil {
			return fmt.Errorf("CreateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
		}
	} else if redirect.RedirectType == localZoneType {
		if err := handler.rewriteRPZFile(false, req.ViewId); err != nil {
			return fmt.Errorf("CreateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
		}
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("CreateRedirection id:%s rewriteNamedFile failed:%s", req.Id, err.Error())
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("CreateRedirection id:%s rndcReconfig failed:%s", req.Id, err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateRedirection(req *pb.UpdateRedirectionReq) error {
	redirectRes, err := dbhandler.Get(req.Id, &[]*resource.AgentRedirection{})
	if err != nil {
		return fmt.Errorf("UpdateRedirection id:%s Get redirection from db failed:%s", req.Id, err.Error())
	}
	redirection := redirectRes.(*resource.AgentRedirection)
	redirection.DataType = req.DataType
	redirection.Rdata = req.RData
	redirection.Ttl = uint(req.Ttl)

	redirectTypeChanged := false
	if redirection.RedirectType != req.RedirectType {
		redirectTypeChanged = true
	}
	redirection.RedirectType = req.RedirectType

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableRedirection,
			map[string]interface{}{
				"data_type":     redirection.DataType,
				"rdata":         redirection.Rdata,
				"ttl":           redirection.Ttl,
				"redirect_type": redirection.RedirectType,
			},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("update redirect id:%s to db failed:%s ", req.Id, err.Error())
	}

	formatCnameValue(&redirection.Rdata, redirection.DataType)
	formatDomain(&redirection.Name, redirection.DataType, redirection.RedirectType)
	if redirection.RedirectType == nxDomain {
		if err := handler.rewriteRedirectFile(redirection.View); err != nil {
			return fmt.Errorf("UpdateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
		}

		if redirectTypeChanged {
			if err := handler.rewriteRPZFile(false, redirection.View); err != nil {
				return fmt.Errorf("UpdateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}
		}
	} else if redirection.RedirectType == localZoneType {
		if err := handler.rewriteRPZFile(false, redirection.View); err != nil {
			return fmt.Errorf("UpdateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
		}

		if redirectTypeChanged {
			if err := handler.rewriteRedirectFile(redirection.View); err != nil {
				return fmt.Errorf("UpdateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
			}
		}
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("UpdateRedirection id:%s rewriteNamedFile failed:%s", req.Id, err.Error())
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("UpdateRedirection id:%s rndcReconfig failed:%s", req.Id, err.Error())
	}
	return nil
}

func (handler *DNSHandler) DeleteRedirection(req *pb.DeleteRedirectionReq) error {
	redirectRes, err := dbhandler.Get(req.Id, &[]*resource.AgentRedirection{})
	if err != nil {
		return err
	}

	redirection := redirectRes.(*resource.AgentRedirection)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(
			resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return fmt.Errorf("delete redirection id:%s from db failed:%s ", req.Id, err.Error())
	}

	if redirection.RedirectType == nxDomain {
		if err := handler.rewriteRedirectFile(redirection.View); err != nil {
			return fmt.Errorf("DeleteRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
		}
	} else if redirection.RedirectType == localZoneType {
		if err := handler.rewriteRPZFile(false, redirection.View); err != nil {
			return fmt.Errorf("DeleteRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
		}
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("DeleteRedirection id:%s rewriteNamedFile failed:%s", req.Id, err.Error())
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("DeleteRedirection id:%s rndcReconfig failed:%s", req.Id, err.Error())
	}
	return nil
}

func (handler *DNSHandler) CreateZone(req *pb.CreateZoneReq) error {
	zone := &resource.AgentZone{
		Name:     req.ZoneName,
		ZoneFile: req.ZoneFileName,
		Ttl:      uint(req.Ttl),
		View:     req.ViewId,
		RrsRole:  req.RrsRole,
	}
	zone.SetID(req.ZoneId)

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(zone); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("insert zone id:%s into db failed:%s", req.ZoneId, err.Error())
	}

	if err := handler.rewriteZonesFile(zone.ID); err != nil {
		return fmt.Errorf("CreateZone rewriteZonesFile id:%s failed:%s", zone.ID, err.Error())
	}

	if err := handler.rndcAddZone(zone.Name, zone.ZoneFile, zone.View); err != nil {
		return fmt.Errorf("CreateZone rndcAddZone id:%s failed:%s", zone.ID, err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateZone(req *pb.UpdateZoneReq) error {
	zoneRes, err := dbhandler.Get(req.Id, &[]*resource.AgentZone{})
	if err != nil {
		return err
	}
	zone := zoneRes.(*resource.AgentZone)
	zone.Ttl = uint(req.Ttl)

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableZone,
			map[string]interface{}{"ttl": zone.Ttl},
			map[string]interface{}{restdb.IDField: zone.ID},
		); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return fmt.Errorf("update zone id:%s into db failed:%s", req.Id, err.Error())
	}

	if err := handler.rewriteZonesFile(req.Id); err != nil {
		return fmt.Errorf("UpdateZone rewriteZonesFile id:%s failed:%s", zone.ID, err.Error())
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("UpdateZone rndcAddZone id:%s failed:%s", zone.ID, err.Error())
	}
	return nil
}

func (handler *DNSHandler) DeleteZone(req *pb.DeleteZoneReq) error {
	zoneRes, err := restdb.GetResourceWithID(db.GetDB(), req.Id, &[]*resource.AgentZone{})
	if err != nil {
		return fmt.Errorf("Get zone id:%s from db failed:%s ", req.Id, err.Error())
	}
	zone := zoneRes.(*resource.AgentZone)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(
			resource.TableZone,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete zone id:%s from db failed:%s", req.Id, err.Error())
		}

		if err := handler.rndcDelZone(zone.Name, zone.View); err != nil {
			return fmt.Errorf("DeleteZone id:%s rndcDelZone view:%s failed:%s", req.Id, zone.View, err.Error())
		}

		if err := os.Remove(filepath.Join(handler.dnsConfPath, zone.ZoneFile)); err != nil {
			return fmt.Errorf("DeleteZone id:%s Remove %s failed:%s", req.Id, zone.ZoneFile, err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) CreateForwardZone(req *pb.CreateForwardZoneReq) error {
	forwardZone := &resource.AgentForwardZone{
		Name:        req.Name,
		ForwardType: req.ForwardType,
		ForwardIds:  req.ForwardIds,
		View:        req.ViewId,
	}
	forwardZone.SetID(req.Id)

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var forwardList []*resource.AgentForward
		sql := fmt.Sprintf(`select * from gr_agent_forward where id in ('%s')`, strings.Join(req.ForwardIds, "','"))
		if err := tx.FillEx(&forwardList, sql); err != nil {
			return fmt.Errorf("get forward ids:%s from db failed:%s", req.ForwardIds, err.Error())
		}
		if len(forwardList) == 0 {
			return fmt.Errorf("get forward ids:%s from db failed:len(forwards)==0", req.ForwardIds)
		}

		for _, value := range forwardList {
			forwardZone.Ips = append(forwardZone.Ips, value.Ips...)
		}

		if _, err := tx.Insert(forwardZone); err != nil {
			return fmt.Errorf("insert forwardzone id:%s to db failed:%s", req.Id, err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("CreateForwardZone id:%s rewriteNamedFile failed:%s", req.Id, err.Error())
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("CreateForwardZone id:%s rndcReconfig failed:%s", req.Id, err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateForwardZone(req *pb.UpdateForwardZoneReq) error {
	forwardZoneRes, err := dbhandler.Get(req.Id, &[]*resource.AgentForwardZone{})
	if err != nil {
		return err
	}

	forwardZone := forwardZoneRes.(*resource.AgentForwardZone)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		updateMap := make(map[string]interface{})
		updateMap["forward_type"] = req.ForwardType
		if isSlicesDiff(forwardZone.ForwardIds, req.ForwardIds) {
			var forwardList []*resource.AgentForward
			sql := fmt.Sprintf(`select * from gr_agent_forward where id in ('%s')`, strings.Join(req.ForwardIds, "','"))
			if err := tx.FillEx(&forwardList, sql); err != nil {
				return fmt.Errorf("get forward ids:%s from db failed:%s", req.ForwardIds, err.Error())
			}
			if len(forwardList) == 0 {
				return fmt.Errorf("get forward ids:%s from db failed:len(forwards)==0", req.ForwardIds)
			}
			for _, value := range forwardList {
				forwardZone.Ips = append(forwardZone.Ips, value.Ips...)
			}
			updateMap["ips"] = forwardZone.Ips
			updateMap["forward_ids"] = req.ForwardIds
		}

		if _, err := tx.Update(resource.TableForwardZone, updateMap,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("update forwardZone id:%s to db failed:%s", req.Id, err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("UpdateForwardZone id:%s rewriteNamedFile failed:%s", req.Id, err.Error())
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("UpdateForwardZone id:%s rndcReconfig failed:%s", req.Id, err.Error())
	}
	return nil
}

func (handler *DNSHandler) DeleteForwardZone(req *pb.DeleteForwardZoneReq) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(
			resource.TableForwardZone,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete forwardzone id:%s failed:%s", req.Id, err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("DeleteForwardZone id:%s rewriteNamedFile failed:%s", req.Id, err.Error())
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("DeleteForwardZone id:%s rndcReconfig failed:%s", req.Id, err.Error())
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

func (handler *DNSHandler) CreateRR(req *pb.CreateRRReq) error {
	rr := &resource.AgentRr{
		Name:        req.Name,
		DataType:    req.DataType,
		Ttl:         uint(req.Ttl),
		Rdata:       req.RData,
		RdataBackup: req.BackupRData,
		ActiveRdata: req.RData,
		View:        req.ViewId,
		Zone:        req.ZoneId,
	}
	rr.SetID(req.Id)

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(rr); err != nil {
			return err
		}
		return nil
	}); err != nil {
		fmt.Errorf("CreateRR id:%s insert into db failed:%s", req.Id, err.Error())
	}

	viewRes, err := dbhandler.Get(req.ViewId, &[]*resource.AgentView{})
	if err != nil {
		return err
	}
	zoneRes, err := dbhandler.Get(req.ZoneId, &[]*resource.AgentZone{})
	if err != nil {
		return err
	}
	view := viewRes.(*resource.AgentView)
	zone := zoneRes.(*resource.AgentZone)

	formatCnameValue(&rr.Rdata, rr.DataType)
	var buildData strings.Builder
	buildData.WriteString(rr.Name)
	buildData.WriteString(".")
	buildData.WriteString(zone.Name)
	buildData.WriteString(" ")
	buildData.WriteString(strconv.FormatUint(uint64(rr.Ttl), 10))
	buildData.WriteString(" IN ")
	buildData.WriteString(rr.DataType)
	buildData.WriteString(" ")
	if zone.RrsRole == RoleBackup && rr.RdataBackup != "" {
		buildData.WriteString(rr.RdataBackup)
	} else {
		buildData.WriteString(rr.Rdata)
	}

	if err := updateRR("key"+view.Name, view.Key, buildData.String(), zone.Name, true); err != nil {
		return fmt.Errorf("updateRR %s error:%s", buildData.String(), err.Error())
	}

	if err := handler.rndcDumpJNLFile(); err != nil {
		return fmt.Errorf("rndcDumpJNLFile error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateRRsByZone(req *pb.UpdateRRsByZoneReq) error {
	zoneRes, err := dbhandler.Get(req.ZoneId, &[]*resource.AgentZone{})
	if err != nil {
		return fmt.Errorf("get zone from db failed:%s ", err.Error())
	}
	zone := zoneRes.(*resource.AgentZone)
	if zone.RrsRole == req.Role {
		return nil
	}

	viewRes, err := dbhandler.Get(zone.View, &[]*resource.AgentView{})
	if err != nil {
		return fmt.Errorf("get view from db failed:%s ", err.Error())
	}

	var rrList []*resource.AgentRr
	if err := dbhandler.List(&rrList); err != nil {
		return fmt.Errorf("list rr from db failed:%s ", err.Error())
	}
	view := viewRes.(*resource.AgentView)

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableZone,
			map[string]interface{}{"rrs_role": req.Role},
			map[string]interface{}{restdb.IDField: req.ZoneId}); err != nil {
			return err
		}

		var buildData strings.Builder
		for _, rr := range rrList {
			formatCnameValue(&rr.Rdata, rr.DataType)
			buildData.Reset()
			buildData.WriteString(rr.Name)
			buildData.WriteString(".")
			buildData.WriteString(zone.Name)
			buildData.WriteString(" ")
			buildData.WriteString(strconv.FormatUint(uint64(rr.Ttl), 10))
			buildData.WriteString(" IN ")
			buildData.WriteString(rr.DataType)
			buildData.WriteString(" ")
			if req.Role == RoleBackup && rr.RdataBackup != "" {
				buildData.WriteString(rr.RdataBackup)
			} else {
				continue
			}

			if err := updateRR("key"+view.Name, view.Key, buildData.String(), zone.Name, false); err != nil {
				return fmt.Errorf("updateRR delete rrset:%s error:%s", buildData.String(), err.Error())
			}

			if err := updateRR("key"+view.Name, view.Key, buildData.String(), zone.Name, true); err != nil {
				return fmt.Errorf("updateRR add rrset:%s error:%s", buildData.String(), err.Error())
			}
		}

		if err := handler.rndcDumpJNLFile(); err != nil {
			return fmt.Errorf("rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateAllRR() error {
	var rrList []*resource.AgentRr
	if err := dbhandler.List(&rrList); err != nil {
		return fmt.Errorf("UpdateAllRR List rr falied:%s", err.Error())
	}

	for _, rr := range rrList {
		viewRes, err := dbhandler.Get(rr.View, &[]*resource.AgentView{})
		if err != nil {
			return fmt.Errorf("UpdateAllRR Get views falied:%s", err.Error())
		}
		zoneRes, err := dbhandler.Get(rr.Zone, &[]*resource.AgentZone{})
		if err != nil {
			return fmt.Errorf("UpdateAllRR Get zones falied:%s", err.Error())
		}
		view := viewRes.(*resource.AgentView)
		zone := zoneRes.(*resource.AgentZone)

		formatCnameValue(&rr.Rdata, rr.DataType)
		var buildData strings.Builder
		buildData.WriteString(rr.Name)
		buildData.WriteString(".")
		buildData.WriteString(rr.Zone)
		buildData.WriteString(" ")
		buildData.WriteString(strconv.FormatUint(uint64(rr.Ttl), 10))
		buildData.WriteString(" IN ")
		buildData.WriteString(rr.DataType)
		buildData.WriteString(" ")
		if zone.RrsRole == RoleBackup && rr.RdataBackup != "" {
			buildData.WriteString(rr.RdataBackup)
		} else {
			buildData.WriteString(rr.Rdata)
		}
		if err := updateRR("key"+view.Name, view.Key, buildData.String(), zone.Name, false); err != nil {
			return fmt.Errorf("UpdateAllRR delete rrset:%s error:%s", buildData.String(), err.Error())
		}

		if err := updateRR("key"+view.Name, view.Key, buildData.String(), zone.Name, true); err != nil {
			return fmt.Errorf("UpdateAllRR add rrset:%s error:%s", buildData.String(), err.Error())
		}
	}

	if err := handler.rndcDumpJNLFile(); err != nil {
		return fmt.Errorf("UpdateAllRR rndcDumpJNLFile error:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) UpdateRR(req *pb.UpdateRRReq) error {
	rr, err := dbhandler.Get(req.Id, &[]*resource.AgentRr{})
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableRR,
			map[string]interface{}{
				"data_type":    req.DataType,
				"ttl":          req.Ttl,
				"rdata":        req.RData,
				"rdata_backup": req.BackupRData,
			},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return err
		}

		viewRes, err := dbhandler.Get(rr.(*resource.AgentRr).View, &[]*resource.AgentView{})
		if err != nil {
			return err
		}
		zoneRes, err := dbhandler.Get(rr.(*resource.AgentRr).Zone, &[]*resource.AgentZone{})
		if err != nil {
			return err
		}
		view := viewRes.(*resource.AgentView)
		zone := zoneRes.(*resource.AgentZone)

		formatCnameValue(&req.RData, req.DataType)
		var buildData strings.Builder
		buildData.WriteString(rr.(*resource.AgentRr).Name)
		buildData.WriteString(".")
		buildData.WriteString(zone.Name)
		buildData.WriteString(" ")
		buildData.WriteString(strconv.FormatUint(uint64(req.Ttl), 10))
		buildData.WriteString(" IN ")
		buildData.WriteString(req.DataType)
		buildData.WriteString(" ")
		if zone.RrsRole == RoleBackup && req.BackupRData != "" {
			buildData.WriteString(req.BackupRData)
		} else {
			buildData.WriteString(req.RData)
		}
		if err := updateRR("key"+view.Name, view.Key, buildData.String(), zone.Name, false); err != nil {
			return fmt.Errorf("updateRR delete rrset:%s error:%s", buildData.String(), err.Error())
		}

		if err := updateRR("key"+view.Name, view.Key, buildData.String(), zone.Name, true); err != nil {
			return fmt.Errorf("updateRR add rrset:%s error:%s", buildData.String(), err.Error())
		}

		if err := handler.rndcDumpJNLFile(); err != nil {
			return fmt.Errorf("rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteRR(req *pb.DeleteRRReq) error {
	rrRes, err := dbhandler.Get(req.Id, &[]*resource.AgentRr{})
	if err != nil {
		return err
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableRR, map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete  rr id:%s from db failed:%s", req.Id, err.Error())
		}

		rr := rrRes.(*resource.AgentRr)
		viewRes, err := dbhandler.Get(rr.View, &[]*resource.AgentView{})
		if err != nil {
			return fmt.Errorf("get view id:%s from db failed:%s", rr.View, err.Error())
		}
		zoneRes, err := dbhandler.Get(rr.Zone, &[]*resource.AgentZone{})
		if err != nil {
			return fmt.Errorf("get zone id:%s from db failed:%s", rr.Zone, err.Error())
		}
		view := viewRes.(*resource.AgentView)
		zone := zoneRes.(*resource.AgentZone)

		var buildData strings.Builder
		buildData.WriteString(rr.Name)
		buildData.WriteString(".")
		buildData.WriteString(zone.Name)
		buildData.WriteString(" ")
		buildData.WriteString(strconv.FormatUint(uint64(rr.Ttl), 10))
		buildData.WriteString(" IN ")
		buildData.WriteString(rr.DataType)
		buildData.WriteString(" ")
		buildData.WriteString(rr.Rdata)
		if err := updateRR("key"+view.Name, view.Key, buildData.String(), zone.Name, false); err != nil {
			return fmt.Errorf("updateRR delete rrset:%s error:%s", buildData.String(), err.Error())
		}

		if err := handler.rndcDumpJNLFile(); err != nil {
			return fmt.Errorf("rndcDumpJNLFile error:%s", err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) CreateForward(req *pb.CreateForwardReq) error {
	forward := &resource.AgentForward{
		Name: req.Name,
		Ips:  req.Ips,
	}
	forward.SetID(req.Id)

	tx, err := db.GetDB().Begin()
	if err != nil {
		return fmt.Errorf("insert forward id:%s to db failed:%s", req.Id, err.Error())
	}
	_, err = tx.Insert(forward)
	if err != nil {
		return fmt.Errorf("insert forward id:%s to db failed:%s", req.Id, err.Error())
	}
	tx.Commit()

	return nil
}

func (handler *DNSHandler) UpdateForward(req *pb.UpdateForwardReq) error {
	forwardRes, err := dbhandler.Get(req.Id, &[]*resource.AgentForward{})
	if err != nil {
		return err
	}
	forward := forwardRes.(*resource.AgentForward)
	if !isSlicesDiff(forward.Ips, req.Ips) {
		return nil
	}

	if err = restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableForward,
			map[string]interface{}{"ips": req.Ips},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			fmt.Errorf("update forward id:%s to db failed:%s", req.Id, err.Error())
		}

		var forwardZoneList []*resource.AgentForwardZone
		if err := tx.FillEx(&forwardZoneList, "select * from gr_agent_forward_zone where $1 = ANY(forward_ids)", req.Id); err != nil {
			return fmt.Errorf("get forwardZoneList from db failed:%s", err.Error())
		}
		for _, forwardZone := range forwardZoneList {
			for k, ip := range append(forwardZone.Ips, req.Ips...) {
				for _, fip := range forward.Ips {
					if fip == ip {
						forwardZone.Ips = append(forwardZone.Ips[:k], forwardZone.Ips[k+1:]...)
						continue
					}
				}
			}

			if _, err := tx.Exec("update gr_agent_forward_zone set ips = $1 where id = $2", forwardZone.Ips, forwardZone.ID); err != nil {
				return fmt.Errorf("update gr_agent_forward_zone id:%s to db failed:%s", forwardZone.ID, err.Error())
			}
		}

		return nil
	}); err != nil {
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

func (handler *DNSHandler) DeleteForward(req *pb.DeleteForwardReq) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableForward, map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete forward id:%s from db failed:%s", req.Id, err.Error())
		}

		return nil
	}); err != nil {
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

func (handler *DNSHandler) CreateIPBlackHole(req *pb.CreateIPBlackHoleReq) error {
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	ipBlackHole := &resource.AgentIpBlackHole{Acl: req.Acl}
	ipBlackHole.SetID(req.Id)
	if _, err := tx.Insert(ipBlackHole); err != nil {
		return fmt.Errorf("insert ipBlackHole to db failed:%s", err.Error())
	}
	tx.Commit()

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateIPBlackHole(req *pb.UpdateIPBlackHoleReq) error {
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	if _, err := tx.Update(resource.TableIpBlackHole,
		map[string]interface{}{"acl": req.Acl},
		map[string]interface{}{restdb.IDField: req.Id}); err != nil {
		return fmt.Errorf("insert ipBlackHole to db failed:%s", err.Error())
	}
	tx.Commit()

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) DeleteIPBlackHole(req *pb.DeleteIPBlackHoleReq) error {
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	if _, err := tx.Delete(resource.TableIpBlackHole,
		map[string]interface{}{restdb.IDField: req.Id}); err != nil {
		return fmt.Errorf("delete ipBlackHole to db failed:%s", err.Error())
	}
	tx.Commit()

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateRecursiveConcurrent(req *pb.UpdateRecurConcuReq) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		c, err := tx.Count(resource.TableRecursiveConcurrent, nil)
		if err != nil {
			return fmt.Errorf("get RecursiveConcurrent from db failed:%s", err.Error())
		}
		if c == 0 {
			recursiveConcurrent := &resource.AgentRecursiveConcurrent{
				RecursiveClients: req.RecursiveClients,
				FetchesPerZone:   req.FetchesPerZone,
			}
			recursiveConcurrent.SetID(req.Id)
			if _, err := tx.Insert(recursiveConcurrent); err != nil {
				return fmt.Errorf("insert RecursiveConcurrent from db failed:%s", err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TableRecursiveConcurrent,
				map[string]interface{}{
					"recursive_clients": req.RecursiveClients,
					"fetches_per_zone":  req.FetchesPerZone,
				},
				map[string]interface{}{restdb.IDField: req.Id}); err != nil {
				return fmt.Errorf("update RecursiveConcurrent from db failed:%s", err.Error())
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("UpdateRecursiveConcurrent failed:%s", err.Error())
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	return nil
}

func (handler *DNSHandler) CreateSortList(req *pb.CreateSortListReq) error {
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) UpdateSortList(req *pb.UpdateSortListReq) error {
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) DeleteSortList(req *pb.DeleteSortListReq) error {
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}
	return nil
}

func (handler *DNSHandler) CreateUrlRedirect(req *pb.CreateUrlRedirectReq) error {
	urlRedirect := &resource.AgentUrlRedirect{
		Domain: req.Domain,
		Url:    req.Url,
		View:   req.ViewId,
	}
	urlRedirect.SetID(req.Id)

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		exist, _ := tx.Exists(resource.TableRedirection,
			map[string]interface{}{"name": urlRedirect.Domain})
		if exist {
			return fmt.Errorf("insert urlredirect id:%s to db failed:domain:%s in redirection has exist", req.Id, req.Domain)
		}

		redirection := &resource.AgentRedirection{
			Name:         urlRedirect.Domain,
			Ttl:          3600,
			DataType:     "A",
			Rdata:        handler.localip,
			RedirectType: localZoneType,
			View:         urlRedirect.View,
		}
		redirection.SetID(req.Id)

		if _, err := tx.Insert(urlRedirect); err != nil {
			return fmt.Errorf("insert urlredirect id:%s to db failed:%s", req.Id, err.Error())
		}

		if _, err := tx.Insert(redirection); err != nil {
			return fmt.Errorf("insert redirect id:%s to db failed:%s", req.Id, err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	if err := handler.rewriteRPZFile(false, urlRedirect.View); err != nil {
		return fmt.Errorf("create redirection local zone for %s error:%s", urlRedirect.Domain, err.Error())
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	if err := handler.rewriteNginxFile(); err != nil {
		return fmt.Errorf("rewrite nginx default config for %s and %s error:%s", urlRedirect.Domain, urlRedirect.Url, err.Error())
	}
	if err := handler.nginxReload(); err != nil {
		return fmt.Errorf("nginx reload error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateUrlRedirect(req *pb.UpdateUrlRedirectReq) error {
	urlRedirectRes, err := dbhandler.Get(req.Id, &[]*resource.AgentUrlRedirect{})
	if err != nil {
		return err
	}

	urlRedirect := urlRedirectRes.(*resource.AgentUrlRedirect)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableUrlRedirect,
			map[string]interface{}{"domain": req.Domain, "url": req.Url},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("update urlredirect id:%s to db failed:%s", req.Id, err.Error())
		}

		if urlRedirect.Domain != req.Domain {
			if _, err := tx.Update(resource.TableRedirection,
				map[string]interface{}{"name": req.Domain},
				map[string]interface{}{restdb.IDField: req.Id}); err != nil {
				return fmt.Errorf("update redirection id:%s to db failed:%s", req.Id, err.Error())
			}
		}
		return nil
	}); err != nil {
		return err
	}

	if urlRedirect.Domain != req.Domain {
		if err := handler.rewriteRPZFile(false, urlRedirect.View); err != nil {
			return fmt.Errorf("create redirection local zone for %s error:%s", urlRedirect.Domain, err.Error())
		}
		if err := handler.rewriteNamedFile(false); err != nil {
			return err
		}
		if err := handler.rndcReconfig(); err != nil {
			return err
		}
	}

	if err := handler.rewriteNginxFile(); err != nil {
		return fmt.Errorf("rewrite nginx default config for %s and %s error:%s", urlRedirect.Domain, urlRedirect.Url, err.Error())
	}
	if err := handler.nginxReload(); err != nil {
		return fmt.Errorf("nginx reload error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) DeleteUrlRedirect(req *pb.DeleteUrlRedirectReq) error {
	urlRedirectRes, err := dbhandler.Get(req.Id, &[]*resource.AgentUrlRedirect{})
	if err != nil {
		return err
	}

	urlRedirect := urlRedirectRes.(*resource.AgentUrlRedirect)
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableUrlRedirect,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete urlRedirect id:%s from db failed:%s", req.Id, err.Error())
		}

		if _, err := tx.Delete(resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete Redirecttion id:%s from db failed:%s", req.Id, err.Error())
		}

		return nil
	}); err != nil {
		return err
	}

	if err := handler.rewriteRPZFile(false, urlRedirect.View); err != nil {
		return fmt.Errorf("create redirection local zone for %s error:%s", urlRedirect.Domain, err.Error())
	}
	if err := handler.rewriteNamedFile(true); err != nil {
		return err
	}
	if err := handler.rewriteNamedFile(false); err != nil {
		return err
	}
	if err := handler.rndcReconfig(); err != nil {
		return err
	}

	if err := handler.rewriteNginxFile(); err != nil {
		return fmt.Errorf("rewrite nginx default config for %s error:%s", req.Id, err.Error())
	}
	if err := handler.nginxReload(); err != nil {
		return fmt.Errorf("nginx reload error:%s", err.Error())
	}
	return nil
}

func (handler *DNSHandler) UpdateGlobalConfig(req *pb.UpdateGlobalConfigReq) error {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableDnsGlobalConfig,
			map[string]interface{}{
				"log_enable":    req.LogEnable,
				"ttl":           req.Ttl,
				"dnssec_enable": req.DnssecEnable,
			},
			map[string]interface{}{restdb.IDField: defaultGlobalConfigID}); err != nil {
			return fmt.Errorf("update dnsGlobalConfig to db failed:%s", err.Error())

		}

		if req.TtlChanged {
			if _, err := tx.Exec(updateZonesTTLSQL, req.Ttl); err != nil {
				return fmt.Errorf("update updateZonesTTLSQL to db failed:%s", err.Error())
			}
			if _, err := tx.Exec(updateRedirectionTtlSQL, req.Ttl); err != nil {
				return fmt.Errorf("update updateRedirectionTtlSQL to db failed:%s", err.Error())
			}
			if _, err := tx.Exec(updateRRsTTLSQL, req.Ttl); err != nil {
				return fmt.Errorf("update updateRRsTTLSQL to db failed:%s", err.Error())
			}

		}
		return nil
	}); err != nil {
		return fmt.Errorf("UpdateGlobalConfig failed:%s", err)
	}

	if req.TtlChanged {
		if err := handler.UpdateAllRR(); err != nil {
			return err
		}
		if err := handler.rewriteRedirectFile(""); err != nil {
			return fmt.Errorf("UpdateGlobalConfig rewriteRedirectFile failed:%s", err)
		}
		if err := handler.rewriteRPZFile(false, ""); err != nil {
			return fmt.Errorf("UpdateGlobalConfig rewriteRPZFile failed:%s", err)
		}
		if err := handler.rewriteZonesFile(""); err != nil {
			return fmt.Errorf("UpdateGlobalConfig rewriteZonesFile failed:%s", err)
		}
	}

	if err := handler.rewriteNamedFile(false); err != nil {
		return fmt.Errorf("UpdateGlobalConfig rewriteNamedFile failed:%s", err)
	}
	if err := handler.rndcReconfig(); err != nil {
		return fmt.Errorf("UpdateGlobalConfig rewriteNamedFile failed:%s", err)
	}

	return nil
}

func adjustPriority(view *resource.AgentView, tx restdb.Transaction, isDelete bool) error {
	if view.GetID() == defaultView {
		return nil
	}

	var views []*resource.AgentView
	if err := db.GetResources(map[string]interface{}{"orderby": "priority"}, &views); err != nil {
		return err
	}

	if len(views) != 1 && int(view.Priority) >= len(views) {
		return fmt.Errorf("view.Priority update error:%d should < default.Priority(%d)", view.Priority, len(views))
	} else if int(view.Priority) < 1 {
		view.Priority = 1
	}

	for i, v := range views {
		if v.GetID() == view.GetID() {
			views = append(views[:i], views[i+1:]...)
			break
		}
	}
	if !isDelete {
		views = append(views[:view.Priority-1], append([]*resource.AgentView{view}, views[view.Priority-1:]...)...)
	}
	for i, view := range views {
		if _, err := tx.Update(resource.TableView, map[string]interface{}{
			"priority": i + 1,
		}, map[string]interface{}{restdb.IDField: view.GetID()}); err != nil {
			return err
		}
	}
	return nil
}

func isSlicesDiff(slice1, slice2 []string) bool {
	if len(slice1) != len(slice2) {
		return true
	}
	temp := make(map[string]struct{})
	for _, v := range slice1 {
		temp[v] = struct{}{}
	}
	for _, v := range slice2 {
		if _, ok := temp[v]; ok {
			delete(temp, v)
		}
	}
	return len(temp) > 0
}
