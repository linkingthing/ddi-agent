package grpcservice

import (
	"context"

	"github.com/linkingthing/ddi-agent/config"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

const (
	opSuccess = 0
	opFail    = 1
)

type DNSService struct {
	handler *DNSHandler
}

func New(conf *config.AgentConfig) (*DNSService, error) {
	handler, err := newDNSHandler(conf.DNS.ConfDir, conf.DNS.DBDir)
	if err != nil {
		return nil, err
	}
	return &DNSService{handler: handler}, nil
}

func (service *DNSService) StartDNS(content context.Context, req *pb.DNSStartReq) (*pb.DDIResponse, error) {
	err := service.handler.StartDNS(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (service *DNSService) StopDNS(context context.Context, req *pb.DNSStopReq) (*pb.DDIResponse, error) {
	err := service.handler.StopDNS()
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateACL(context context.Context, req *pb.CreateACLReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateACL(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateACL(context context.Context, req *pb.UpdateACLReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateACL(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteACL(context context.Context, req *pb.DeleteACLReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteACL(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateView(context context.Context, req *pb.CreateViewReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateView(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateView(context context.Context, req *pb.UpdateViewReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateView(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteView(context context.Context, req *pb.DeleteViewReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteView(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateZone(context context.Context, req *pb.CreateZoneReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateZone(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateZone(context context.Context, req *pb.UpdateZoneReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateZone(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteZone(context context.Context, req *pb.DeleteZoneReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteZone(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateForwardZone(context context.Context, req *pb.CreateForwardZoneReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateForwardZone(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateForwardZone(context context.Context, req *pb.UpdateForwardZoneReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateForwardZone(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteForwardZone(context context.Context, req *pb.DeleteForwardZoneReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteForwardZone(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateRR(context context.Context, req *pb.CreateRRReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateRR(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateRR(context context.Context, req *pb.UpdateRRReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateRR(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteRR(context context.Context, req *pb.DeleteRRReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteRR(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateForward(context context.Context, req *pb.CreateForwardReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateForward(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateForward(context context.Context, req *pb.UpdateForwardReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateForward(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteForward(context context.Context, req *pb.DeleteForwardReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteForward(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateRedirection(context context.Context, req *pb.CreateRedirectionReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateRedirection(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateRedirection(context context.Context, req *pb.UpdateRedirectionReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateRedirection(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteRedirection(context context.Context, req *pb.DeleteRedirectionReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteRedirection(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateDNS64(context context.Context, req *pb.CreateDNS64Req) (*pb.DDIResponse, error) {
	err := service.handler.CreateDNS64(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateDNS64(context context.Context, req *pb.UpdateDNS64Req) (*pb.DDIResponse, error) {
	err := service.handler.UpdateDNS64(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteDNS64(context context.Context, req *pb.DeleteDNS64Req) (*pb.DDIResponse, error) {
	err := service.handler.DeleteDNS64(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateIPBlackHole(context context.Context, req *pb.CreateIPBlackHoleReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateIPBlackHole(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateIPBlackHole(context context.Context, req *pb.UpdateIPBlackHoleReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateIPBlackHole(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteIPBlackHole(context context.Context, req *pb.DeleteIPBlackHoleReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteIPBlackHole(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateRecursiveConcurrent(context context.Context, req *pb.UpdateRecurConcuReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateRecursiveConcurrent(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) CreateSortList(context context.Context, req *pb.CreateSortListReq) (*pb.DDIResponse, error) {
	err := service.handler.CreateSortList(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) UpdateSortList(context context.Context, req *pb.UpdateSortListReq) (*pb.DDIResponse, error) {
	err := service.handler.UpdateSortList(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}
func (service *DNSService) DeleteSortList(context context.Context, req *pb.DeleteSortListReq) (*pb.DDIResponse, error) {
	err := service.handler.DeleteSortList(*req)
	if err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (service *DNSService) Close() {
	service.handler.Close()
}
