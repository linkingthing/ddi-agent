package grpcservice

import (
	"context"

	"github.com/linkingthing/ddi-agent/config"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

type DNSService struct {
	handler *DNSHandler
}

func New(conf *config.AgentConfig) (*DNSService, error) {
	handler, err := newDNSHandler(conf)
	if err != nil {
		return nil, err
	}

	return &DNSService{handler: handler}, nil
}

func (service *DNSService) StartDNS(content context.Context, req *pb.DNSStartReq) (*pb.DDIResponse, error) {
	if err := service.handler.StartDNS(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) StopDNS(context context.Context, req *pb.DNSStopReq) (*pb.DDIResponse, error) {
	if err := service.handler.StopDNS(); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) Close() {
	service.handler.Close()
}

func (service *DNSService) UpdateGlobalConfig(context context.Context, req *pb.UpdateGlobalConfigReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateGlobalConfig(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) CreateACL(context context.Context, req *pb.CreateACLReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateACL(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateACL(context context.Context, req *pb.UpdateACLReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateACL(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) DeleteACL(context context.Context, req *pb.DeleteACLReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteACL(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) CreateView(context context.Context, req *pb.CreateViewReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateView(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateView(context context.Context, req *pb.UpdateViewReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateView(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) DeleteView(context context.Context, req *pb.DeleteViewReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteView(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) CreateZone(context context.Context, req *pb.CreateZoneReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateZone(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateZone(context context.Context, req *pb.UpdateZoneReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateZone(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) DeleteZone(context context.Context, req *pb.DeleteZoneReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteZone(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) CreateRR(context context.Context, req *pb.CreateRRReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateRR(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateRR(context context.Context, req *pb.UpdateRRReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateRR(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) DeleteRR(context context.Context, req *pb.DeleteRRReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteRR(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateRRsByZone(context context.Context, req *pb.UpdateRRsByZoneReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateRRsByZone(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) CreateRedirection(context context.Context, req *pb.CreateRedirectionReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateRedirection(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateRedirection(context context.Context, req *pb.UpdateRedirectionReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateRedirection(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) DeleteRedirection(context context.Context, req *pb.DeleteRedirectionReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteRedirection(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) CreateUrlRedirect(context context.Context, req *pb.CreateUrlRedirectReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateUrlRedirect(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateUrlRedirect(context context.Context, req *pb.UpdateUrlRedirectReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateUrlRedirect(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) DeleteUrlRedirect(context context.Context, req *pb.DeleteUrlRedirectReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteUrlRedirect(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateForward(context context.Context, req *pb.UpdateForwardReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateForward(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateForwardZone(context context.Context, req *pb.UpdateForwardZoneReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateForwardZone(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) CreateIPBlackHole(context context.Context, req *pb.CreateIPBlackHoleReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateIPBlackHole(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateIPBlackHole(context context.Context, req *pb.UpdateIPBlackHoleReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateIPBlackHole(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) DeleteIPBlackHole(context context.Context, req *pb.DeleteIPBlackHoleReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteIPBlackHole(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateRecursiveConcurrent(context context.Context, req *pb.UpdateRecurConcuReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateRecursiveConcurrent(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}
