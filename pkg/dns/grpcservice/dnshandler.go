package grpcservice

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/g53"
	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/ddi-agent/config"
	"github.com/linkingthing/ddi-agent/pkg/db"
	"github.com/linkingthing/ddi-agent/pkg/dns/dbhandler"
	"github.com/linkingthing/ddi-agent/pkg/dns/resource"
	"github.com/linkingthing/ddi-agent/pkg/grpcclient"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	monitorpb "github.com/linkingthing/ddi-monitor/pkg/proto"
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
	TemplateDir                  = "/etc/dns/templates"

	updateZonesTTLSQL       = "update gr_agent_zone set ttl = $1"
	updateRRsTTLSQL         = "update gr_agent_rr set ttl = $1"
	updateRedirectionTtlSQL = "update gr_agent_redirection set ttl = $1"
)

type DNSHandler struct {
	tpl                 *template.Template
	dnsConfPath         string
	tplPath             string
	ticker              *time.Ticker
	quit                chan int
	nginxDefaultConfDir string
	localip             string
	localipv6           string
	dnsServer           string
	rndcConfPath        string
	rndcPath            string
	nginxConfPath       string
	namedViewPath       string
	namedOptionPath     string
	namedAclPath        string
}

func newDNSHandler(conf *config.AgentConfig) (*DNSHandler, error) {
	instance := &DNSHandler{
		dnsConfPath:         filepath.Join(conf.DNS.ConfDir),
		tplPath:             TemplateDir,
		nginxDefaultConfDir: conf.NginxDefaultDir,
		localip:             conf.Server.IP,
		localipv6:           conf.Server.IPV6,
		dnsServer:           conf.DNS.ServerIp + ":53",
	}

	instance.tpl = template.Must(template.ParseGlob(filepath.Join(instance.tplPath, "*.tpl")))
	instance.ticker = time.NewTicker(checkPeriod * time.Second)
	instance.quit = make(chan int)

	if err := instance.StartDNS(&pb.DNSStartReq{}); err != nil {
		return nil, err
	}

	return instance, nil
}

func (handler *DNSHandler) StartDNS(req *pb.DNSStartReq) error {
	if err := handler.reconfigOrStartDNS(true); err != nil {
		return err
	}

	go handler.keepDNSAlive()
	return nil
}

func (handler *DNSHandler) reconfigOrStartDNS(init bool) error {
	err := initDefaultDbData()
	if err != nil {
		return fmt.Errorf("initDefaultDbData failed:%s", err.Error())
	}

	if err := handler.initFiles(); err != nil {
		return err
	}

	if init {
		if resp, err := grpcclient.GetDDIMonitorGrpcClient().GetDNSState(context.Background(),
			&monitorpb.GetDNSStateRequest{}); err != nil {
			return err
		} else if resp.GetIsRunning() {
			err = handler.rndcReload()
		} else {
			_, err = grpcclient.GetDDIMonitorGrpcClient().StartDNS(context.Background(), &monitorpb.StartDNSRequest{})
		}
	} else {
		_, err = grpcclient.GetDDIMonitorGrpcClient().StartDNS(context.Background(), &monitorpb.StartDNSRequest{})
	}

	return err
}

