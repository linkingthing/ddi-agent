package grpcservice

import (
	"testing"

	ut "github.com/zdnscloud/cement/unittest"

	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

var (
	handler *DNSHandler
)

func init() {
	//p := newDNSHandler("/root/bindtest/", "/root/bindtest/", "/home/lx/nginx/conf.d/", "10.0.0.19")
	//handler = p
}

func TestStartDNS(t *testing.T) {
	config := ""
	dnsStartReq := pb.DNSStartReq{Config: config}
	err := handler.StartDNS(dnsStartReq)
	ut.Assert(t, err == nil, "start successfully:%v", err)
}

func TestCreateACL(t *testing.T) {
	var ipList = []string{"10.0.0.0/24", "192.168.198.0/24"}
	createACLReq := pb.CreateACLReq{
		Name: "southchina",
		ID:   "ACL001",
		IPs:  ipList}
	err := handler.CreateACL(createACLReq)
	ut.Assert(t, err == nil, "Create ACL successfully!:%v", err)
}

func TestUpdateACL(t *testing.T) {
	var ipList = []string{"10.0.0.0/24", "192.168.191.0/24"}
	updateACLReq := pb.UpdateACLReq{
		Name:   "southchina",
		ID:     "ACL001",
		NewIPs: ipList}
	err := handler.UpdateACL(updateACLReq)
	ut.Assert(t, err == nil, "Create ACL successfully!:%v", err)
}

func TestDeleteACL(t *testing.T) {
	deleteACLReq := pb.DeleteACLReq{ID: "ACL001"}
	err := handler.DeleteACL(deleteACLReq)
	ut.Assert(t, err == nil, "Delete ACL successfully!:%v", err)
}

func TestCreateView(t *testing.T) {
	TestCreateACL(t)
	createViewReq := pb.CreateViewReq{
		ViewName: "DianXinView",
		ViewID:   "viewID001",
		Priority: 1,
		ACLIDs:   []string{"ACL001"}}
	err := handler.CreateView(createViewReq)
	ut.Assert(t, err == nil, "Create View Success!:%v", err)
}

func TestUpdateView(t *testing.T) {
	var ipList = []string{"192.168.199.0/24", "192.168.198.0/24"}
	createACLReq := pb.CreateACLReq{
		Name: "southchina_2",
		ID:   "ACL002",
		IPs:  ipList}
	err := handler.CreateACL(createACLReq)
	ut.Assert(t, err == nil, "Create ACL successfully!:%v", err)
	updateViewReq := pb.UpdateViewReq{
		ViewID:       "viewID001",
		Priority:     1,
		DeleteACLIDs: []string{"ACL001"},
		AddACLIDs:    []string{"ACL002"}}
	err = handler.UpdateView(updateViewReq)
	ut.Assert(t, err == nil, "Create View Success!:%v", err)
}

func TestCreateZone(t *testing.T) {
	createZoneReq := pb.CreateZoneReq{ViewID: "viewID001", ZoneID: "zoneID001", ZoneName: "test1031.com", ZoneFileName: "test1031.com.zone"}
	err := handler.CreateZone(createZoneReq)
	ut.Assert(t, err == nil, "Create Zone Success!:%v", err)
}

func TestCreateRR(t *testing.T) {
	createRRReq := pb.CreateRRReq{ViewID: "viewID001", ZoneID: "zoneID001", RRID: "rr002", Name: "mail.test1031.com", TTL: "1000", Type: "A", Value: "10.2.21.1"}
	err := handler.CreateRR(createRRReq)
	ut.Assert(t, err == nil, "Create RR Success!:%v", err)
	createRRReq = pb.CreateRRReq{ViewID: "viewID001", ZoneID: "zoneID001", RRID: "rr003", Name: "mail.test1031.com", TTL: "1000", Type: "A", Value: "10.2.21.2"}
	err = handler.CreateRR(createRRReq)
	ut.Assert(t, err == nil, "Create RR Success!:%v", err)
}

func TestUpdateRR(t *testing.T) {
	updateRRReq := pb.UpdateRRReq{ViewID: "viewID001", ZoneID: "zoneID001", RRID: "rr002", Name: "mail.test1031.com", TTL: "1000", Type: "A", Value: "10.2.21.3"}
	err := handler.UpdateRR(updateRRReq)
	ut.Assert(t, err == nil, "Update RR Success!:%v", err)

	updateRRReq = pb.UpdateRRReq{ViewID: "viewID001", ZoneID: "zoneID001", RRID: "rr003", Name: "mail.test1031.com", TTL: "1000", Type: "A", Value: "10.2.21.4"}
	err = handler.UpdateRR(updateRRReq)
	ut.Assert(t, err == nil, "Update RR Success!:%v", err)
}

func TestUpdateDefaultForward(t *testing.T) {
	ips := []string{"10.0.0.12", "10.0.0.13", "10.0.0.14"}
	updateForward := pb.UpdateDefaultForwardReq{Type: "first", IPs: ips}
	err := handler.UpdateDefaultForward(updateForward)
	ut.Assert(t, err == nil, "Update DefaultForward Success!:%v", err)
}

func TestUpdateForward(t *testing.T) {
	ips := []string{"10.0.0.55", "10.0.0.56", "10.0.0.57"}
	updateForward := pb.UpdateForwardReq{ViewID: "viewID001", ZoneID: "zoneID001", Type: "first", IPs: ips}
	err := handler.UpdateForward(updateForward)
	ut.Assert(t, err == nil, "Update Forward Success!:%v", err)
}

func TestCreateDefaultDNS64(t *testing.T) {
	req := pb.CreateDefaultDNS64Req{ID: "020301", Prefix: "64:FF9C::/96", ClientACL: "ACL001", AAddress: "ACL002"}
	err := handler.CreateDefaultDNS64(req)
	ut.Assert(t, err == nil, "Create Default DNS64 Success!:%v", err)
}

func TestUpdateDefaultDNS64(t *testing.T) {
	req := pb.UpdateDefaultDNS64Req{ID: "020301", Prefix: "64:FF9C::/96", ClientACL: "ACL002", AAddress: "ACL001"}
	err := handler.UpdateDefaultDNS64(req)
	ut.Assert(t, err == nil, "Update Default DNS64 Success!:%v", err)
}

func TestCreateDNS64(t *testing.T) {
	req := pb.CreateDNS64Req{ID: "020302", ViewID: "viewID001", Prefix: "64:FF9C::/96", ClientACL: "ACL001", AAddress: "ACL002"}
	err := handler.CreateDNS64(req)
	ut.Assert(t, err == nil, "Create  DNS64 Success!:%v", err)
}

func TestUpdateDNS64(t *testing.T) {
	req := pb.UpdateDNS64Req{ID: "020302", ViewID: "viewID001", Prefix: "64:FF9C::/96", ClientACL: "ACL002", AAddress: "ACL001"}
	err := handler.UpdateDNS64(req)
	ut.Assert(t, err == nil, "Update  DNS64 Success!:%v", err)
}

func TestCreateRedirection(t *testing.T) {
	req := pb.CreateRedirectionReq{ID: "020303", ViewID: "viewID001", Name: "www.baidu.com", TTL: "300", DataType: "AAAA", Value: "240e::ff:1", RedirectType: "redirect"}
	err := handler.CreateRedirection(req)
	ut.Assert(t, err == nil, "Create  Redirection Success!:%v", err)
	rpzReq := pb.CreateRedirectionReq{ID: "020304", ViewID: "viewID001", Name: "www.abc.com", TTL: "300", DataType: "AAAA", Value: "240e::ff:2", RedirectType: "rpz"}
	err = handler.CreateRedirection(rpzReq)
	ut.Assert(t, err == nil, "Create  RPZ Success!:%v", err)
}

func TestUpdateRedirection(t *testing.T) {
	req := pb.UpdateRedirectionReq{ID: "020303", ViewID: "viewID001", Name: "www.baidu.com", TTL: "300", DataType: "AAAA", Value: "240e::ff:aaa1", RedirectType: "redirect"}
	err := handler.UpdateRedirection(req)
	ut.Assert(t, err == nil, "Update  Redirection Success!:%v", err)
	rpzReq := pb.UpdateRedirectionReq{ID: "020304", ViewID: "viewID001", Name: "www.baidu.com", TTL: "300", DataType: "AAAA", Value: "240e::ff:aaa2", RedirectType: "rpz"}
	err = handler.UpdateRedirection(rpzReq)
	ut.Assert(t, err == nil, "Update  RPZ Success!:%v", err)
}

func TestCreateIPBlackHole(t *testing.T) {
	req := pb.CreateIPBlackHoleReq{ID: "020305", ACLID: "ACL001"}
	err := handler.CreateIPBlackHole(req)
	ut.Assert(t, err == nil, "Create  IPBlackHole Success!:%v", err)
}

func TestUpdateIPBlackHole(t *testing.T) {
	req := pb.UpdateIPBlackHoleReq{ID: "020305", ACLID: "ACL002"}
	err := handler.UpdateIPBlackHole(req)
	ut.Assert(t, err == nil, "Update  IPBlackHole Success!:%v", err)
}

func TestUpdateRecursiveConcurrent(t *testing.T) {
	req := pb.UpdateRecurConcuReq{RecursiveClients: "800", FetchesPerZone: "100"}
	err := handler.UpdateRecursiveConcurrent(req)
	ut.Assert(t, err == nil, "Update Recursive Concurrent Success!:%v", err)
}

func TestDeleteDefaultForward(t *testing.T) {
	deleteForward := pb.DeleteDefaultForwardReq{}
	err := handler.DeleteDefaultForward(deleteForward)
	ut.Assert(t, err == nil, "delete DefaultForward Success!:%v", err)
}

func TestDeleteDefaultDNS64(t *testing.T) {
	req := pb.DeleteDefaultDNS64Req{ID: "020301"}
	err := handler.DeleteDefaultDNS64(req)
	ut.Assert(t, err == nil, "Delete Default DNS64 Success!:%v", err)
}

func TestDeleteIPBlackHole(t *testing.T) {
	req := pb.DeleteIPBlackHoleReq{ID: "020305"}
	err := handler.DeleteIPBlackHole(req)
	ut.Assert(t, err == nil, "Delete  IPBlackHole Success!:%v", err)
}

func TestDeleteRedirection(t *testing.T) {
	req := pb.DeleteRedirectionReq{ID: "020303", ViewID: "viewID001", RedirectType: "redirect"}
	err := handler.DeleteRedirection(req)
	ut.Assert(t, err == nil, "Delete  Redirection Success!:%v", err)
	rpzReq := pb.DeleteRedirectionReq{ID: "020304", ViewID: "viewID001", RedirectType: "rpz"}
	err = handler.DeleteRedirection(rpzReq)
	ut.Assert(t, err == nil, "Delete  RPZ Success!:%v", err)
}

func TestDeleteDNS64(t *testing.T) {
	req := pb.DeleteDNS64Req{ID: "020302", ViewID: "viewID001"}
	err := handler.DeleteDNS64(req)
	ut.Assert(t, err == nil, "Delete  DNS64 Success!:%v", err)
}

func TestDeleteForward(t *testing.T) {
	deleteForward := pb.DeleteForwardReq{ViewID: "viewID001", ZoneID: "zoneID001"}
	err := handler.DeleteForward(deleteForward)
	ut.Assert(t, err == nil, "Delete Forward Success!:%v", err)
}

func TestDeleteRR(t *testing.T) {
	delRRReq := pb.DeleteRRReq{ViewID: "viewID001", ZoneID: "zoneID001", RRID: "rr002"}
	err := handler.DeleteRR(delRRReq)
	ut.Assert(t, err == nil, "Delete RR Success!:%v", err)
	delRRReq = pb.DeleteRRReq{ViewID: "viewID001", ZoneID: "zoneID001", RRID: "rr003"}
	err = handler.DeleteRR(delRRReq)
	ut.Assert(t, err == nil, "Delete RR Success!:%v", err)
}

func TestDeleteZone(t *testing.T) {
	delZoneReq := pb.DeleteZoneReq{ViewID: "viewID001", ZoneID: "zoneID001"}
	err := handler.DeleteZone(delZoneReq)
	ut.Assert(t, err == nil, "Create Delete Zone Success!:%v", err)
}

func TestDeleteView(t *testing.T) {
	delViewReq := pb.DeleteViewReq{ViewID: "viewID001"}
	err := handler.DeleteView(delViewReq)
	ut.Assert(t, err == nil, "Delete View Success!:%v", err)
	deleteACLReq := pb.DeleteACLReq{ID: "ACL001"}
	err = handler.DeleteACL(deleteACLReq)
	ut.Assert(t, err == nil, "Delete ACL successfully!:%v", err)
	deleteACLReq.ID = "ACL002"
	err = handler.DeleteACL(deleteACLReq)
	ut.Assert(t, err == nil, "Delete ACL successfully!:%v", err)
}

func TestStopDNS(t *testing.T) {
	err := handler.StopDNS()
	ut.Assert(t, err == nil, "stop successfully!:%v", err)
}
