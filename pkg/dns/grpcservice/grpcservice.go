package grpcservice

import (
	"context"
	"fmt"

	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

const (
	opSuccess = 0
	opFail    = 1
)

type DNSService struct {
	handler *DNSHandler
}

func New(dnsConfPath string, agentPath string) (*DNSService, error) {
	handler, err := newDNSHandler(dnsConfPath, agentPath)
	if err != nil {
		return nil, err
	}
	return &DNSService{handler: handler}, nil
}

func (service *DNSService) StartDNS(content context.Context, req *pb.DNSStartReq) (*pb.OperResult, error) {
	err := service.handler.StartDNS(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}

func (service *DNSService) StopDNS(context context.Context, req *pb.DNSStopReq) (*pb.OperResult, error) {
	err := service.handler.StopDNS()
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateACL(context context.Context, req *pb.CreateACLReq) (*pb.OperResult, error) {
	err := service.handler.CreateACL(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateACL(context context.Context, req *pb.UpdateACLReq) (*pb.OperResult, error) {
	err := service.handler.UpdateACL(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteACL(context context.Context, req *pb.DeleteACLReq) (*pb.OperResult, error) {
	err := service.handler.DeleteACL(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateView(context context.Context, req *pb.CreateViewReq) (*pb.OperResult, error) {
	err := service.handler.CreateView(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateView(context context.Context, req *pb.UpdateViewReq) (*pb.OperResult, error) {
	err := service.handler.UpdateView(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteView(context context.Context, req *pb.DeleteViewReq) (*pb.OperResult, error) {
	err := service.handler.DeleteView(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateZone(context context.Context, req *pb.CreateZoneReq) (*pb.OperResult, error) {
	err := service.handler.CreateZone(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateZone(context context.Context, req *pb.UpdateZoneReq) (*pb.OperResult, error) {
	err := service.handler.UpdateZone(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteZone(context context.Context, req *pb.DeleteZoneReq) (*pb.OperResult, error) {
	err := service.handler.DeleteZone(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateForwardZone(context context.Context, req *pb.CreateForwardZoneReq) (*pb.OperResult, error) {
	err := service.handler.CreateForwardZone(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateForwardZone(context context.Context, req *pb.UpdateForwardZoneReq) (*pb.OperResult, error) {
	err := service.handler.UpdateForwardZone(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteForwardZone(context context.Context, req *pb.DeleteForwardZoneReq) (*pb.OperResult, error) {
	err := service.handler.DeleteForwardZone(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateRR(context context.Context, req *pb.CreateRRReq) (*pb.OperResult, error) {
	err := service.handler.CreateRR(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateRR(context context.Context, req *pb.UpdateRRReq) (*pb.OperResult, error) {
	err := service.handler.UpdateRR(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteRR(context context.Context, req *pb.DeleteRRReq) (*pb.OperResult, error) {
	err := service.handler.DeleteRR(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateForward(context context.Context, req *pb.CreateForwardReq) (*pb.OperResult, error) {
	err := service.handler.CreateForward(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateForward(context context.Context, req *pb.UpdateForwardReq) (*pb.OperResult, error) {
	err := service.handler.UpdateForward(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteForward(context context.Context, req *pb.DeleteForwardReq) (*pb.OperResult, error) {
	err := service.handler.DeleteForward(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateRedirection(context context.Context, req *pb.CreateRedirectionReq) (*pb.OperResult, error) {
	err := service.handler.CreateRedirection(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateRedirection(context context.Context, req *pb.UpdateRedirectionReq) (*pb.OperResult, error) {
	err := service.handler.UpdateRedirection(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteRedirection(context context.Context, req *pb.DeleteRedirectionReq) (*pb.OperResult, error) {
	err := service.handler.DeleteRedirection(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateDNS64(context context.Context, req *pb.CreateDNS64Req) (*pb.OperResult, error) {
	err := service.handler.CreateDNS64(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateDNS64(context context.Context, req *pb.UpdateDNS64Req) (*pb.OperResult, error) {
	err := service.handler.UpdateDNS64(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteDNS64(context context.Context, req *pb.DeleteDNS64Req) (*pb.OperResult, error) {
	err := service.handler.DeleteDNS64(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateIPBlackHole(context context.Context, req *pb.CreateIPBlackHoleReq) (*pb.OperResult, error) {
	err := service.handler.CreateIPBlackHole(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateIPBlackHole(context context.Context, req *pb.UpdateIPBlackHoleReq) (*pb.OperResult, error) {
	err := service.handler.UpdateIPBlackHole(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteIPBlackHole(context context.Context, req *pb.DeleteIPBlackHoleReq) (*pb.OperResult, error) {
	err := service.handler.DeleteIPBlackHole(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateRecursiveConcurrent(context context.Context, req *pb.UpdateRecurConcuReq) (*pb.OperResult, error) {
	err := service.handler.UpdateRecursiveConcurrent(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) CreateSortList(context context.Context, req *pb.CreateSortListReq) (*pb.OperResult, error) {
	err := service.handler.CreateSortList(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) UpdateSortList(context context.Context, req *pb.UpdateSortListReq) (*pb.OperResult, error) {
	err := service.handler.UpdateSortList(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}
func (service *DNSService) DeleteSortList(context context.Context, req *pb.DeleteSortListReq) (*pb.OperResult, error) {
	err := service.handler.DeleteSortList(*req)
	if err != nil {
		return &pb.OperResult{RetCode: opFail, RetMsg: fmt.Sprintf("%s", err)}, err
	} else {
		return &pb.OperResult{RetCode: opSuccess}, nil
	}
}

func (service *DNSService) Close() {
	service.handler.Close()
}
