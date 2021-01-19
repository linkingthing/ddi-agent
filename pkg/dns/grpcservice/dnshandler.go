package grpcservice

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
	"time"

	monitorpb "github.com/linkingthing/ddi-monitor/pkg/proto"
	"github.com/zdnscloud/cement/uuid"
	"github.com/zdnscloud/g53"
	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/ddi-agent/config"
	"github.com/linkingthing/ddi-agent/pkg/db"
	"github.com/linkingthing/ddi-agent/pkg/dns/dbhandler"
	"github.com/linkingthing/ddi-agent/pkg/dns/resource"
	"github.com/linkingthing/ddi-agent/pkg/grpcclient"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

const (
	mainConfName          = "named.conf"
	namedTpl              = "named.tpl"
	namedViewTpl          = "named_view.tpl"
	namedViewConfName     = "named_view.conf"
	namedAclTpl           = "named_acl.tpl"
	namedAclConfName      = "named_acl.conf"
	namedOptionsTpl       = "named_options.tpl"
	namedOptionsConfName  = "named_options.conf"
	nginxDefaultTpl       = "nginxdefault.tpl"
	nginxSslTpl           = "nginxssl.tpl"
	redirectTpl           = "redirect.tpl"
	rpzTpl                = "rpz.tpl"
	zoneTpl               = "zone.tpl"
	zoneSuffix            = ".zone"
	nzfTpl                = "nzf.tpl"
	nzfSuffix             = ".nzf"
	checkPeriod           = 5
	anyACL                = "any"
	noneACL               = "none"
	DefaultView           = "default"
	nginxDefaultConfFile  = "ddi_domains.conf"
	defaultGlobalConfigID = "globalConfig"
	TemplateDir           = "/etc/dns/templates"
	FilePermissions       = 0777
)

type DNSHandler struct {
	tpl                 *template.Template
	dnsConfPath         string
	tplPath             string
	ticker              *time.Ticker
	quit                chan int
	nginxDefaultConfDir string
	nginxKeyDir         string
	localip             string
	interfaceIPs        []string
	localipv6           string
	dnsServerIP         string
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
		nginxKeyDir:         conf.NginxDefaultDir + "/key",
		localip:             conf.Server.IP,
		localipv6:           conf.Server.IPV6,
		dnsServerIP:         conf.DNS.ServerIp,
	}
	instance.interfaceIPs, _ = getInterfaceIPs()
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
	if _, err := grpcclient.GetDDIMonitorGrpcClient().StopDNS(context.Background(),
		&monitorpb.StopDNSRequest{}); err != nil {
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

		if exist, err := dbhandler.ExistWithTx(resource.TableView, DefaultView, tx); err != nil {
			return fmt.Errorf("check agent_view DefaultView exist from db failed:%s", err.Error())
		} else if !exist {
			view := &resource.AgentView{Name: DefaultView, Priority: 1, Recursion: true}
			view.SetID(DefaultView)
			view.Acls = append(view.Acls, anyACL)
			key, _ := uuid.Gen()
			view.Key = base64.StdEncoding.EncodeToString([]byte(key))
			if _, err = tx.Insert(view); err != nil {
				return fmt.Errorf("Insert agent_view DefaultView into db failed:%s ", err.Error())
			}
		}

		if exist, err := dbhandler.ExistWithTx(resource.TableDnsGlobalConfig, defaultGlobalConfigID, tx); err != nil {
			return fmt.Errorf("check agent_dns_global_config exist from db failed:%s", err.Error())
		} else if !exist {
			dnsGlobalConfig := resource.CreateDefaultResource()
			dnsGlobalConfig.SetID(defaultGlobalConfigID)
			if _, err = tx.Insert(dnsGlobalConfig); err != nil {
				return fmt.Errorf("Insert defaultGlobalConfigID into db failed:%s ", err.Error())
			}
		}

		return nil
	})
}

