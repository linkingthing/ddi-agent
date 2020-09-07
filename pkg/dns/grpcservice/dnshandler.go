package grpcservice

import (
	"encoding/base64"
	"fmt"
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

	if err := initDefaultDbData(); err != nil {
		return nil, fmt.Errorf("initDefaultDbData failed:%s", err.Error())
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

	if err := handler.initFiles(); err != nil {
		return err
	}

	path := filepath.Join(handler.dnsConfPath, "named.log")
	if pathExists(path) {
		if err := os.Remove(path); err != nil {
			log.Errorf("remove  named.log fail:%s", err.Error())
		}
	}
	var param = "-c" + filepath.Join(handler.dnsConfPath, mainConfName)
	var logParam = "-L" + "named.log"
	if _, err := shell.Shell(filepath.Join(handler.dnsConfPath, "named"), param, logParam); err != nil {
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

func (handler *DNSHandler) initFiles() error {
	path := filepath.Join(handler.dnsConfPath, "redirection")
	if !pathExists(path) {
		if err := os.Mkdir(path, 0644); err != nil {
			log.Errorf("create redirection fail:%s", err.Error())
		}
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

func formatDomain(name *string, datatype string, redirectType string) error {
	if datatype == ptrType || redirectType == localZoneType {
		return nil
	}

	n, err := g53.NameFromString(*name)
	if err != nil {
		return fmt.Errorf("formatDomain failed:%s", err.Error())
	}

	*name = n.String(false)
	return nil
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
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		acl := &resource.AgentAcl{Name: req.Name, Ips: req.Ips}
		acl.SetID(req.Id)
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
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		view := &resource.AgentView{
			Name:     req.Name,
			Priority: uint(req.Priority),
			Acls:     req.Acls,
			Dns64:    req.Dns64,
		}
		view.SetID(req.Id)
		key, _ := uuid2.Gen()
		view.Key = base64.StdEncoding.EncodeToString([]byte(key))

		if err := adjustPriority(view, tx, false); err != nil {
			return fmt.Errorf("CreateView id:%s Insert to db failed:%s", req.Id, err.Error())
		}
		if _, err := tx.Insert(view); err != nil {
			return fmt.Errorf("CreateView id:%s Insert to db failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("CreateView id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}

		return nil
	})
}

func (handler *DNSHandler) UpdateView(req *pb.UpdateViewReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		viewRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateView id:%s get db failed:%s", req.Id, err.Error())
		}
		view := viewRes.(*resource.AgentView)
		view.Priority = uint(req.Priority)
		view.Acls = req.Acls
		view.Dns64 = req.Dns64

		if err := adjustPriority(view, tx, false); err != nil {
			return fmt.Errorf("UpdateView id:%s update to db failed:%s", req.Id, err.Error())
		}
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

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("UpdateView id:%s rewriteNamedViewFile failed:%s", req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteView(req *pb.DeleteViewReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		viewRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("DeleteView get view failed:%s", err.Error())
		}
		if err := adjustPriority(viewRes.(*resource.AgentView), tx, true); err != nil {
			return fmt.Errorf("DeleteView adjust priority when delete view failed:%s", err.Error())
		}
		if _, err := tx.Delete(resource.TableView, map[string]interface{}{
			restdb.IDField: req.Id,
		}); err != nil {
			return fmt.Errorf("DeleteView delete view %s from db failed: %s", req.Id, err.Error())
		}
		if _, err := tx.Delete(resource.TableZone, map[string]interface{}{
			"view": req.Id,
		}); err != nil {
			return fmt.Errorf("DeleteView delete zone viewID:%s from db failed: %s", req.Id, err.Error())
		}
		if _, err := tx.Delete(resource.TableRR, map[string]interface{}{
			"view": req.Id,
		}); err != nil {
			return fmt.Errorf("DeleteView delete rr viewID:%s from db failed: %s", req.Id, err.Error())
		}
		if _, err := tx.Delete(resource.TableRedirection, map[string]interface{}{
			"view": req.Id,
		}); err != nil {
			return fmt.Errorf("DeleteView delete redirection viewID:%s from db failed: %s", req.Id, err.Error())
		}
		if _, err := tx.Delete(resource.TableUrlRedirect, map[string]interface{}{
			"view": req.Id,
		}); err != nil {
			return fmt.Errorf("DeleteView delete urlredirect viewID:%s from db failed: %s", req.Id, err.Error())
		}
		if _, err := tx.Delete(resource.TableForwardZone, map[string]interface{}{
			"view": req.Id,
		}); err != nil {
			return fmt.Errorf("DeleteView delete forwardzone viewID:%s from db failed: %s", req.Id, err.Error())
		}

		if err := removeFiles(filepath.Join(handler.dnsConfPath), req.Id+"_", zoneSuffix); err != nil {
			return fmt.Errorf("DeleteView zonefile in %s err: %s", filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}
		if err := RemoveOneFile(filepath.Join(handler.dnsConfPath, req.Id) + nzfSuffix); err != nil {
			return fmt.Errorf("DeleteView delete nzf failed:%s", err.Error())
		}
		if err := RemoveOneFile(filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+req.Id)); err != nil {
			return fmt.Errorf("DeleteView delete rpz failed:%s", err.Error())
		}
		if err := RemoveOneFile(filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+req.Id)); err != nil {
			return fmt.Errorf("DeleteView delete redirect failed:%s", err.Error())
		}
		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("DeleteView rewriteNamedViewFile  failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateRedirection(req *pb.CreateRedirectionReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		redirect := &resource.AgentRedirection{
			Name:         req.Name,
			Ttl:          uint(req.Ttl),
			DataType:     req.DataType,
			RedirectType: req.RedirectType,
			Rdata:        req.RData,
			View:         req.ViewId,
		}
		redirect.SetID(req.Id)
		if err := formatDomainName(&redirect.Rdata, redirect.DataType); err != nil {
			return err
		}
		if err := formatDomain(&redirect.Name, redirect.DataType, redirect.RedirectType); err != nil {
			return err
		}
		if _, err := tx.Insert(redirect); err != nil {
			return err
		}

		if redirect.RedirectType == localZoneType {
			if err := handler.rewriteOneRPZFile(redirect.View, tx); err != nil {
				return fmt.Errorf("CreateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}
		} else {
			if err := handler.rewriteOneRedirectFile(redirect.View, tx); err != nil {
				return fmt.Errorf("CreateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
			}
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateRedirection(req *pb.UpdateRedirectionReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		redirectRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentRedirection{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateRedirection id:%s Get redirection from db failed:%s", req.Id, err.Error())
		}
		redirectTypeChanged := false
		redirection := redirectRes.(*resource.AgentRedirection)
		redirection.DataType = req.DataType
		redirection.Rdata = req.RData
		redirection.Ttl = uint(req.Ttl)
		if redirection.RedirectType != req.RedirectType {
			redirectTypeChanged = true
		}
		redirection.RedirectType = req.RedirectType
		if err := formatDomainName(&redirection.Rdata, redirection.DataType); err != nil {
			return err
		}
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

		if redirection.RedirectType == nxDomain {
			if err := handler.rewriteOneRedirectFile(redirection.View, tx); err != nil {
				return fmt.Errorf("UpdateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
			}

			if redirectTypeChanged {
				if err := handler.rewriteOneRPZFile(redirection.View, tx); err != nil {
					return fmt.Errorf("UpdateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
				}
			}
		} else if redirection.RedirectType == localZoneType {
			if err := handler.rewriteOneRPZFile(redirection.View, tx); err != nil {
				return fmt.Errorf("UpdateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}

			if redirectTypeChanged {
				if err := handler.rewriteOneRedirectFile(redirection.View, tx); err != nil {
					return fmt.Errorf("UpdateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
				}
			}
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteRedirection(req *pb.DeleteRedirectionReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		redirectRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentRedirection{}, tx)
		if err != nil {
			return fmt.Errorf("DeleteRedirection get redirect id:%s failed:%s", req.Id, err.Error())
		}
		redirection := redirectRes.(*resource.AgentRedirection)
		if _, err := tx.Delete(
			resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("DeleteRedirection insert id:%s to db failed:%s", req.Id, err.Error())
		}

		if redirection.RedirectType == localZoneType {
			if err := handler.rewriteOneRPZFile(redirection.View, tx); err != nil {
				return fmt.Errorf("DeleteRedirection UpdateRedirection id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}
		} else {
			if err := handler.rewriteOneRedirectFile(redirection.View, tx); err != nil {
				return fmt.Errorf("DeleteRedirection UpdateRedirection id:%s rewriteRedirectFile failed:%s", req.Id, err.Error())
			}
		}
		return nil
	})
}

func (handler *DNSHandler) CreateZone(req *pb.CreateZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zone := &resource.AgentZone{
			Name:     req.ZoneName,
			ZoneFile: req.ZoneFileName,
			Ttl:      uint(req.Ttl),
			View:     req.ViewId,
			RrsRole:  req.RrsRole,
		}
		zone.SetID(req.ZoneId)

		if _, err := tx.Insert(zone); err != nil {
			return fmt.Errorf("CreateZone id:%s failed:%s", req.ZoneId, err.Error())
		}

		if err := handler.createZoneFile(zone); err != nil {
			return fmt.Errorf("CreateZone rewriteZoneFile id:%s failed:%s", zone.ID, err.Error())
		}

		if err := handler.rndcAddZone(zone.Name, zone.ZoneFile, zone.View); err != nil {
			return fmt.Errorf("CreateZone rndcAddZone id:%s failed:%s", zone.ID, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateZone(req *pb.UpdateZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zoneRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentZone{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateZone get zone id:%s failed:%s", req.Id, err.Error())
		}
		zone := zoneRes.(*resource.AgentZone)
		zone.Ttl = uint(req.Ttl)
		if _, err := tx.Update(
			resource.TableZone,
			map[string]interface{}{"ttl": zone.Ttl},
			map[string]interface{}{restdb.IDField: zone.ID},
		); err != nil {
			return fmt.Errorf("UpdateZone id:%s failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteOneZoneFile(req.Id, zone.ZoneFile, tx); err != nil {
			return fmt.Errorf("UpdateZone rewriteZoneFile id:%s failed:%s", req.Id, err.Error())
		}

		if err := handler.rndcModZone(zone.Name, zone.ZoneFile, zone.View); err != nil {
			return fmt.Errorf("UpdateZone rndcAddZone id:%s failed:%s", zone.ID, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteZone(req *pb.DeleteZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zoneRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentZone{}, tx)
		if err != nil {
			return fmt.Errorf("DeleteZone get zone from db failed:%s", req.Id)
		}
		zone := zoneRes.(*resource.AgentZone)
		if _, err := tx.Delete(
			resource.TableZone,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("DeleteZone zone id:%s from db failed:%s", req.Id, err.Error())
		}

		if err := handler.rndcDelZone(zone.Name, zone.View); err != nil {
			return fmt.Errorf("DeleteZone id:%s rndcDelZone view:%s failed:%s", req.Id, zone.View, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateForwardZone(req *pb.CreateForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		forwardZone := &resource.AgentForwardZone{
			Name:        req.Name,
			ForwardType: req.ForwardType,
			ForwardIds:  req.ForwardIds,
			View:        req.ViewId,
		}
		forwardZone.SetID(req.Id)

		var forwardList []*resource.AgentForward
		sql := fmt.Sprintf(`select * from gr_agent_forward where id in ('%s')`, strings.Join(req.ForwardIds, "','"))
		if err := tx.FillEx(&forwardList, sql); err != nil {
			return fmt.Errorf("CreateForwardZone get forward ids:%s from db failed:%s", req.ForwardIds, err.Error())
		}
		if len(forwardList) == 0 {
			return fmt.Errorf("CreateForwardZone get forward ids:%s from db failed:len(forwards)==0", req.ForwardIds)
		}

		for _, value := range forwardList {
			forwardZone.Ips = append(forwardZone.Ips, value.Ips...)
		}

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
		forwardZoneRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentForwardZone{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateForwardZone get forwardzone id:%s failed:%s", req.Id, err.Error())
		}
		forwardZone, ok := forwardZoneRes.(*resource.AgentForwardZone)
		if !ok {
			return fmt.Errorf("UpdateForwardZone inflect forwardzone failed")
		}
		updateMap := make(map[string]interface{})
		updateMap["forward_type"] = req.ForwardType
		if isSlicesDiff(forwardZone.ForwardIds, req.ForwardIds) {
			var forwardList []*resource.AgentForward
			sql := fmt.Sprintf(`select * from gr_agent_forward where id in ('%s')`, strings.Join(req.ForwardIds, "','"))
			if err := tx.FillEx(&forwardList, sql); err != nil {
				return fmt.Errorf("UpdateForwardZone get forward ids:%s from db failed:%s", req.ForwardIds, err.Error())
			}
			if len(forwardList) == 0 {
				return fmt.Errorf("UpdateForwardZone get forward ids:%s from db failed:len(forwards)==0", req.ForwardIds)
			}
			for _, value := range forwardList {
				forwardZone.Ips = append(forwardZone.Ips, value.Ips...)
			}
			updateMap["ips"] = forwardZone.Ips
			updateMap["forward_ids"] = req.ForwardIds
		}

		if _, err := tx.Update(resource.TableForwardZone, updateMap,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
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

func formatDomainName(rr *string, datatype string) error {
	rrType, err := g53.TypeFromString(datatype)
	if err != nil {
		return fmt.Errorf("formatDomainName datatype error:%s", err.Error())
	}

	rdata, err := g53.RdataFromString(rrType, *rr)
	if err != nil {
		return fmt.Errorf("formatDomainName rdata error:%s", err.Error())
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

	var rdata g53.Rdata
	if RrsRole == RoleBackup && rr.RdataBackup != "" {
		rdata, err = g53.RdataFromString(rrType, rr.Rdata)
		if err != nil {
			return nil, fmt.Errorf("generateRRset rdata:%s error:%s", rr.Rdata, err.Error())
		}
	} else {
		rdata, err = g53.RdataFromString(rrType, rr.Rdata)
		if err != nil {
			return nil, fmt.Errorf("generateRRset rdata:%s error:%s", rr.Rdata, err.Error())
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
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
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
		if _, err := tx.Insert(rr); err != nil {
			return fmt.Errorf("CreateRR insert id:%s to dbfailed:%s", req.Id, err.Error())
		}

		viewRes, err := dbhandler.GetWithTx(req.ViewId, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("CreateRR get view id:%s to dbfailed:%s", req.ViewId, err.Error())
		}
		zoneRes, err := dbhandler.GetWithTx(req.ZoneId, &[]*resource.AgentZone{}, tx)
		if err != nil {
			return fmt.Errorf("CreateRR get view id:%s to dbfailed:%s", req.ZoneId, err.Error())
		}
		view := viewRes.(*resource.AgentView)
		zone := zoneRes.(*resource.AgentZone)
		rrset, err := generateRRset(rr, zone.Name, zone.RrsRole)
		if err != nil {
			return fmt.Errorf("CreateRR generateRRset failed:%s", err.Error())
		}

		if err := updateRR("key"+view.Name, view.Key, rrset, zone.Name, true); err != nil {
			return fmt.Errorf("updateRR %s error:%s", rrset.String(), err.Error())
		}

		if err := handler.rndcZoneDumpJNLFile(zone.Name, view.Name); err != nil {
			return fmt.Errorf("CreateRR rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateRRsByZone(req *pb.UpdateRRsByZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		zoneRes, err := dbhandler.GetWithTx(req.ZoneId, &[]*resource.AgentZone{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateRRsByZone get zone from db failed:%s ", err.Error())
		}
		zone, ok := zoneRes.(*resource.AgentZone)
		if !ok {
			return fmt.Errorf("UpdateRRsByZone inflect zone failed")
		}
		if zone.RrsRole == req.Role {
			return nil
		}

		viewRes, err := dbhandler.GetWithTx(zone.View, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateRRsByZone get view from db failed:%s ", err.Error())
		}

		var rrList []*resource.AgentRr
		if err := dbhandler.ListWithTx(&rrList, tx); err != nil {
			return fmt.Errorf("UpdateRRsByZone list rr from db failed:%s ", err.Error())
		}
		view, ok := viewRes.(*resource.AgentView)
		if !ok {
			return fmt.Errorf("UpdateRRsByZone inflect view failed")
		}

		if _, err := tx.Update(
			resource.TableZone,
			map[string]interface{}{"rrs_role": req.Role},
			map[string]interface{}{restdb.IDField: req.ZoneId}); err != nil {
			return fmt.Errorf("UpdateRRsByZone update role failed:%s", err.Error())
		}

		for _, rr := range rrList {
			rrset, err := generateRRset(rr, zone.Name, zone.RrsRole)
			if err != nil {
				return fmt.Errorf("UpdateRRsByZone generateRRset failed:%s", err.Error())
			}

			if err := updateRR("key"+view.Name, view.Key, rrset, zone.Name, false); err != nil {
				return fmt.Errorf("UpdateRRsByZone updateRR delete rrset:%s error:%s", rrset.String(), err.Error())
			}

			if err := updateRR("key"+view.Name, view.Key, rrset, zone.Name, true); err != nil {
				return fmt.Errorf("UpdateRRsByZone updateRR add rrset:%s error:%s", rrset.String(), err.Error())
			}
		}

		if err := handler.rndcDumpJNLFile(); err != nil {
			return fmt.Errorf("UpdateRRsByZone rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateAllRRTtl(ttl uint32) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var rrList []*resource.AgentRr
		if err := dbhandler.ListWithTx(&rrList, tx); err != nil {
			return fmt.Errorf("UpdateAllRRTtl List rr falied:%s", err.Error())
		}

		for _, rr := range rrList {
			viewRes, err := dbhandler.GetWithTx(rr.View, &[]*resource.AgentView{}, tx)
			if err != nil {
				return fmt.Errorf("UpdateAllRRTtl Get views falied:%s", err.Error())
			}
			zoneRes, err := dbhandler.GetWithTx(rr.Zone, &[]*resource.AgentZone{}, tx)
			if err != nil {
				return fmt.Errorf("UpdateAllRRTtl Get zones falied:%s", err.Error())
			}
			view := viewRes.(*resource.AgentView)
			zone := zoneRes.(*resource.AgentZone)
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
	})
}

func (handler *DNSHandler) UpdateRR(req *pb.UpdateRRReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		rrRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentRr{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateRR get rr id:%s from db failed:%s", req.Id, err.Error())
		}
		rr := rrRes.(*resource.AgentRr)
		viewRes, err := dbhandler.GetWithTx(rr.View, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateRR get view id:%s from db failed:%s", rr.View, err.Error())
		}
		zoneRes, err := dbhandler.GetWithTx(rr.Zone, &[]*resource.AgentZone{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateRR get zone id:%s from db failed:%s", rr.Zone, err.Error())
		}
		view := viewRes.(*resource.AgentView)
		zone := zoneRes.(*resource.AgentZone)
		rrset, err := generateRRset(rr, zone.Name, zone.RrsRole)
		if err != nil {
			return fmt.Errorf("UpdateRR generateRRset failed:%s", err.Error())
		}
		if err := updateRR("key"+view.Name, view.Key, rrset, zone.Name, false); err != nil {
			return fmt.Errorf("updateRR delete rrset:%s error:%s", rrset.String(), err.Error())
		}

		rr.DataType = req.DataType
		rr.Ttl = uint(req.Ttl)
		rr.Rdata = req.RData
		rr.RdataBackup = req.BackupRData
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

		rrset, err = generateRRset(rr, zone.Name, zone.RrsRole)
		if err != nil {
			return fmt.Errorf("UpdateRR generateRRset failed:%s", err.Error())
		}
		if err := updateRR("key"+view.Name, view.Key, rrset, zone.Name, true); err != nil {
			return fmt.Errorf("updateRR add rrset:%s error:%s", rrset.String(), err.Error())
		}

		if err := handler.rndcZoneDumpJNLFile(zone.Name, view.Name); err != nil {
			return fmt.Errorf("UpdateRR rndcDumpJNLFile error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteRR(req *pb.DeleteRRReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		rrRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentRr{}, tx)
		if err != nil {
			return fmt.Errorf("DeleteRR get rr id:%s from db failed:%s", req.Id, err.Error())
		}
		rr := rrRes.(*resource.AgentRr)
		if _, err := tx.Delete(resource.TableRR, map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete rr id:%s from db failed:%s", req.Id, err.Error())
		}

		viewRes, err := dbhandler.GetWithTx(rr.View, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("DeleteRR get view id:%s from db failed:%s", rr.View, err.Error())
		}
		zoneRes, err := dbhandler.GetWithTx(rr.Zone, &[]*resource.AgentZone{}, tx)
		if err != nil {
			return fmt.Errorf("DeleteRR get zone id:%s from db failed:%s", rr.Zone, err.Error())
		}
		view := viewRes.(*resource.AgentView)
		zone := zoneRes.(*resource.AgentZone)
		rrset, err := generateRRset(rr, zone.Name, "")
		if err != nil {
			return fmt.Errorf("DeleteRR generateRRset failed:%s", err.Error())
		}

		if err := updateRR("key"+view.Name, view.Key, rrset, zone.Name, false); err != nil {
			return fmt.Errorf("DeleteRR delete rrset:%s error:%s", rrset.String(), err.Error())
		}

		if err := handler.rndcZoneDumpJNLFile(zone.Name, view.Name); err != nil {
			return fmt.Errorf("CreateRR rndcDumpJNLFile error:%s", err.Error())
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

	tx, err := db.GetDB().Begin()
	defer tx.Rollback()

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
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		forwardRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentForward{}, tx)
		if err != nil {
			return err
		}

		forward, ok := forwardRes.(*resource.AgentForward)
		if !ok {
			return fmt.Errorf("inflect forward failed")
		}
		if !isSlicesDiff(forward.Ips, req.Ips) {
			return nil
		}
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
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	ipBlackHole := &resource.AgentIpBlackHole{Acl: req.Acl}
	ipBlackHole.SetID(req.Id)
	if _, err := tx.Insert(ipBlackHole); err != nil {
		return fmt.Errorf("insert ipBlackHole to db failed:%s", err.Error())
	}

	if err := handler.rewriteNamedOptionsFile(tx); err != nil {
		return fmt.Errorf("CreateIPBlackHole id:%s rewriteNamedOptionsFile failed:%s", req.Id, err.Error())
	}

	return tx.Commit()
}

func (handler *DNSHandler) UpdateIPBlackHole(req *pb.UpdateIPBlackHoleReq) error {
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	if _, err := tx.Update(resource.TableIpBlackHole,
		map[string]interface{}{"acl": req.Acl},
		map[string]interface{}{restdb.IDField: req.Id}); err != nil {
		return fmt.Errorf("insert ipBlackHole to db failed:%s", err.Error())
	}
	if err := handler.rewriteNamedOptionsFile(tx); err != nil {
		return fmt.Errorf("UpdateIPBlackHole id:%s rewriteNamedOptionsFile failed:%s", req.Id, err.Error())
	}

	return tx.Commit()
}

func (handler *DNSHandler) DeleteIPBlackHole(req *pb.DeleteIPBlackHoleReq) error {
	tx, _ := db.GetDB().Begin()
	defer tx.Rollback()
	if _, err := tx.Delete(resource.TableIpBlackHole,
		map[string]interface{}{restdb.IDField: req.Id}); err != nil {
		return fmt.Errorf("delete ipBlackHole to db failed:%s", err.Error())
	}
	if err := handler.rewriteNamedOptionsFile(tx); err != nil {
		return fmt.Errorf("DeleteIPBlackHole id:%s rewriteNamedOptionsFile failed:%s", req.Id, err.Error())
	}

	return tx.Commit()
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
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		urlRedirect := &resource.AgentUrlRedirect{
			Domain: req.Domain,
			Url:    req.Url,
			View:   req.ViewId,
		}
		urlRedirect.SetID(req.Id)

		exist, _ := tx.Exists(resource.TableRedirection,
			map[string]interface{}{"name": urlRedirect.Domain})
		if exist {
			return fmt.Errorf("CreateUrlRedirect insert urlredirect id:%s to db failed:domain:%s in redirection has exist", req.Id, req.Domain)
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
			return fmt.Errorf("CreateUrlRedirect insert urlredirect id:%s to db failed:%s", req.Id, err.Error())
		}

		if _, err := tx.Insert(redirection); err != nil {
			return fmt.Errorf("CreateUrlRedirect insert redirect id:%s to db failed:%s", req.Id, err.Error())
		}

		if err := handler.rewriteOneRPZFile(urlRedirect.View, tx); err != nil {
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
		urlRedirectRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentUrlRedirect{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateUrlRedirect get urlredirect id:%s failed:%s", req.Id, err.Error())
		}
		urlRedirect := urlRedirectRes.(*resource.AgentUrlRedirect)
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

		if urlRedirect.Domain != req.Domain {
			if err := handler.rewriteOneRPZFile(urlRedirect.View, tx); err != nil {
				return fmt.Errorf("UpdateUrlRedirect id:%s rewriteRPZFile failed:%s", req.Id, err.Error())
			}
		}
		if err := handler.rewriteNginxFile(tx); err != nil {
			return fmt.Errorf("UpdateUrlRedirect rewrite nginx default config for %s and %s error:%s", urlRedirect.Domain, urlRedirect.Url, err.Error())
		}
		if err := handler.nginxReload(); err != nil {
			return fmt.Errorf("UpdateUrlRedirect nginx reload error:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteUrlRedirect(req *pb.DeleteUrlRedirectReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		urlRedirectRes, err := dbhandler.Get(req.Id, &[]*resource.AgentUrlRedirect{})
		if err != nil {
			return fmt.Errorf("DeleteUrlRedirect get id:%s failed:%s", req.Id, err.Error())
		}
		urlRedirect := urlRedirectRes.(*resource.AgentUrlRedirect)

		if _, err := tx.Delete(resource.TableUrlRedirect,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete urlRedirect id:%s from db failed:%s", req.Id, err.Error())
		}

		if _, err := tx.Delete(resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete Redirecttion id:%s from db failed:%s", req.Id, err.Error())
		}
		if err := handler.rewriteOneRPZFile(urlRedirect.View, tx); err != nil {
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
			if _, err := tx.Exec(updateRRsTTLSQL, req.Ttl); err != nil {
				return fmt.Errorf("update updateRRsTTLSQL to db failed:%s", err.Error())
			}

			if err := handler.UpdateAllRRTtl(req.Ttl); err != nil {
				return err
			}
			if err := handler.rewriteNamedViewFile(false, tx); err != nil {
				return fmt.Errorf("UpdateGlobalConfig rewriteNamedViewFile failed:%s", err.Error())
			}
			if err := handler.rewriteNamedViewFile(false, tx); err != nil {
				return fmt.Errorf("UpdateGlobalConfig  rewriteNamedViewFile failed:%s", err.Error())
			}
		}

		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("UpdateGlobalConfig rewriteNamedOptionsFile failed:%s", err.Error())
		}
		return nil
	})
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