func (handler *DNSHandler) StopDNS() error {
	if _, err := grpcclient.GetDDIMonitorGrpcClient().StopDNS(context.Background(), &monitorpb.StopDNSRequest{}); err != nil {
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
			if resp, err := grpcclient.GetDDIMonitorGrpcClient().GetDNSState(context.Background(),
				&monitorpb.GetDNSStateRequest{}); err != nil {
				continue
			} else if resp.GetIsRunning() == false {
				handler.reconfigOrStartDNS(false)
			}
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

		if exist, err := dbhandler.ExistWithTx(resource.TableAcl, anyACL, tx); err != nil {
			return fmt.Errorf("check agent_acl anyACL exist from db failed:%s", err.Error())
		} else if !exist {
			acl := &resource.AgentAcl{Name: anyACL}
			acl.SetID(anyACL)
			if _, err = tx.Insert(acl); err != nil {
				return fmt.Errorf("Insert agent_acl anyACL into db failed:%s ", err.Error())
			}
		}

		if exist, err := dbhandler.ExistWithTx(resource.TableView, defaultView, tx); err != nil {
			return fmt.Errorf("check agent_view defaultView exist from db failed:%s", err.Error())
		} else if !exist {
			view := &resource.AgentView{Name: defaultView, Priority: 1}
			view.SetID(defaultView)
			view.Acls = append(view.Acls, anyACL)
			key, _ := uuid.Gen()
			view.Key = base64.StdEncoding.EncodeToString([]byte(key))
			if _, err = tx.Insert(view); err != nil {
				return fmt.Errorf("Insert agent_view defaultView into db failed:%s ", err.Error())
			}
		}

		if exist, err := dbhandler.ExistWithTx(resource.TableDnsGlobalConfig, defaultGlobalConfigID, tx); err != nil {
			return fmt.Errorf("check agent_dns_global_config exist from db failed:%s", err.Error())
		} else if !exist {
			dnsGlobalConfig := &resource.AgentDnsGlobalConfig{
				LogEnable: true, Ttl: 3600, DnssecEnable: false,
			}
			dnsGlobalConfig.SetID(defaultGlobalConfigID)
			if _, err = tx.Insert(dnsGlobalConfig); err != nil {
				return fmt.Errorf("Insert defaultGlobalConfigID into db failed:%s ", err.Error())
			}
		}

		return nil
	})
}

func (handler *DNSHandler) updateRR(key string, secret string, rrset *g53.RRset, zone string, isAdd bool) error {
	serverAddr, err := net.ResolveUDPAddr("udp", handler.dnsServer)
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
		msg.UpdateRemoveRdata(rrset)
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
	_, err = conn.Write(render.Data())
	if err != nil {
		return err
	}

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

		if err := handler.rewriteNamedAclFile(tx); err != nil {
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

		if err := handler.rewriteNamedAclFile(tx); err != nil {
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

		if err := handler.rewriteNamedAclFile(tx); err != nil {
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
					return fmt.Errorf("CreateView id:%s update priority failed:%s", v.Id, err.Error())
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
				return fmt.Errorf("CreateView id:%s update priority failed:%s", view.Id, err.Error())
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

		if err := removeFiles(
			filepath.Join(handler.dnsConfPath), req.Id+"#", zoneSuffix); err != nil {
			return fmt.Errorf("DeleteView zonefile in %s err: %s",
				filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}
		if err := removeOneFile(
			filepath.Join(handler.dnsConfPath, req.Id) + nzfSuffix); err != nil {
			return fmt.Errorf("DeleteView delete nzf failed:%s", err.Error())
		}
		if err := removeOneFile(
			filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+req.Id)); err != nil {
			return fmt.Errorf("DeleteView delete rpz failed:%s", err.Error())
		}
		if err := removeOneFile(
			filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+req.Id)); err != nil {
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
	if err := formatRdata(&redirect.Rdata, &redirect.Name,
		redirect.RedirectType, redirect.DataType); err != nil {
		return fmt.Errorf("formatRdata id:%s failed:%s", req.Id, err.Error())
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(redirect); err != nil {
			return fmt.Errorf("CreateRedirection insert id:%s failed:%s",
				req.Id, err.Error())
		}

		return handler.rewriteRpzOrRedirectFile(
			req.Id, redirect.AgentView, req.RedirectType, false, tx)
	})
}

func (handler *DNSHandler) UpdateRedirection(req *pb.UpdateRedirectionReq) error {
	if err := formatRdata(&req.RData, &req.Name,
		req.RedirectType, req.DataType); err != nil {
		return fmt.Errorf("UpdateRedirection formatRdata id:%s failed:%s",
			req.Id, err.Error())
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableRedirection,
			map[string]interface{}{
				"data_type":     req.DataType,
				"rdata":         req.RData,
				"ttl":           req.Ttl,
				"name":          req.Name,
				"redirect_type": req.RedirectType,
			},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return err
		}

		return handler.rewriteRpzOrRedirectFile(
			req.Id, req.View, req.RedirectType, req.RedirectTypeChanged, tx)
	})
}