func (handler *DNSHandler) CreateACL(req *pb.CreateAclReq) error {
	acl := &resource.AgentAcl{Name: req.GetAcl().GetName(), Ips: req.GetAcl().GetIps()}
	acl.SetID(req.GetAcl().GetId())

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(acl); err != nil {
			return fmt.Errorf("CreateACL insert acl db id:%s failed: %s ", req.GetAcl().Id, err.Error())
		}

		if err := handler.rewriteNamedAclFile(tx); err != nil {
			return fmt.Errorf("CreateACL id:%s rewriteNamedFile failed :%s", req.GetAcl().Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) BatchCreateACL(req *pb.BatchCreateAclReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, acl := range req.Acls {
			acl := &resource.AgentAcl{Name: acl.Name, Ips: acl.Ips}
			acl.SetID(acl.ID)
			if _, err := tx.Insert(acl); err != nil {
				return fmt.Errorf("BatchCreateACL insert acl db id:%s failed: %s ", acl.ID, err.Error())
			}
		}

		if err := handler.rewriteNamedAclFile(tx); err != nil {
			return fmt.Errorf("BatchCreateACL rewriteNamedFile failed :%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateACL(req *pb.UpdateAclReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(
			resource.TableAcl,
			map[string]interface{}{"ips": req.GetAcl().Ips},
			map[string]interface{}{restdb.IDField: req.GetAcl().Id}); err != nil {
			return fmt.Errorf("UpdateACL failed:%s", err.Error())
		}

		if err := handler.rewriteNamedAclFile(tx); err != nil {
			return fmt.Errorf("UpdateACL id:%s rewriteNamedFile failed :%s", req.GetAcl().Id, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteACL(req *pb.DeleteAclReq) error {
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
		Name:      req.Name,
		Priority:  uint(req.Priority),
		Acls:      req.Acls,
		Dns64:     req.Dns64,
		Key:       req.Key,
		Recursion: req.Recursion,
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

		if err := handler.rewriteNamedViewFile(tx, false); err != nil {
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
				"priority":  req.Priority,
				"acls":      req.Acls,
				"dns64":     req.Dns64,
				"recursion": req.Recursion,
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

		if err := handler.rewriteNamedViewFile(tx, false); err != nil {
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
			filepath.Join(handler.dnsConfPath), req.Id+"#", ""); err != nil {
			return fmt.Errorf("DeleteView zonefile in %s err: %s",
				filepath.Join(handler.dnsConfPath, "redirection"), err.Error())
		}
		if err := removeFile(
			filepath.Join(handler.dnsConfPath, req.Id) + nzfSuffix); err != nil {
			return fmt.Errorf("DeleteView delete nzf failed:%s", err.Error())
		}
		if err := removeFile(
			filepath.Join(handler.dnsConfPath, "redirection", "rpz_"+req.Id)); err != nil {
			return fmt.Errorf("DeleteView delete rpz failed:%s", err.Error())
		}
		if err := removeFile(
			filepath.Join(handler.dnsConfPath, "redirection", "redirect_"+req.Id)); err != nil {
			return fmt.Errorf("DeleteView delete redirect failed:%s", err.Error())
		}
		if err := handler.rewriteNamedViewFile(tx, false); err != nil {
			return fmt.Errorf("DeleteView rewriteNamedViewFile  failed:%s", err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) CreateAuthZone(req *pb.CreateAuthZoneReq) error {
	zone := &resource.AgentAuthZone{
		Name:      req.GetAuthZone().Name,
		Ttl:       req.GetAuthZone().Ttl,
		AgentView: req.GetAuthZone().View,
		Role:      resource.AuthZoneRole(req.GetAuthZone().Role),
		Masters:   req.GetAuthZone().Masters,
		Slaves:    req.GetAuthZone().Slaves}
	if err := zone.Validate(); err != nil {
		return fmt.Errorf("auth zone name %s is invalid %s", zone.Name, err.Error())
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(zone); err != nil {
			return fmt.Errorf("create auth zone %s with view %s failed:%s", zone.Name, zone.AgentView, err.Error())
		}

		if err := handler.rewriteAuthZoneFile(tx, zone); err != nil {
			return fmt.Errorf("create auth zone %s with view %s file failed:%s",
				zone.Name, zone.AgentView, err.Error())
		}

		if err := handler.rndcAddZone(zone); err != nil {
			return fmt.Errorf("add auth zone %s with view %s to dns failed:%s", zone.Name, zone.AgentView, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateAuthZone(req *pb.UpdateAuthZoneReq) error {
	zone := &resource.AgentAuthZone{
		Name:      req.GetAuthZone().Name,
		Ttl:       req.GetAuthZone().Ttl,
		AgentView: req.GetAuthZone().View,
		Role:      resource.AuthZoneRole(req.GetAuthZone().Role),
		Masters:   req.GetAuthZone().Masters,
		Slaves:    req.GetAuthZone().Slaves}

	if err := zone.Validate(); err != nil {
		return fmt.Errorf("auth zone name %s is invalid %s", zone.Name, err.Error())
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableAgentAuthZone,
			map[string]interface{}{
				"ttl": req.GetAuthZone().Ttl, "role": req.GetAuthZone().Role,
				"masters": req.GetAuthZone().Masters, "slaves": req.GetAuthZone().Slaves},
			map[string]interface{}{"agent_view": zone.AgentView, "name": zone.Name},
		); err != nil {
			return fmt.Errorf("update auth zone %s with view %s failed:%s", zone.Name, zone.AgentView, err.Error())
		}

		if err := handler.rewriteAuthZoneFile(tx, zone); err != nil {
			return fmt.Errorf("rewrite auth zone %s with view %s file failed:%s",
				zone.Name, zone.AgentView, err.Error())
		}

		if err := handler.rndcModifyZone(zone); err != nil {
			return fmt.Errorf("update auth zone %s with view %s to dns failed:%s",
				zone.Name, zone.AgentView, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteAuthZone(req *pb.DeleteAuthZoneReq) error {
	zone := &resource.AgentAuthZone{Name: req.Name, AgentView: req.View}
	if err := zone.Validate(); err != nil {
		return fmt.Errorf("auth zone name %s is invalid %s", zone.Name, err.Error())
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableAgentAuthZone, map[string]interface{}{
			"agent_view": zone.AgentView, "name": zone.Name,
		}); err != nil {
			return fmt.Errorf("delete zone %s with view %s from db failed:%s",
				zone.Name, zone.AgentView, err.Error())
		}

		return handler.deleteZoneAuthRR(tx, zone)
	})
}

func (handler *DNSHandler) deleteZoneAuthRR(tx restdb.Transaction, zone *resource.AgentAuthZone) error {
	if _, err := tx.Delete(resource.TableAgentAuthRR, map[string]interface{}{
		"zone": zone.Name, "agent_view": zone.AgentView}); err != nil {
		return fmt.Errorf("delete zone %s with view %s rrs from db failed:%s",
			zone.Name, zone.AgentView, err.Error())
	}

	if err := handler.rndcDeleteZone(zone.Name, zone.AgentView); err != nil {
		return fmt.Errorf("delete zone %s with view %s rrs from dns failed:%s",
			zone.Name, zone.AgentView, err.Error())
	}

	return nil
}

func (handler *DNSHandler) CreateAuthZoneAuthRRs(req *pb.CreateAuthZoneAuthRRsReq) error {
	zone := &resource.AgentAuthZone{
		Name:      req.GetAuthZone().Name,
		Ttl:       req.GetAuthZone().Ttl,
		AgentView: req.GetAuthZone().View,
		Role:      resource.AuthZoneRole(req.GetAuthZone().Role),
		Masters:   req.GetAuthZone().Masters,
		Slaves:    req.GetAuthZone().Slaves}
	if err := zone.Validate(); err != nil {
		return fmt.Errorf("auth zone name %s is invalid %s", zone.Name, err.Error())
	}

	sql, err := genBatchInsertAuthRRsSql(req.AuthZoneRrs)
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(zone); err != nil {
			return fmt.Errorf(
				"create auth zone %s with view %s failed:%s", zone.Name, zone.AgentView, err.Error())
		}

		if sql != "" {
			if _, err := tx.Exec(sql); err != nil {
				return fmt.Errorf(
					"create auth rrs with zone %s with view %s failed:%s", zone.Name, zone.AgentView, err.Error())
			}
		}

		if err := handler.rewriteAuthZoneFile(tx, zone); err != nil {
			return fmt.Errorf("create auth zone %s with view %s file failed:%s",
				zone.Name, zone.AgentView, err.Error())
		}

		if err := handler.rndcAddZone(zone); err != nil {
			return fmt.Errorf(
				"add auth zone %s with view %s to dns failed:%s", zone.Name, zone.AgentView, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateAuthZoneAXFR(req *pb.UpdateAuthZoneAXFRReq) error {
	if len(req.AuthZoneRrs) == 0 {
		return nil
	}

	sql, err := genBatchInsertAuthRRsSql(req.AuthZoneRrs)
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var zones []*resource.AgentAuthZone
		for _, authZone := range req.AuthZones {
			zone := &resource.AgentAuthZone{
				Name:      authZone.Name,
				Ttl:       authZone.Ttl,
				AgentView: authZone.View,
				Role:      resource.AuthZoneRole(authZone.Role),
				Masters:   authZone.Masters,
				Slaves:    authZone.Slaves}
			if err := zone.Validate(); err != nil {
				return fmt.Errorf("auth zone name %s is invalid %s", zone.Name, err.Error())
			}
			if err := handler.deleteZoneAuthRR(tx, zone); err != nil {
				return err
			}
			zones = append(zones, zone)
		}

		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("create auth rrs failed:%s", err.Error())
		}

		for _, zone := range zones {
			if err := handler.rewriteAuthZoneFile(tx, zone); err != nil {
				return fmt.Errorf("create auth zone %s with view %s file failed:%s",
					zone.Name, zone.AgentView, err.Error())
			}

			if err := handler.rndcAddZone(zone); err != nil {
				return fmt.Errorf(
					"add auth zone %s with view %s to dns failed:%s", zone.Name, zone.AgentView, err.Error())
			}
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateAuthZoneIXFR(req *pb.UpdateAuthZoneIXFRReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		for _, oldAuthZoneRr := range req.OldAuthZoneRrs {
			if err := handler.deleteAuthRRFromDB(tx, oldAuthZoneRr); err != nil {
				return err
			}
		}

		for _, newAuthZoneRr := range req.NewAuthZoneRrs {
			if err := handler.addAuthRRToDB(tx, newAuthZoneRr); err != nil {
				return err
			}
		}

		return nil
	})
}

func (handler *DNSHandler) CreateForwardZone(req *pb.CreateForwardZoneReq) error {
	forwardZone := &resource.AgentForwardZone{
		Name:         req.Name,
		ForwardStyle: req.ForwardStyle,
		AgentView:    req.View,
		Addresses:    req.Addresses,
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(forwardZone); err != nil {
			return fmt.Errorf("insert forward zone %s with view %s to db failed:%s",
				forwardZone.Name, forwardZone.AgentView, err.Error())
		}

		if err := handler.rewriteNamedViewFile(tx, false); err != nil {
			return fmt.Errorf("add forward zone %s with view %s to dns failed:%s",
				forwardZone.Name, forwardZone.AgentView, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateForwardZone(req *pb.UpdateForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableAgentForwardZone, map[string]interface{}{
			"forward_style": req.ForwardStyle, "addresses": req.Addresses,
		}, map[string]interface{}{
			"agent_view": req.View, "zone": req.Name,
		}); err != nil {
			return fmt.Errorf("update forward zone %s with view %s to db failed:%s",
				req.Name, req.View, err.Error())
		}

		if err := handler.rewriteNamedViewFile(tx, false); err != nil {
			return fmt.Errorf("update forward zone %s with view %s to dns failed:%s",
				req.Name, req.View, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteForwardZone(req *pb.DeleteForwardZoneReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableAgentForwardZone, map[string]interface{}{
			"agent_view": req.View, "zone": req.Name,
		}); err != nil {
			return fmt.Errorf("delete forward zone %s with view %s from db failed:%s",
				req.Name, req.View, err.Error())
		}

		if err := handler.rewriteNamedViewFile(tx, false); err != nil {
			return fmt.Errorf("delete forward zone %s with view %s from dns failed:%s",
				req.Name, req.View, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) FlushForwardZone(req *pb.FlushForwardZoneReq) error {
	oldSql := getDeleteForwardZonesSql(req.OldForwardZones)
	newSql := getAddForwardZonesSql(req.NewForwardZones)
	if oldSql == "" && newSql == "" {
		return nil
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if oldSql != "" {
			if _, err := tx.Exec(oldSql); err != nil {
				return fmt.Errorf("delete forward zones from db failed:%s", err.Error())
			}
		}
		if newSql != "" {
			if _, err := tx.Exec(newSql); err != nil {
				return fmt.Errorf("batch create forward zones to db failed:%s", err.Error())
			}
		}

		if err := handler.rewriteNamedViewFile(tx, false); err != nil {
			return fmt.Errorf("reflush forward zones from dns failed:%s", err.Error())
		}

		return nil
	})
}

func getDeleteForwardZonesSql(forwardZones []*pb.FlushForwardZoneReqForwardZone) string {
	if len(forwardZones) == 0 {
		return ""
	}

	oldViews := make(map[string]struct{})
	oldZones := make(map[string]struct{})
	for _, forwardZone := range forwardZones {
		oldViews[forwardZone.View] = struct{}{}
		oldZones[forwardZone.Name] = struct{}{}
	}

	var views, zones []string
	for view := range oldViews {
		views = append(views, view)
	}

	for zone := range oldZones {
		zones = append(zones, zone)
	}

	return "delete from gr_agent_forward_zone where agent_view in ('" +
		strings.Join(views, "','") + "') and zone in ('" + strings.Join(zones, "','") + "');"
}

func getAddForwardZonesSql(forwardZones []*pb.FlushForwardZoneReqForwardZone) string {
	if len(forwardZones) == 0 {
		return ""
	}

	var buf bytes.Buffer
	buf.WriteString("insert into gr_agent_forward_zone (id, create_time, name, forward_style, ips, agent_view) values")
	for _, zone := range forwardZones {
		buf.WriteString("('")
		id, _ := uuid.Gen()
		buf.WriteString(id)
		buf.WriteString("','")
		buf.WriteString(time.Now().Format(time.RFC3339))
		buf.WriteString("','")
		buf.WriteString(zone.Name)
		buf.WriteString("','")
		buf.WriteString(zone.ForwardStyle)
		buf.WriteString("','{")
		buf.WriteString(strings.Join(zone.GetForwardIps(), ","))
		buf.WriteString("}','")
		buf.WriteString(zone.View)
		buf.WriteString("')")
		buf.WriteString(",")
	}

	return strings.TrimSuffix(buf.String(), ",") + ";"
}

func (handler *DNSHandler) CreateAuthRR(req *pb.CreateAuthRRReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return handler.addAuthRRToDB(tx, req.Rr)
	})
}

func (handler *DNSHandler) addAuthRRToDB(tx restdb.Transaction, authZoneRr *pb.AuthZoneRR) error {
	rr, rrSet, err := pbAuthRRToAgentAuthRRAndRRset(authZoneRr)
	if err != nil {
		return err
	}
	if rr.AgentView == DefaultView {
		rrRes, err := dbhandler.GetWithTx(DefaultView, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("create auth rr get default view from db failed:%s", err.Error())
		}
		authZoneRr.ViewKey = rrRes.(*resource.AgentView).Key
	}
	if _, err := tx.Insert(rr); err != nil {
		return fmt.Errorf("insert auth rr %s with zone %s and view %s to db failed:%s",
			rr.Name, rr.Zone, rr.AgentView, err.Error())
	}
	if err := handler.updateRR("key"+rr.AgentView, authZoneRr.ViewKey, rrSet, rr.Zone, true); err != nil {
		return fmt.Errorf("update auth rr %s to dns failed:%s", rrSet.String(), err.Error())
	}
	if err := handler.rndcZoneDumpJNLFile(rr.Zone, rr.AgentView); err != nil {
		return fmt.Errorf("CreateRR rndcDumpJNLFile error:%s", err.Error())
	}

	return nil
}

func pbAuthRRToAgentAuthRRAndRRset(pbRR *pb.AuthZoneRR) (*resource.AgentAuthRr, *g53.RRset, error) {
	rr := &resource.AgentAuthRr{
		Name:      pbRR.Name,
		RrType:    pbRR.Type,
		Ttl:       pbRR.Ttl,
		Rdata:     pbRR.Rdata,
		AgentView: pbRR.View,
		Zone:      pbRR.Zone,
	}

	rrset, err := rr.ToRRset()
	if err != nil {
		return nil, nil, fmt.Errorf("rr %s with zone %s and view %s is invalid: %s",
			rr.Name, rr.Zone, rr.AgentView, err.Error())
	}

	return rr, rrset, nil
}

func (handler *DNSHandler) updateRR(key string, secret string, rrset *g53.RRset, zone string, isAdd bool) error {
	serverAddr, err := net.ResolveUDPAddr("udp", handler.dnsServerIP+":53")
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

func (handler *DNSHandler) UpdateAuthRR(req *pb.UpdateAuthRRReq) error {
	oldRR, oldRRset, err := pbAuthRRToAgentAuthRRAndRRset(req.OldRr)
	if err != nil {
		return err
	}

	newRR, newRRset, err := pbAuthRRToAgentAuthRRAndRRset(req.NewRr)
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if req.NewRr.View == DefaultView {
			rrRes, err := dbhandler.GetWithTx(DefaultView, &[]*resource.AgentView{}, tx)
			if err != nil {
				return fmt.Errorf("update auth rr get default view from db failed:%s", err.Error())
			}
			req.NewRr.ViewKey = rrRes.(*resource.AgentView).Key
		}

		if err := handler.updateRR("key"+oldRR.AgentView, req.NewRr.ViewKey, oldRRset, oldRR.Zone, false); err != nil {
			return fmt.Errorf("delete old rrset %s failed: %s", oldRRset.String(), err.Error())
		}

		if _, err := tx.Update(resource.TableAgentAuthRR, map[string]interface{}{
			"ttl":   newRR.Ttl,
			"rdata": newRR.Rdata}, map[string]interface{}{
			"agent_view": oldRR.AgentView, "zone": oldRR.Zone,
			"name": oldRR.Name, "rr_type": oldRR.RrType, "rdata": oldRR.Rdata,
		}); err != nil {
			return err
		}

		if err := handler.updateRR("key"+newRR.AgentView, req.NewRr.ViewKey, newRRset, newRR.Zone, true); err != nil {
			return fmt.Errorf("update auth rrset %s failed: %s", newRRset.String(), err.Error())
		}

		if err := handler.rndcZoneDumpJNLFile(newRR.Zone, newRR.AgentView); err != nil {
			return fmt.Errorf("update auth rrset %s to dns failed:%s", newRRset.String(), err.Error())
		}

		return nil
	})
}

func (handler *DNSHandler) DeleteAuthRR(req *pb.DeleteAuthRRReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		return handler.deleteAuthRRFromDB(tx, req.Rr)
	})
}

func (handler *DNSHandler) deleteAuthRRFromDB(tx restdb.Transaction, authZoneRr *pb.AuthZoneRR) error {
	rr, rrSet, err := pbAuthRRToAgentAuthRRAndRRset(authZoneRr)
	if err != nil {
		return err
	}
	if rr.AgentView == DefaultView {
		rrRes, err := dbhandler.GetWithTx(DefaultView, &[]*resource.AgentView{}, tx)
		if err != nil {
			return fmt.Errorf("delete auth rr get default view from db failed:%s", err.Error())
		}
		authZoneRr.ViewKey = rrRes.(*resource.AgentView).Key
	}
	if _, err := tx.Delete(resource.TableAgentAuthRR, map[string]interface{}{
		"agent_view": rr.AgentView, "zone": rr.Zone,
		"name": rr.Name, "rr_type": rr.RrType, "rdata": rr.Rdata,
	}); err != nil {
		return fmt.Errorf("delete auth rr %s from db failed:%s", rrSet.String(), err.Error())
	}
	if err := handler.updateRR("key"+rr.AgentView, authZoneRr.ViewKey, rrSet, rr.Zone, false); err != nil {
		return fmt.Errorf("delete auth rrset %s failed: %s", rrSet.String(), err.Error())
	}
	if err := handler.rndcZoneDumpJNLFile(rr.Zone, rr.AgentView); err != nil {
		return fmt.Errorf("delete rrset %s from dns failed: %s", rrSet.String(), err.Error())
	}

	return nil
}

func (handler *DNSHandler) BatchCreateAuthRRs(req *pb.BatchCreateAuthRRsReq) error {
	if len(req.AuthZoneRrs) == 0 {
		return nil
	}

	reqView := req.AuthZoneRrs[0].View
	reqZone := req.AuthZoneRrs[0].Zone
	sql, err := genBatchInsertAuthRRsSql(req.AuthZoneRrs)
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var zones []*resource.AgentAuthZone
		if err := tx.Fill(map[string]interface{}{"agent_view": reqView, "name": reqZone}, &zones); err != nil {
			return fmt.Errorf("found zone %s with view %s failed: %s", reqZone, reqView, err.Error())
		} else if len(zones) != 1 {
			return fmt.Errorf("no found zone %s with view %s", reqZone, reqView)
		}

		if _, err := tx.Exec(sql); err != nil {
			return fmt.Errorf("batch create zone %s rrs with view %s failed: %s", reqZone, reqView, err.Error())
		}

		if err := handler.rewriteAuthZoneFile(tx, zones[0]); err != nil {
			return fmt.Errorf("rewrite zone %s files with view %s failed: %s", reqZone, reqView, err.Error())
		}

		if err := handler.rndcModifyZone(zones[0]); err != nil {
			return fmt.Errorf("reconfig zone %s with view %s failed: %s", reqZone, reqView, err.Error())
		}

		return nil
	})
}

func genBatchInsertAuthRRsSql(authZoneRrs []*pb.AuthZoneRR) (string, error) {
	if len(authZoneRrs) == 0 {
		return "", nil
	}

	var buf bytes.Buffer
	buf.WriteString("insert into gr_agent_auth_rr (id, create_time, name, rr_type, ttl, rdata, zone, agent_view) values ")
	for _, authZoneRr := range authZoneRrs {
		rr, _, err := pbAuthRRToAgentAuthRRAndRRset(authZoneRr)
		if err != nil {
			return "", err
		}

		buf.WriteString("('")
		id, _ := uuid.Gen()
		buf.WriteString(id)
		buf.WriteString("','")
		buf.WriteString(time.Now().Format(time.RFC3339))
		buf.WriteString("','")
		buf.WriteString(rr.Name)
		buf.WriteString("','")
		buf.WriteString(rr.RrType)
		buf.WriteString("','")
		buf.WriteString(strconv.Itoa(int(rr.Ttl)))
		buf.WriteString("','")
		buf.WriteString(rr.Rdata)
		buf.WriteString("','")
		buf.WriteString(rr.Zone)
		buf.WriteString("','")
		buf.WriteString(rr.AgentView)
		buf.WriteString("')")
		buf.WriteString(",")
	}

	return strings.TrimSuffix(buf.String(), ",") + ";", nil
}

func (handler *DNSHandler) CreateRedirection(req *pb.CreateRedirectionReq) error {
	redirect, err := pbRedirectionToAgentRedirection(req.Redirection)
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Insert(redirect); err != nil {
			return fmt.Errorf("insert redirection %s with view %s to db failed: %s",
				redirect.Name, redirect.AgentView, err.Error())
		}

		return handler.rewriteRpzOrRedirectFile(tx, redirect, false)
	})
}

func pbRedirectionToAgentRedirection(redirection *pb.Redirection) (*resource.AgentRedirection, error) {
	redirect := &resource.AgentRedirection{
		Name:         redirection.Name,
		Ttl:          redirection.Ttl,
		RrType:       redirection.RrType,
		RedirectType: redirection.RedirectType,
		Rdata:        redirection.Rdata,
		AgentView:    redirection.View,
	}

	if err := redirect.Validate(); err != nil {
		return nil, fmt.Errorf("redirection %s with view %s invalid: %s", redirect.Name, redirect.AgentView, err.Error())
	}

	return redirect, nil
}

func (handler *DNSHandler) rewriteRpzOrRedirectFile(tx restdb.Transaction, redirect *resource.AgentRedirection, redirectTypeChanged bool) error {
	if redirect.RedirectType == resource.LocalZoneType {
		if err := handler.rewriteOneRPZFile(redirect.AgentView, tx); err != nil {
			return fmt.Errorf("update redirection %s with view %s rewrite rpz file failed:%s",
				redirect.Name, redirect.AgentView, err.Error())
		}
		if redirectTypeChanged {
			if err := handler.rewriteOneRedirectFile(redirect.AgentView, tx); err != nil {
				return fmt.Errorf("update redirection %s with view %s rewrite redirect file failed:%s",
					redirect.Name, redirect.AgentView, err.Error())
			}
		}
	} else {
		if err := handler.rewriteOneRedirectFile(redirect.AgentView, tx); err != nil {
			return fmt.Errorf("update redirection %s with view %s rewrite redirect file failed:%s",
				redirect.Name, redirect.AgentView, err.Error())
		}
		if redirectTypeChanged {
			if err := handler.rewriteOneRPZFile(redirect.AgentView, tx); err != nil {
				return fmt.Errorf("update redirection %s with view %s rewrite rpz file failed:%s",
					redirect.Name, redirect.AgentView, err.Error())
			}
		}
	}

	return nil
}

func (handler *DNSHandler) UpdateRedirection(req *pb.UpdateRedirectionReq) error {
	oldRedirect, err := pbRedirectionToAgentRedirection(req.OldRedirection)
	if err != nil {
		return err
	}

	newRedirect, err := pbRedirectionToAgentRedirection(req.NewRedirection)
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableAgentRedirection, map[string]interface{}{
			"rdata":         newRedirect.Rdata,
			"ttl":           newRedirect.Ttl,
			"redirect_type": newRedirect.RedirectType,
		}, map[string]interface{}{
			"agent_view": oldRedirect.AgentView,
			"name":       oldRedirect.Name,
			"rr_type":    oldRedirect.RrType,
			"rdata":      oldRedirect.Rdata,
		}); err != nil {
			return err
		}

		return handler.rewriteRpzOrRedirectFile(tx, newRedirect, oldRedirect.RedirectType != newRedirect.RedirectType)
	})
}

func (handler *DNSHandler) DeleteRedirection(req *pb.DeleteRedirectionReq) error {
	oldRedirect, err := pbRedirectionToAgentRedirection(req.Redirection)
	if err != nil {
		return err
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableAgentRedirection, map[string]interface{}{
			"agent_view": oldRedirect.AgentView,
			"name":       oldRedirect.Name,
			"rr_type":    oldRedirect.RrType,
			"rdata":      oldRedirect.Rdata,
		}); err != nil {
			return fmt.Errorf("delete redirection %s with view %s failed:%s",
				oldRedirect.Name, oldRedirect.AgentView, err.Error())
		}

		return handler.rewriteRpzOrRedirectFile(tx, oldRedirect, false)
	})
}

func (handler *DNSHandler) CreateUrlRedirect(req *pb.CreateUrlRedirectReq) error {
	urlRedirect := &resource.AgentUrlRedirect{
		Domain:    req.Domain,
		Url:       req.Url,
		AgentView: req.View,
		IsHttps:   req.IsHttps,
	}

	redirection := &resource.AgentRedirection{
		Name:         urlRedirect.Domain,
		Ttl:          3600,
		RrType:       "A",
		Rdata:        handler.localip,
		RedirectType: resource.LocalZoneType,
		AgentView:    urlRedirect.AgentView,
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		exist, _ := tx.Exists(resource.TableAgentRedirection, map[string]interface{}{"name": urlRedirect.Domain})
		if exist {
			return fmt.Errorf("create urlredirect domain %s is duplicate", urlRedirect.Domain)
		}

		if _, err := tx.Insert(urlRedirect); err != nil {
			return fmt.Errorf("insert urlredirect %s to db failed:%s", urlRedirect.Domain, err.Error())
		}

		if _, err := tx.Insert(redirection); err != nil {
			return fmt.Errorf("create urlredirect insert redirect %s to db failed:%s", redirection.Name, err.Error())
		}

		if handler.localipv6 != "" {
			redirectionIpv6 := &resource.AgentRedirection{
				Name:         urlRedirect.Domain,
				Ttl:          3600,
				RrType:       "AAAA",
				Rdata:        handler.localipv6,
				RedirectType: resource.LocalZoneType,
				AgentView:    urlRedirect.AgentView,
			}

			if _, err := tx.Insert(redirectionIpv6); err != nil {
				return fmt.Errorf("create urlredirect insert redirection ipv6 %s to db failed:%s",
					redirectionIpv6.Name, err.Error())
			}
		}

		if err := handler.rewriteOneRPZFile(urlRedirect.AgentView, tx); err != nil {
			return fmt.Errorf("create urlredirect %s and rewrite rpz file failed:%s", urlRedirect.Domain, err.Error())
		}

		if urlRedirect.IsHttps {
			if err := handler.addNginxHttpsFile(req.Key, req.Crt, urlRedirect); err != nil {
				return fmt.Errorf("create urlredirect %s and add nginx https file failed: %s",
					urlRedirect.Domain, err.Error())
			}
		} else {
			if err := handler.rewriteNginxHttpFile(tx); err != nil {
				return fmt.Errorf("create urlredirect %s and rewrite nginx config failed:%s",
					urlRedirect.Domain, err.Error())
			}
		}

		if err := handler.nginxReload(); err != nil {
			return fmt.Errorf("create urlredirect %s and nginx reload failed: %s",
				urlRedirect.Domain, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateUrlRedirect(req *pb.UpdateUrlRedirectReq) error {
	urlRedirect := &resource.AgentUrlRedirect{
		Domain:    req.Domain,
		Url:       req.Url,
		AgentView: req.View,
		IsHttps:   req.IsHttps,
	}

	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Update(resource.TableAgentUrlRedirect, map[string]interface{}{
			"url": urlRedirect.Url}, map[string]interface{}{
			"domain": urlRedirect.Domain, "agent_view": urlRedirect.AgentView,
		}); err != nil {
			return fmt.Errorf("update urlredirect %s to db failed: %s",
				urlRedirect.Domain, err.Error())
		}

		if err := handler.rewriteOneRPZFile(urlRedirect.AgentView, tx); err != nil {
			return fmt.Errorf("update urlredirect %s and rewrite rpz file failed:%s",
				urlRedirect.Domain, err.Error())
		}

		if urlRedirect.IsHttps {
			if err := handler.updateNginxHttpsFile(urlRedirect); err != nil {
				return fmt.Errorf("update urlredirect %s and update nginx https file failed:%s",
					urlRedirect.Domain, err.Error())
			}
		} else {
			if err := handler.rewriteNginxHttpFile(tx); err != nil {
				return fmt.Errorf("update urlredirect %s and update nginx config failed:%s",
					urlRedirect.Domain, err.Error())
			}
		}

		if err := handler.nginxReload(); err != nil {
			return fmt.Errorf("update urlredirect %s and nginx reload failed:%s",
				urlRedirect.Domain, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) DeleteUrlRedirect(req *pb.DeleteUrlRedirectReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if _, err := tx.Delete(resource.TableAgentUrlRedirect,
			map[string]interface{}{restdb.IDField: req.Domain}); err != nil {
			return fmt.Errorf("delete urlredirect %s from db failed:%s",
				req.Domain, err.Error())
		}

		if _, err := tx.Delete(resource.TableAgentRedirection,
			map[string]interface{}{"name": req.Domain, "agent_view": req.View}); err != nil {
			return fmt.Errorf("delete redirect %s from db failed:%s",
				req.Domain, err.Error())
		}

		if err := handler.rewriteOneRPZFile(req.View, tx); err != nil {
			return fmt.Errorf("delete redirect %s and rewrite rpz file failed:%s",
				req.Domain, err.Error())
		}

		if req.IsHttps {
			if err := handler.removeNginxHttpsFile(req.Domain); err != nil {
				return fmt.Errorf("delete redirect %s and remove https file failed:%s",
					req.Domain, err.Error())
			}
		} else {
			if err := handler.rewriteNginxHttpFile(tx); err != nil {
				return fmt.Errorf("delete redirect %s and rewrite nginx config failed:%s",
					req.Domain, err.Error())
			}
		}

		if err := handler.nginxReload(); err != nil {
			return fmt.Errorf("delete redirect %s and reload nginx failed:%s",
				req.Domain, err.Error())
		}
		return nil
	})
}

func (handler *DNSHandler) UpdateGlobalConfig(req *pb.UpdateGlobalConfigReq) error {
	return restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		update := make(map[string]interface{})
		updateTtl := false
		switch req.UpdateModel {
		case resource.DnsConfigUpdateModelLog:
			update["log_enable"] = req.LogEnable
		case resource.DnsConfigUpdateModelTTL:
			update["ttl"] = req.Ttl
			updateTtl = true
		case resource.DnsConfigUpdateModelDnssec:
			update["dnssec_enable"] = req.DnssecEnable
		case resource.DnsConfigUpdateModelBlackhole:
			update["blackhole_enable"] = req.BlackholeEnable
			update["blackholes"] = req.Blackholes
		case resource.DnsConfigUpdateModelRecursion:
			update["recursion_enable"] = req.RecursionEnable
		case resource.DnsConfigUpdateModelRecursive:
			update["recursive_clients"] = req.RecursiveClients
		case resource.DnsConfigUpdateModelTransferPort:
			update["transfer_port"] = req.TransferPort
		default:
			return fmt.Errorf("unknown updateState")
		}

		if _, err := tx.Update(resource.TableDnsGlobalConfig, update,
			map[string]interface{}{restdb.IDField: defaultGlobalConfigID}); err != nil {
			return fmt.Errorf("update dnsGlobalConfig to db failed:%s", err.Error())
		}

		if updateTtl {
			if _, err := tx.Exec("update gr_agent_auth_zone set ttl = '" +
				strconv.FormatUint(uint64(req.Ttl), 10) + "'"); err != nil {
				return fmt.Errorf("update updateZonesTTLSQL to db failed:%s",
					err.Error())
			}
			if _, err := tx.Exec("update gr_agent_redirection set ttl = '" +
				strconv.FormatUint(uint64(req.Ttl), 10) + "'"); err != nil {
				return fmt.Errorf("update updateRedirectionTtlSQL to db failed:%s",
					err.Error())
			}

			if err := handler.UpdateAllRRTtl(req.Ttl, tx); err != nil {
				return err
			}

			if err := handler.initRPZFile(tx); err != nil {
				return fmt.Errorf("updateGlobalConfig initRPZFile failed:%s", err.Error())
			}
			if err := handler.initRedirectFile(tx); err != nil {
				return fmt.Errorf("updateGlobalConfig initRedirectFile failed:%s", err.Error())
			}
			if err := handler.rndcReconfig(); err != nil {
				return fmt.Errorf("updateGlobalConfig rndcReconfig failed:%s", err.Error())
			}
		}

		if err := handler.rewriteNamedOptionsFile(tx); err != nil {
			return fmt.Errorf("updateGlobalConfig rewriteNamedOptionsFile failed:%s",
				err.Error())
		}

		return nil
	})
}

func (handler *DNSHandler) UpdateAllRRTtl(ttl uint32, tx restdb.Transaction) error {
	var rrs []*resource.AgentAuthRr
	var zones []*resource.AgentAuthZone
	if err := tx.Fill(nil, &rrs); err != nil {
		return fmt.Errorf("update all rrs ttl when get rrs from db falied:%s", err.Error())
	}

	if err := tx.Fill(nil, &zones); err != nil {
		return fmt.Errorf("update all rrs ttl when get zones from db falied:%s", err.Error())
	}

	if _, err := tx.Exec("update gr_agent_auth_rr set ttl = '" + strconv.FormatUint(uint64(ttl), 10) + "'"); err != nil {
		return fmt.Errorf("update rrs ttl to db failed:%s", err.Error())
	}

	for _, zone := range zones {
		if err := handler.rewriteAuthZoneFile(tx, zone); err != nil {
			return fmt.Errorf("rewrite auth zone %s with view %s file failed:%s",
				zone.Name, zone.AgentView, err.Error())
		}

		if err := handler.rndcModifyZone(zone); err != nil {
			return fmt.Errorf("update auth zone %s with view %s to dns failed:%s",
				zone.Name, zone.AgentView, err.Error())
		}
	}

	return nil
}

func getInterfaceIPs() ([]string, error) {
	var interfaces4 []string
	its, err := net.Interfaces()
	if err != nil {
		return interfaces4, nil
	}

	for _, it := range its {
		addrs, err := it.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			ipnet, ok := addr.(*net.IPNet)
			if ok == false {
				continue
			}

			ip := ipnet.IP
			if ip.To4() != nil {
				if ip.IsGlobalUnicast() {
					interfaces4 = append(interfaces4, ip.String())
				}
			}
		}
	}

	return interfaces4, nil
}
