package grpcservice

import (
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
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
	namedTpl                     = "named.tpl"
	namedViewTpl                 = "named_view.tpl"
	namedViewConfName            = "named_view.conf"
	namedAclTpl                  = "named_acl.tpl"
	namedAclConfName             = "named_acl.conf"
	namedOptionsTpl              = "named_options.tpl"
	namedOptionsConfName         = "named_options.conf"
	nginxDefaultTpl              = "nginxdefault.tpl"
	redirectTpl                  = "redirect.tpl"
	rpzTpl                       = "rpz.tpl"
	zoneTpl                      = "zone.tpl"
	zoneSuffix                   = ".zone"
	nzfTpl                       = "nzf.tpl"
	nzfSuffix                    = ".nzf"
	dnsServer                    = "localhost:53"
	rndcPort                     = "953"
	checkPeriod                  = 5
	anyACL                       = "any"
	noneACL                      = "none"
	defaultView                  = "default"
	localZoneType                = "localzone"
	nxDomain                     = "nxdomain"
	ptrType                      = "PTR"
	RoleBackup                   = "backup"
	nginxDefaultConfFile         = "default.conf"
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
	dnsServer           string
	rndcConfPath        string
	rndcPath            string
	nginxConfPath       string
	namedViewPath       string
	namedOptionPath     string
	namedAclPath        string
}

func newDNSHandler(dnsConfPath string, agentPath string, nginxDefaultConfDir string, localIP string) (*DNSHandler, error) {
	instance := &DNSHandler{
		dnsConfPath:         filepath.Join(dnsConfPath),
		dBPath:              filepath.Join(agentPath),
		tplPath:             filepath.Join(dnsConfPath, "templates"),
		nginxDefaultConfDir: nginxDefaultConfDir,
		localip:             localIP,
	}

	instance.tpl = template.Must(template.ParseGlob(filepath.Join(instance.tplPath, "*.tpl")))
	instance.ticker = time.NewTicker(checkPeriod * time.Second)
	instance.quit = make(chan int)

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

	if err := initDefaultDbData(); err != nil {
		return fmt.Errorf("initDefaultDbData failed:%s", err.Error())
	}

	if err := handler.initFiles(); err != nil {
		return err
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

func initDefaultDbData() error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exist, err := tx.Exists(resource.TableAcl, map[string]interface{}{restdb.IDField: noneACL}); err != nil {
			return fmt.Errorf("check agent_acl noneACL exist from db failed:%s", err.Error())
		} else if !exist {
			acl := &resource.AgentAcl{Name: noneACL}
			acl.SetID(noneACL)
			if _, err = tx.Insert(acl); err != nil {
				return fmt.Errorf("Insert agent_acl noneACL into db failed:%s ", err.Error())
			}
		}

		if exist, err := dbhandler.Exist(resource.TableAcl, anyACL); err != nil {
			return fmt.Errorf("check agent_acl anyACL exist from db failed:%s", err.Error())
		} else if !exist {
			acl := &resource.AgentAcl{Name: anyACL}
			acl.SetID(anyACL)
			if _, err = tx.Insert(acl); err != nil {
				return fmt.Errorf("Insert agent_acl anyACL into db failed:%s ", err.Error())
			}
		}

		if exist, err := dbhandler.Exist(resource.TableView, defaultView); err != nil {
			return fmt.Errorf("check agent_view defaultView exist from db failed:%s", err.Error())
		} else if !exist {
			view := &resource.AgentView{Name: defaultView, Priority: 1}
			view.SetID(defaultView)
			view.Acls = append(view.Acls, anyACL)
			key, _ := uuid2.Gen()
			view.Key = base64.StdEncoding.EncodeToString([]byte(key))
			if _, err = tx.Insert(view); err != nil {
				return fmt.Errorf("Insert agent_view defaultView into db failed:%s ", err.Error())
			}
		}

		if exist, err := dbhandler.Exist(resource.TableDnsGlobalConfig, defaultGlobalConfigID); err != nil {
			return fmt.Errorf("check agent_dns_global_config defaultGlobalConfigID exist from db failed:%s", err.Error())
		} else if !exist {
			dnsGlobalConfig := &resource.AgentDnsGlobalConfig{
				LogEnable: true, Ttl: 3600, DnssecEnable: false,
			}
			dnsGlobalConfig.SetID(defaultGlobalConfigID)
			if _, err = tx.Insert(dnsGlobalConfig); err != nil {
				return fmt.Errorf("Insert agent_view defaultGlobalConfigID into db failed:%s ", err.Error())
			}
		}

		return nil
	})
}