func (handler *DNSHandler) DeleteRedirection(req *pb.DeleteRedirectionReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(
			resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("DeleteRedirection insert id:%s to db failed:%s",
				req.Id, err.Error())
		}

		return handler.rewriteRpzOrRedirectFile(
			req.Id, req.View, req.RedirectType, false, tx)
	})
}

func (handler *DNSHandler) rewriteRpzOrRedirectFile(
	id, view, RedirectType string,
	RedirectTypeChanged bool, tx restdb.Transaction) error {

	if RedirectType == localZoneType {
		if err := handler.rewriteOneRPZFile(view, tx); err != nil {
			return fmt.Errorf("UpdateRedirection id:%s rewriteRPZFile failed:%s",
				id, err.Error())
		}
		if RedirectTypeChanged {
			if err := handler.rewriteOneRedirectFile(view, tx); err != nil {
				return fmt.Errorf("UpdateRedirection id:%s rewriteRedirectFile failed:%s",
					id, err.Error())
			}
		}
	} else {
		if err := handler.rewriteOneRedirectFile(view, tx); err != nil {
			return fmt.Errorf("UpdateRedirection id:%s rewriteRedirectFile failed:%s",
				id, err.Error())
		}
		if RedirectTypeChanged {
			if err := handler.rewriteOneRPZFile(view, tx); err != nil {
				return fmt.Errorf("UpdateRedirection id:%s rewriteRPZFile failed:%s",
					id, err.Error())
			}
		}
	}

	return nil
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

		if _, err := tx.Delete(
			resource.TableRR,
			map[string]interface{}{"zone": req.Id, "agent_view": req.View}); err != nil {
			return fmt.Errorf("DeleteZone delete rrs in zone id:%s from db failed:%s", req.Id, err.Error())
		}

		if err := handler.rndcDeleteZone(req.Name, req.View); err != nil {
			return fmt.Errorf("DeleteZone id:%s rndcDeleteZone view:%s failed:%s",
				req.Id, req.View, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateForwardZone(req *pb.CreateForwardZoneReq) error {
	forwardZone := &resource.AgentForwardZone{
		Name:        req.Name,
		ForwardType: req.ForwardType,
		AgentView:   req.ViewId,
		Ips:         req.ForwardIps,
	}
	forwardZone.SetID(req.Id)

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(forwardZone); err != nil {
			return fmt.Errorf("CreateForwardZone insert id:%s to db failed:%s",
				req.Id, err.Error())
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("CreateForwardZone id:%s rewriteNamedViewFile failed:%s",
				req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateForwardZone(req *pb.UpdateForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableForwardZone, map[string]interface{}{
			"forward_type": req.ForwardType,
			"ips":          req.ForwardIps,
		}, map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("UpdateForwardZone update forwardZone id:%s to db failed:%s",
				req.Id, err.Error())
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("UpdateForwardZone id:%s rewriteNamedViewFile failed:%s",
				req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteForwardZone(req *pb.DeleteForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(
			resource.TableForwardZone,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("DeleteForwardZone delete id:%s failed:%s",
				req.Id, err.Error())
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("DeleteForwardZone id:%s rewriteNamedViewFile failed:%s",
				req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateForward(req *pb.UpdateForwardReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, zone := range req.ForwardZones {
			if _, err := tx.Update(resource.TableForwardZone,
				map[string]interface{}{"ips": zone.ForwardIps},
				map[string]interface{}{restdb.IDField: zone.Id}); err != nil {
				return fmt.Errorf("update forwardZone id:%s to db failed:%s",
					zone.Id, err.Error())
			}
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("UpdateForward rewriteNamedViewFile failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) BatchCreateForwardZone(req *pb.BatchCreateForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, protoZone := range req.ForwardZones {
			forwardZone := &resource.AgentForwardZone{
				Name:        protoZone.Domain,
				ForwardType: protoZone.ForwardType,
				Ips:         protoZone.ForwardIps,
				AgentView:   protoZone.View,
			}
			forwardZone.SetID(protoZone.Id)
			if _, err := tx.Insert(forwardZone); err != nil {
				return fmt.Errorf("BatchCreateForwardZone id:%s update forwardzone id:%s failed:%s",
					protoZone.Id, protoZone.Id, err.Error())
			}
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("BatchCreateForwardZone rewriteNamedViewFile failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) BatchUpdateForwardZone(req *pb.BatchUpdateForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, protoZone := range req.ForwardZones {
			if _, err := tx.Update(resource.TableForwardZone, map[string]interface{}{
				"forward_type": protoZone.ForwardType,
				"ips":          protoZone.ForwardIps,
			}, map[string]interface{}{restdb.IDField: protoZone.Id}); err != nil {
				return fmt.Errorf("BatchUpdateForwardZone update forwardZone id:%s to db failed:%s",
					protoZone.Id, err.Error())
			}
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("BatchUpdateForwardZone rewriteNamedViewFile failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) BatchDeleteForwardZone(req *pb.BatchDeleteForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, protoZone := range req.ForwardZones {
			if _, err := tx.Delete(
				resource.TableForwardZone,
				map[string]interface{}{restdb.IDField: protoZone.Id}); err != nil {
				return fmt.Errorf("DeleteForwardZone delete id:%s failed:%s",
					protoZone.Id, err.Error())
			}
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("BatchDeleteForwardZone rewriteNamedViewFile failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) FlushForwardZone(req *pb.FlushForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, forwardView := range req.ForwardViews {
			if _, err := tx.Delete(resource.TableForwardZone,
				map[string]interface{}{"agent_view": forwardView.View}); err != nil {
				return fmt.Errorf("BatchCreateForwardZone view:%s delete forwardzones failed:%s",
					forwardView.View, err.Error())
			}

			for _, protoZone := range forwardView.ForwardZones {
				forwardZone := &resource.AgentForwardZone{
					Name:        protoZone.Domain,
					ForwardType: protoZone.ForwardType,
					Ips:         protoZone.ForwardIps,
					AgentView:   forwardView.View,
				}
				forwardZone.SetID(protoZone.Id)
				if _, err := tx.Insert(forwardZone); err != nil {
					return fmt.Errorf("BatchCreateForwardZone view:%s update forwardzone id:%s failed:%s",
						forwardView.View, protoZone.Id, err.Error())
				}
			}
		}

		if err := handler.rewriteNamedViewFile(false, tx); err != nil {
			return fmt.Errorf("BatchCreateForwardZone rewriteNamedViewFile failed:%s", err.Error())
		}
		return nil
	})
}

func formatRdata(rr, name *string, redirectType, datatype string) error {
	rrType, err := g53.TypeFromString(datatype)
	if err != nil {
		return fmt.Errorf("formatRdata datatype error:%s", err.Error())
	}

	rdata, err := g53.RdataFromString(rrType, *rr)
	if err != nil {
		return fmt.Errorf("formatRdata rdata error:%s", err.Error())
	}
	*rr = rdata.String()

	if datatype == ptrType || redirectType == localZoneType {
		return nil
	}

	if ret := strings.HasSuffix(*name, "."); !ret {
		*name += "."
	}

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
		if rdata, err = g53.RdataFromString(rrType, rr.RdataBackup); err != nil {
			return nil, fmt.Errorf("generateRRset rdata:%s error:%s",
				rr.RdataBackup, err.Error())
		}
	}

	return &g53.RRset{
		Name:   name,
		Type:   rrType,
		Class:  cls,
		Ttl:    ttl,
		Rdatas: []g53.Rdata{rdata},
	}, nil
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
		if req.ViewId == defaultView {
			rrRes, err := dbhandler.GetWithTx(req.ViewId, &[]*resource.AgentView{}, tx)
			if err != nil {
				return fmt.Errorf("CreateRR get default view id:%s from db failed:%s",
					req.ViewId, err.Error())
			}
			req.ViewKey = rrRes.(*resource.AgentView).Key
		}

		if _, err := tx.Insert(rr); err != nil {
			return fmt.Errorf("CreateRR insert id:%s to db failed:%s", req.Id, err.Error())
		}

		rrset, err := generateRRset(rr, req.ZoneName, req.ZoneRrsRole)
		if err != nil {
			return fmt.Errorf("CreateRR generateRRset failed:%s", err.Error())
		}

		if err := handler.updateRR("key"+req.ViewId, req.ViewKey, rrset, req.ZoneName, true); err != nil {
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
		if req.ViewName == defaultView {
			rrRes, err := dbhandler.GetWithTx(req.ViewName, &[]*resource.AgentView{}, tx)
			if err != nil {
				return fmt.Errorf("UpdateRRsByZone get default view id:%s from db failed:%s",
					req.ViewName, err.Error())
			}
			req.ViewKey = rrRes.(*resource.AgentView).Key
		}

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
			if rr.DataType != "A" && rr.DataType != "AAAA" {
				continue
			}

			if rr.Zone == req.ZoneId {
				rrset, err := generateRRset(rr, req.ZoneName, req.OldRrsRole)
				if err != nil {
					return fmt.Errorf("UpdateRRsByZone generateRRset failed:%s", err.Error())
				}
				if err := handler.updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, false); err != nil {
					return fmt.Errorf("UpdateRRsByZone updateRR delete rrset:%s error:%s",
						rrset.String(), err.Error())
				}

				rrset, err = generateRRset(rr, req.ZoneName, req.NewRrsRole)
				if err != nil {
					return fmt.Errorf("UpdateRRsByZone generateRRset failed:%s", err.Error())
				}
				if err := handler.updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, true); err != nil {
					return fmt.Errorf("UpdateRRsByZone updateRR add rrset:%s error:%s",
						rrset.String(), err.Error())
				}
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
			return fmt.Errorf("UpdateAllRRTtl Get views rr.id:%s falied:%s",
				rr.ID, err.Error())
		}
		zoneRes, err := dbhandler.GetWithTx(rr.Zone, &[]*resource.AgentZone{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateAllRRTtl Get zones rr.id:%s falied:%s",
				rr.ID, err.Error())
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
		if err := handler.updateRR("key"+view.Name, view.Key, rrset, zone.Name, false); err != nil {
			return fmt.Errorf("UpdateAllRRTtl delete rrset:%s error:%s",
				rrset.String(), err.Error())
		}

		rr.Ttl = uint(ttl)
		newRRset, err := generateRRset(rr, zone.Name, zone.RrsRole)
		if err != nil {
			return fmt.Errorf("UpdateAllRRTtl generateRRset failed:%s", err.Error())
		}
		if err := handler.updateRR("key"+view.Name, view.Key, newRRset, zone.Name, true); err != nil {
			return fmt.Errorf("UpdateAllRRTtl add rrset:%s error:%s",
				rrset.String(), err.Error())
		}
	}

	if err := handler.rndcDumpJNLFile(); err != nil {
		return fmt.Errorf("UpdateAllRRTtl rndcDumpJNLFile error:%s", err.Error())
	}

	return nil
}

func (handler *DNSHandler) UpdateRR(req *pb.UpdateRRReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if req.ViewName == defaultView {
			rrRes, err := dbhandler.GetWithTx(req.ViewName, &[]*resource.AgentView{}, tx)
			if err != nil {
				return fmt.Errorf("UpdateRR get default view id:%s from db failed:%s",
					req.ViewName, err.Error())
			}
			req.ViewKey = rrRes.(*resource.AgentView).Key
		}

		rrRes, err := dbhandler.GetWithTx(req.Id, &[]*resource.AgentRr{}, tx)
		if err != nil {
			return fmt.Errorf("UpdateRR get rr id:%s from db failed:%s",
				req.Id, err.Error())
		}
		rr := rrRes.(*resource.AgentRr)
		rrset, err := generateRRset(rr, req.ZoneName, req.ZoneRrsRole)
		if err != nil {
			return fmt.Errorf("UpdateRR generateRRset failed:%s", err.Error())
		}
		if err := handler.updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, false); err != nil {
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
		if err := handler.updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, true); err != nil {
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
		if req.ViewName == defaultView {
			rrRes, err := dbhandler.GetWithTx(req.ViewName, &[]*resource.AgentView{}, tx)
			if err != nil {
				return fmt.Errorf("DeleteRR get default view id:%s from db failed:%s",
					req.ViewName, err.Error())
			}
			req.ViewKey = rrRes.(*resource.AgentView).Key
		}

		if _, err := tx.Delete(resource.TableRR, map[string]interface{}{restdb.IDField: rr.ID}); err != nil {
			return fmt.Errorf("DeleteRR rr id:%s from db failed:%s", rr.ID, err.Error())
		}

		rrset, err := generateRRset(rr, req.ZoneName, req.ZoneRrsRole)
		if err != nil {
			return fmt.Errorf("DeleteRR generateRRset failed:%s", err.Error())
		}

		if err := handler.updateRR("key"+req.ViewName, req.ViewKey, rrset, req.ZoneName, false); err != nil {
			return fmt.Errorf("DeleteRR delete rrset:%s error:%s", rrset.String(), err.Error())
		}

		if err := handler.rndcZoneDumpJNLFile(req.ZoneName, req.ViewName); err != nil {
			return fmt.Errorf("DeleteRR rndcDumpJNLFile error:%s", err.Error())
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
			return fmt.Errorf("CreateIPBlackHole id:%s rewriteNamedOptionsFile failed:%s",
				req.Id, err.Error())
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
			return fmt.Errorf("UpdateIPBlackHole id:%s rewriteNamedOptionsFile failed:%s",
				req.Id, err.Error())
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
			return fmt.Errorf("DeleteIPBlackHole id:%s rewriteNamedOptionsFile failed:%s",
				req.Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateRecursiveConcurrent(req *pb.UpdateRecurConcuReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if req.IsCreate {
			recursiveConcurrent := &resource.AgentRecursiveConcurrent{
				RecursiveClients: req.RecursiveClients,
				FetchesPerZone:   req.FetchesPerZone,
			}
			recursiveConcurrent.SetID(req.Id)
			if _, err := tx.Insert(recursiveConcurrent); err != nil {
				return fmt.Errorf("insert RecursiveConcurrent from db failed:%s",
					err.Error())
			}
		} else {
			if _, err := tx.Update(resource.TableRecursiveConcurrent,
				map[string]interface{}{
					"recursive_clients": req.RecursiveClients,
					"fetches_per_zone":  req.FetchesPerZone,
				},
				map[string]interface{}{restdb.IDField: req.Id}); err != nil {
				return fmt.Errorf("update RecursiveConcurrent from db failed:%s",
					err.Error())
			}
		}

		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("UpdateRecursiveConcurrent id:%s rewriteNamedOptionsFile failed:%s",
				req.Id, err.Error())
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
			return fmt.Errorf("CreateUrlRedirect insert urlredirect domain:%s has exist",
				req.Domain)
		}

		if _, err := tx.Insert(urlRedirect); err != nil {
			return fmt.Errorf("CreateUrlRedirect insert urlredirect id:%s to db failed:%s",
				req.Id, err.Error())
		}

		if _, err := tx.Insert(redirection); err != nil {
			return fmt.Errorf("CreateUrlRedirect insert redirect id:%s to db failed:%s",
				req.Id, err.Error())
		}

		if handler.localipv6 != "" {
			redirectionIpv6 := &resource.AgentRedirection{
				Name:         urlRedirect.Domain,
				Ttl:          3600,
				DataType:     "AAAA",
				Rdata:        handler.localipv6,
				RedirectType: localZoneType,
				AgentView:    urlRedirect.AgentView,
			}
			redirectionIpv6.SetID(req.Id + "_v6")

			if _, err := tx.Insert(redirectionIpv6); err != nil {
				return fmt.Errorf("CreateUrlRedirect insert redirectionIpv6 id:%s to db failed:%s",
					req.Id, err.Error())
			}
		}

		if err := handler.rewriteOneRPZFile(urlRedirect.AgentView, tx); err != nil {
			return fmt.Errorf("CreateUrlRedirect id:%s rewriteNamedViewFile failed:%s",
				req.Id, err.Error())
		}
		if err := handler.rewriteNginxFile(tx); err != nil {
			return fmt.Errorf("CreateUrlRedirect rewrite nginxConfig for %s error:%s",
				urlRedirect.Domain, err.Error())
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
			return fmt.Errorf("update urlredirect id:%s to db failed:%s",
				req.Id, err.Error())
		}

		if _, err := tx.Update(resource.TableRedirection,
			map[string]interface{}{"name": req.Domain},
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("update redirection id:%s to db failed:%s",
				req.Id, err.Error())
		}

		if _, err := tx.Update(resource.TableRedirection,
			map[string]interface{}{"name": req.Domain},
			map[string]interface{}{restdb.IDField: req.Id + "_v6"}); err != nil {
			return fmt.Errorf("update redirectionIpv6 id:%s to db failed:%s",
				req.Id, err.Error())
		}

		if err := handler.rewriteOneRPZFile(req.View, tx); err != nil {
			return fmt.Errorf("UpdateUrlRedirect id:%s rewriteRPZFile failed:%s",
				req.Id, err.Error())
		}

		if err := handler.rewriteNginxFile(tx); err != nil {
			return fmt.Errorf("UpdateUrlRedirect rewrite nginxConfig for %s error:%s",
				req.Domain, err.Error())
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
			return fmt.Errorf("delete urlRedirect id:%s from db failed:%s",
				req.Id, err.Error())
		}

		if _, err := tx.Delete(resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id}); err != nil {
			return fmt.Errorf("delete Redirecttion id:%s from db failed:%s",
				req.Id, err.Error())
		}

		if _, err := tx.Delete(resource.TableRedirection,
			map[string]interface{}{restdb.IDField: req.Id + "_v6"}); err != nil {
			return fmt.Errorf("delete RedirecttionIpv6 id:%s from db failed:%s",
				req.Id, err.Error())
		}

		if err := handler.rewriteOneRPZFile(req.View, tx); err != nil {
			return fmt.Errorf("DeleteUrlRedirect id:%s rewriteNamedViewFile failed:%s",
				req.Id, err.Error())
		}
		if err := handler.rewriteNginxFile(tx); err != nil {
			return fmt.Errorf("DeleteUrlRedirect rewrite nginxconfig for %s error:%s",
				req.Id, err.Error())
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
				return fmt.Errorf("update updateZonesTTLSQL to db failed:%s",
					err.Error())
			}
			if _, err := tx.Exec(updateRedirectionTtlSQL, req.Ttl); err != nil {
				return fmt.Errorf("update updateRedirectionTtlSQL to db failed:%s",
					err.Error())
			}

			if err := handler.UpdateAllRRTtl(req.Ttl, tx); err != nil {
				return err
			}

			if err := handler.initRPZFile(tx); err != nil {
				return fmt.Errorf("UpdateGlobalConfig initRPZFile failed:%s", err.Error())
			}
			if err := handler.initRedirectFile(tx); err != nil {
				return fmt.Errorf("UpdateGlobalConfig initRedirectFile failed:%s", err.Error())
			}
			if err := handler.rndcReconfig(); err != nil {
				return fmt.Errorf("UpdateGlobalConfig rndcReconfig failed:%s", err.Error())
			}
		}

		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("UpdateGlobalConfig rewriteNamedOptionsFile failed:%s",
				err.Error())
		}

		return nil
	})
}