func updateRR(key string, secret string, rrset *g53.RRset, zone string, isAdd bool) error {
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

	msg := g53.MakeUpdate(zone_)
	if isAdd {
		msg.UpdateAddRRset(rrset)
	} else {
		msg.UpdateRemoveRRset(rrset)
	}
	msg.Header.Id = 1200

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

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(acl); err != nil {
			return fmt.Errorf("CreateACL insert acl db id:%s failed: %s ", req.Id, err.Error())
		}

		if err := handler.rewriteNamedAclFile(false, tx); err != nil {
			return fmt.Errorf("CreateACL id:%s rewriteNamedFile failed :%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateACL(req *pb.UpdateACLReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableAcl,
			map[string]interface{}{"ips": req.Ips},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("UpdateACL failed:%s", err.Error())
		}

		if err := handler.rewriteNamedAclFile(false, tx); err != nil {
			return fmt.Errorf("UpdateACL id:%s rewriteNamedFile failed :%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteACL(req *pb.DeleteACLReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableAcl, map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("DeleteACL acl id:%s from db failed: %s", req.Id, err.Error())
		}

		if err := handler.rewriteNamedAclFile(false, tx); err != nil {
			return fmt.Errorf("DeleteACL id:%s rewriteNamedFile failed :%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateView(req *pb.CreateViewReq) error {
	view := &resource.AgentView{
		Name:     req.Name,
		Priority: uint(req.Priority),
		Acls:     req.Acls,
		Dns64:    req.Dns64,
		Key:      req.Key,
	}
	view.SetID(req.Id)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(view); err != nil {
			return fmt.Errorf("CreateView id:%s Insert to db failed:%s", req.Id, err.Error())
		}

		for _, v := range req.ViewPriority {
			if v.Id != "" && v.Id != view.GetID() {
				if _, err := tx.Update(resource.TableView,
					map[string]interface{}{"priority": v.Priority},
					map[string]interface{}{restdb.IDField: v.Id}); err != nil {
					return fmt.Errorf("CreateView id:%s update priority to db failed:%s", v.Id, err.Error())
				}
			}
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("CreateView id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}

		return nil
	})
}

func (handler *DNSHandler) UpdateView(req *pb.UpdateViewReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableView,
			map[string]interface{}{
				"priority": req.Priority,
				"acls":     req.Acls,
				"dns64":    req.Dns64,
			},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("UpdateView id:%s update to db failed:%s", req.Id, err.Error())
		}

		for _, view := range req.ViewPriority {
			if _, err := tx.Update(resource.TableView,
				map[string]interface{}{"priority": view.Priority},
				map[string]interface{}{restdb.IDField: view.Id}); err != nil {
				return fmt.Errorf("CreateView id:%s update priority to db failed:%s", view.Id, err.Error())
			}
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("UpdateView id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteView(req *pb.DeleteViewReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableView, map[string]interface{}{
			restdb.IDField: req.Id,
		}); err != nil {
			return fmt.Errorf("DeleteView delete view %s from db failed: %s", req.Id, err.Error())
		}
		for _, view := range req.ViewPriority {
			if _, err := tx.Update(resource.TableView,
				map[string]interface{}{"priority": view.Priority},
				map[string]interface{}{restdb.IDField: view.Id}); err != nil {
				return fmt.Errorf("CreateView id:%s update priority to db failed:%s", view.Id, err.Error())
			}
		}

		if err := removeFiles(filepath.Join(handler.dnsConfPath), req.Id+"#", zoneSuffix); err != nil {
			return fmt.Errorf("DeleteView zonefile in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}
		if err := removeOneFile(filepath.Join(handler.dnsConfPath, req.Id) + nzfSuffix); err != nil {
			return fmt.Errorf("DeleteView delete nzf failed:%s", err.Error())
		}
		if err := removeOneFile(filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+req.Id)); err != nil {
			return fmt.Errorf("DeleteView delete rpz failed:%s", err.Error())
		}
		if err := removeOneFile(filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+req.Id)); err != nil {
			return fmt.Errorf("DeleteView delete redirect failed:%s", err.Error())
		}
		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("DeleteView rewriteNamedViewFile  failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateRedirection(req *pb.CreateRedirectionReq) error {
	redirect := &resource.AgentRedirection{
		Name:         req.Name,
		Ttl:          uint(req.Ttl),
		DataType:     req.DataType,
		RedirectType: req.RedirectType,
		Rdata:        req.RData,
		AgentView:    req.ViewId,
	}
	redirect.SetID(req.Id)
	if err := formatRdata(&redirect.Rdata, redirect.DataType); err != nil {
		return fmt.Errorf("formatRdata id:%s failed:%s", req.Id, err.Error())
	}
	if redirect.DataType != ptrType && redirect.RedirectType != localZoneType {
		name, err := g53.NameFromString(redirect.Name)
		if err != nil {
			return fmt.Errorf("formatDomain failed:%s", err.Error())
		}
		redirect.Name = name.String(false)
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(redirect); err != nil {
			return fmt.Errorf("CreateRedirection insert to db id:%s failed:%s", req.Id, err.Error())
		}

		if redirect.RedirectType == localZoneType {
			if err := handler.rewriteOneRPZFile(redirect.AgentView, tx); err != nil {
				return fmt.Errorf("CreateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}
			return nil
		}

		if err := handler.rewriteOneRedirectFile(redirect.AgentView, tx); err != nil {
			return fmt.Errorf("CreateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateRedirection(req *pb.UpdateRedirectionReq) error {
	if err := formatRdata(&req.RData, req.DataType); err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableRedirection,
			map[string]interface{}{
				"data_type":     req.DataType,
				"rdata":         req.RData,
				"ttl":           req.Ttl,
				"redirect_type": req.RedirectType,
			},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return err
		}

		if req.RedirectType == localZoneType {
			if err := handler.rewriteOneRPZFile(req.View, tx); err != nil {
				return fmt.Errorf("UpdateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}
			if req.RedirectTypeChanged {
				if err := handler.rewriteOneRedirectFile(req.View, tx); err != nil {
					return fmt.Errorf("UpdateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
				}
			}
			return nil
		}

		if err := handler.rewriteOneRedirectFile(req.View, tx); err != nil {
			return fmt.Errorf("UpdateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
		}
		if req.RedirectTypeChanged {
			if err := handler.rewriteOneRPZFile(req.View, tx); err != nil {
				return fmt.Errorf("UpdateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteRedirection(req *pb.DeleteRedirectionReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(
			resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("DeleteRedirection insert id:%s to db failed:%s", req.Id, err.Error())
		}

		if req.RedirectType == localZoneType {
			if err := handler.rewriteOneRPZFile(req.View, tx); err != nil {
				return fmt.Errorf("DeleteRedirection UpdateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}
			return nil
		}

		if err := handler.rewriteOneRedirectFile(req.View, tx); err != nil {
			return fmt.Errorf("DeleteRedirection UpdateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateZone(req *pb.CreateZoneReq) error {
	zone := &resource.AgentZone{
		Name:      req.ZoneName,
		ZoneFile:  req.ZoneFileName,
		Ttl:       uint(req.Ttl),
		AgentView: req.ViewId,
		RrsRole:   req.RrsRole,
	}
	zone.SetID(req.ZoneId)

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(zone); err != nil {
			return fmt.Errorf("CreateZone id:%s failed:%s", req.ZoneId, err.Error())
		}

		if err := handler.createZoneFile(zone); err != nil {
			return fmt.Errorf("CreateZone rewriteZoneFile id:%s failed:%s", zone.ID, err.Error())
		}

		if err := handler.rndcAddZone(zone.Name, zone.ZoneFile, zone.AgentView); err != nil {
			return fmt.Errorf("CreateZone rndcAddZone id:%s failed:%s", zone.ID, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateZone(req *pb.UpdateZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableZone,
			map[string]interface{}{"ttl": req.Ttl},
			map[string]interface{}{restdb.IDField: req.Id},
		); err != nil {
			return fmt.Errorf("UpdateZone id:%s failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteOneZoneFile(req.Id, req.ZoneFileName, tx); err != nil {
			return fmt.Errorf("UpdateZone rewriteZoneFile id:%s failed:%s", req.Id, err.Error())
		}

		if err := handler.rndcModifyZone(req.Name, req.ZoneFileName, req.View); err != nil {
			return fmt.Errorf("UpdateZone rndcAddZone id:%s failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteZone(req *pb.DeleteZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(
			resource.TableZone,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("DeleteZone zone id:%s from db failed:%s", req.Id, err.Error())
		}

		if err := handler.rndcDeleteZone(req.Name, req.View); err != nil {
			return fmt.Errorf("DeleteZone id:%s rndcDeleteZone view:%s failed:%s", req.Id, req.View, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateForwardZone(req *pb.CreateForwardZoneReq) error {
	forwardZone := &resource.AgentForwardZone{
		Name:        req.Name,
		ForwardType: req.ForwardType,
		ForwardIds:  req.ForwardIds,
		AgentView:   req.ViewId,
		Ips:         req.ForwardIps,
	}
	forwardZone.SetID(req.Id)

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(forwardZone); err != nil {
			return fmt.Errorf("CreateForwardZone insert id:%s to db failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("CreateForwardZone id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateForwardZone(req *pb.UpdateForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableForwardZone, map[string]interface{}{
			"forward_type": req.ForwardType,
			"ips":          req.ForwardIps,
			"forward_ids":  req.ForwardIds,
		}, map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("UpdateForwardZone update forwardZone id:%s to db failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("UpdateForwardZone id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteForwardZone(req *pb.DeleteForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(
			resource.TableForwardZone,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("DeleteForwardZone delete id:%s failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("DeleteForwardZone id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func formatRdata(rr *string, datatype string) error {
	rrType, err := g53.TypeFromString(datatype)
	if err != nil {
		return fmt.Errorf("formatRdata datatype error:%s", err.Error())
	}

	rdata, err := g53.RdataFromString(rrType, *rr)
	if err != nil {
		return fmt.Errorf("formatRdata rdata error:%s", err.Error())
	}

	*rr = rdata.String()
	return nil
}

func generateRRset(rr *resource.AgentRr, zoneName string, RrsRole string) (*g53.RRset, error) {
	domainName := rr.Name + "." + zoneName
	if rr.Name == "@" {
		domainName = zoneName
	}
	name, err := g53.NameFromString(domainName)
	if err != nil {
		return nil, fmt.Errorf("generateRRset name:%s error:%s", domainName, err.Error())
	}
	ttl, err := g53.TTLFromString(strconv.FormatUint(uint64(rr.Ttl), 10))
	if err != nil {
		return nil, fmt.Errorf("generateRRset ttl:%d error:%s", rr.Ttl, err.Error())
	}
	cls, err := g53.ClassFromString("IN")
	if err != nil {
		return nil, fmt.Errorf("generateRRset cls:IN error:%s", err.Error())
	}
	rrType, err := g53.TypeFromString(rr.DataType)
	if err != nil {
		return nil, fmt.Errorf("generateRRset rrType:%s error:%s", rr.DataType, err.Error())
	}

	rdata, err := g53.RdataFromString(rrType, rr.Rdata)
	if err != nil {
		return nil, fmt.Errorf("generateRRset rdata:%s error:%s", rr.Rdata, err.Error())
	}
	if RrsRole == RoleBackup && rr.RdataBackup != "" {
		rdata, err = g53.RdataFromString(rrType, rr.RdataBackup)
		if err != nil {
			return nil, fmt.Errorf("generateRRset rdata:%s error:%s", rr.RdataBackup, err.Error())
		}
	}

	rrset := &g53.RRset{
		Name:   name,
		Type:   rrType,
		Class:  cls,
		Ttl:    ttl,
		Rdatas: []g53.Rdata{rdata},
	}

	return rrset, nil
}

func (handler *DNSHandler) CreateRR(req *pb.CreateRRReq) error {
	rr := &resource.AgentRr{
		Name:        req.Name,
		DataType:    req.DataType,
		Ttl:         uint(req.Ttl),
		Rdata:       req.RData,
		RdataBackup: req.BackupRData,
		ActiveRdata: req.RData,
		AgentView:   req.ViewId,
		Zone:        req.ZoneId,
	}
	rr.SetID(req.Id)

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(rr); err != nil {
			return fmt.Errorf("CreateRR insert id:%s to dbfailed:%s", req.Id, err.Error())
		}

		rrset, err := generateRRset(rr, req.ZoneName, req.ZoneRrsRole)
		if err != nil {
			return fmt.Errorf("CreateRR generateRRset failed:%s", err.Error())
		}

		if err := updateRR("key"+req.ViewId, req.ViewKey, rrset, req.ZoneName, true); err != nil {
			return fmt.Errorf("updateRR %s error:%s", rrset.String(), err.Error())
		}

		if err := handler.rndcZoneDumpJNLFile(req.ZoneName, req.ViewId); err != nil {
			return fmt.Errorf("CreateRR rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateRRsByZone(req *pb.UpdateRRsByZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var rrList []*resource.AgentRr
		if err := dbhandler.ListWithTx(&rrList, tx); err != nil {
			return fmt.Errorf("UpdateRRsByZone list rr from db failed:%s ", err.Error())
		}
		if _, err := tx.Update(
			resource.TableZone,
			map[string]interface{}{"rrs_role": req.NewRrsRole},
			map[string]interface{}{restdb.IDField: req.ZoneId}); err != nil {
			return fmt.Errorf("UpdateRRsByZone update role failed:%s", err.Error())
		}

		for _, rr := range rrList {
			rrset, err := generateRRset(rr, req.ZoneName, req.OldRrsRole)
			if err != nil {
				return fmt.Errorf("UpdateRRsByZone generateRRset failed:%s", err.Error())
			}
			if err := updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, false); err != nil {
				return fmt.Errorf("UpdateRRsByZone updateRR delete rrset:%s error:%s", rrset.String(), err.Error())
			}

			rrset, err = generateRRset(rr, req.ZoneName, req.NewRrsRole)
			if err != nil {
				return fmt.Errorf("UpdateRRsByZone generateRRset failed:%s", err.Error())
			}
			if err := updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, true); err != nil {
				return fmt.Errorf("UpdateRRsByZone updateRR add rrset:%s error:%s", rrset.String(), err.Error())
			}
		}

		if err := handler.rndcDumpJNLFile(); err != nil {
			return fmt.Errorf("UpdateRRsByZone rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateAllRRTtl(ttl uint32, tx restdb.Transaction) error {
	var rrList []*resource.AgentRr
	if err := dbhandler.ListWithTx(&rrList, tx); err != nil {
		return fmt.Errorf("UpdateAllRRTtl List rr falied:%s", err.Error())
	}

	for _, rr := range rrList {
		viewRes, err := dbhandler.GetWithTx(rr.AgentView, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateAllRRTtl Get views rr.id:%s falied:%s", rr.ID, err.Error())
		}
		zoneRes, err := dbhandler.GetWithTx(rr.Zone, &[]*resource.AgentZone{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateAllRRTtl Get zones rr.id:%s falied:%s", rr.ID, err.Error())
		}
		view := viewRes.(*resource.AgentView)
		zone := zoneRes.(*resource.AgentZone)
		if _, err := tx.Exec(updateRRsTTLSQL, ttl); err != nil {
			return fmt.Errorf("update updateRR to db failed:%s", err.Error())
		}

		rrset, err := generateRRset(rr, zone.Name, zone.RrsRole)
		if err != nil {
			return fmt.Errorf("UpdateAllRRTtl generateRRset failed:%s", err.Error())
		}
		if err := updateRR("key"+view.Name, view.Key, rrset, zone.Name, false); err != nil {
			return fmt.Errorf("UpdateAllRRTtl delete rrset:%s error:%s", rrset.String(), err.Error())
		}

		rr.Ttl = uint(ttl)
		newRRset, err := generateRRset(rr, zone.Name, zone.RrsRole)
		if err != nil {
			return fmt.Errorf("UpdateAllRRTtl generateRRset failed:%s", err.Error())
		}
		if err := updateRR("key"+view.Name, view.Key, newRRset, zone.Name, true); err != nil {
			return fmt.Errorf("UpdateAllRRTtl add rrset:%s error:%s", rrset.String(), err.Error())
		}

	}

	if err := handler.rndcDumpJNLFile(); err != nil {
		return fmt.Errorf("UpdateAllRRTtl rndcDumpJNLFile error:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) UpdateRR(req *pb.UpdateRRReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		rrRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentRr{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateRR get rr id:%s from db failed:%s", req.Id, err.Error())
		}
		rr := rrRes.(*resource.AgentRr)
		rrset, err := generateRRset(rr, req.ZoneName, req.ZoneRrsRole)
		if err != nil {
			return fmt.Errorf("UpdateRR generateRRset failed:%s", err.Error())
		}
		if err := updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, false); err != nil {
			return fmt.Errorf("updateRR delete rrset:%s error:%s", rrset.String(), err.Error())
		}

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

		rr.DataType = req.DataType
		rr.Ttl = uint(req.Ttl)
		rr.Rdata = req.RData
		rr.RdataBackup = req.BackupRData
		rrset, err = generateRRset(rr, req.ZoneName, req.ZoneRrsRole)
		if err != nil {
			return fmt.Errorf("UpdateRR generateRRset failed:%s", err.Error())
		}
		if err := updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, true); err != nil {
			return fmt.Errorf("updateRR add rrset:%s error:%s", rrset.String(), err.Error())
		}

		if err := handler.rndcZoneDumpJNLFile(req.ZoneName, req.ViewName); err != nil {
			return fmt.Errorf("UpdateRR rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteRR(req *pb.DeleteRRReq) error {
	rr := &resource.AgentRr{
		Name:        req.Name,
		DataType:    req.DataType,
		Ttl:         uint(req.Ttl),
		Rdata:       req.RData,
		RdataBackup: req.BackupRData,
		Zone:        req.ZoneId,
		AgentView:   req.ViewName,
	}
	rr.SetID(req.Id)

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableRR, map[string]interface{}{restdb.IDField: rr.ID}); err != nil {
			return fmt.Errorf("delete rr id:%s from db failed:%s", rr.ID, err.Error())
		}
		rrset, err := generateRRset(rr, req.ZoneName, req.ZoneRrsRole)
		if err != nil {
			return fmt.Errorf("DeleteRR generateRRset failed:%s", err.Error())
		}

		if err := updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, false); err != nil {
			return fmt.Errorf("DeleteRR delete rrset:%s error:%s", rrset.String(), err.Error())
		}

		if err := handler.rndcZoneDumpJNLFile(req.ZoneName, req.ViewName); err != nil {
			return fmt.Errorf("DeleteRR rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateForward(req *pb.CreateForwardReq) error {
	forward := &resource.AgentForward{
		Name: req.Name,
		Ips:  req.Ips,
	}
	forward.SetID(req.Id)

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(forward); err != nil {
			return fmt.Errorf("CreateForward insert id:%s to db failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateForward(req *pb.UpdateForwardReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableForward,
			map[string]interface{}{"ips": req.Ips},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("update forward id:%s to db failed:%s", req.Id, err.Error())
		}
		for _, zone := range req.ForwardZones {
			if _, err := tx.Update(resource.TableForwardZone,
				map[string]interface{}{"ips": zone.ForwardIps},
				map[string]interface{}{restdb.IDField: zone.Id}); err != nil {
				return fmt.Errorf("update forwardZone id:%s to db failed:%s", zone.Id, err.Error())
			}
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("UpdateForward id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteForward(req *pb.DeleteForwardReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableForward, map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete forward id:%s from db failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("DeleteForward id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateIPBlackHole(req *pb.CreateIPBlackHoleReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		ipBlackHole := &resource.AgentIpBlackHole{Acl: req.Acl}
		ipBlackHole.SetID(req.Id)
		if _, err := tx.Insert(ipBlackHole); err != nil {
			return fmt.Errorf("insert ipBlackHole to db failed:%s", err.Error())
		}

		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("CreateIPBlackHole id:%s rewriteNamedOptionsFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateIPBlackHole(req *pb.UpdateIPBlackHoleReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableIpBlackHole,
			map[string]interface{}{"acl": req.Acl},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("insert ipBlackHole to db failed:%s", err.Error())
		}
		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("UpdateIPBlackHole id:%s rewriteNamedOptionsFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteIPBlackHole(req *pb.DeleteIPBlackHoleReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableIpBlackHole,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete ipBlackHole to db failed:%s", err.Error())
		}
		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("DeleteIPBlackHole id:%s rewriteNamedOptionsFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateRecursiveConcurrent(req *pb.UpdateRecurConcuReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
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

		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("UpdateRecursiveConcurrent id:%s rewriteNamedOptionsFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateSortList(req *pb.CreateSortListReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("CreateSortList  rewriteNamedOptionsFile failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateSortList(req *pb.UpdateSortListReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("UpdateSortList rewriteNamedOptionsFile failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteSortList(req *pb.DeleteSortListReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("DeleteSortList rewriteNamedOptionsFile failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateUrlRedirect(req *pb.CreateUrlRedirectReq) error {
	urlRedirect := &resource.AgentUrlRedirect{
		Domain:    req.Domain,
		Url:       req.Url,
		AgentView: req.ViewId,
	}
	urlRedirect.SetID(req.Id)

	redirection := &resource.AgentRedirection{
		Name:         urlRedirect.Domain,
		Ttl:          3600,
		DataType:     "A",
		Rdata:        handler.localip,
		RedirectType: localZoneType,
		AgentView:    urlRedirect.AgentView,
	}
	redirection.SetID(req.Id)
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		exist, _ := tx.Exists(resource.TableRedirection,
			map[string]interface{}{"name": urlRedirect.Domain})
		if exist {
			return fmt.Errorf("CreateUrlRedirect insert urlredirect id:%s to db failed:domain:%s in redirection has exist", req.Id, req.Domain)
		}

		if _, err := tx.Insert(urlRedirect); err != nil {
			return fmt.Errorf("CreateUrlRedirect insert urlredirect id:%s to db failed:%s", req.Id, err.Error())
		}

		if _, err := tx.Insert(redirection); err != nil {
			return fmt.Errorf("CreateUrlRedirect insert redirect id:%s to db failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteOneRPZFile(urlRedirect.AgentView, tx); err != nil {
			return fmt.Errorf("CreateUrlRedirect id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		if err := handler.rewriteNginxFile(tx); err != nil {
			return fmt.Errorf("CreateUrlRedirect rewrite nginx default config for %s and %s error:%s", urlRedirect.Domain, urlRedirect.Url, err.Error())
		}
		if err := handler.nginxReload(); err != nil {
			return fmt.Errorf("CreateUrlRedirect nginx reload error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateUrlRedirect(req *pb.UpdateUrlRedirectReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableUrlRedirect,
			map[string]interface{}{"domain": req.Domain, "url": req.Url},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("update urlredirect id:%s to db failed:%s", req.Id, err.Error())
		}

		if _, err := tx.Update(resource.TableRedirection,
			map[string]interface{}{"name": req.Domain},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("update redirection id:%s to db failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteOneRPZFile(req.View, tx); err != nil {
			return fmt.Errorf("UpdateUrlRedirect id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteNginxFile(tx); err != nil {
			return fmt.Errorf("UpdateUrlRedirect rewrite nginx default config for %s and %s error:%s", req.Domain, req.Url, err.Error())
		}
		if err := handler.nginxReload(); err != nil {
			return fmt.Errorf("UpdateUrlRedirect nginx reload error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteUrlRedirect(req *pb.DeleteUrlRedirectReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableUrlRedirect,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete urlRedirect id:%s from db failed:%s", req.Id, err.Error())
		}

		if _, err := tx.Delete(resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete Redirecttion id:%s from db failed:%s", req.Id, err.Error())
		}
		if err := handler.rewriteOneRPZFile(req.View, tx); err != nil {
			return fmt.Errorf("DeleteUrlRedirect id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		if err := handler.rewriteNginxFile(tx); err != nil {
			return fmt.Errorf("DeleteUrlRedirect rewrite nginxconfig for %s error:%s", req.Id, err.Error())
		}
		if err := handler.nginxReload(); err != nil {
			return fmt.Errorf("DeleteUrlRedirect nginx reload error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateGlobalConfig(req *pb.UpdateGlobalConfigReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
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

			if err := handler.UpdateAllRRTtl(req.Ttl, tx); err != nil {
				return err
			}

			if err := handler.initRPZFile(tx); err != nil {
				return fmt.Errorf("UpdateGlobalConfig initRPZFile failed:%s", err.Error())
			}
			if err := handler.initRedirectFile(tx); err != nil {
				return fmt.Errorf("UpdateGlobalConfig  rewriteNamedViewFile failed:%s", err.Error())
			}
			if err := handler.rndcReconfig(); err != nil {
				return fmt.Errorf("UpdateGlobalConfig  rndcReconfig failed:%s", err.Error())
			}

		}

		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("UpdateGlobalConfig rewriteNamedOptionsFile failed:%s", err.Error())
		}
		return nil
	})
}
