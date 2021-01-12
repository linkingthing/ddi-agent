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

func (service *DNSService) CreateAcl(context context.Context, req *pb.CreateAclReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateACL(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UpdateAcl(context context.Context, req *pb.UpdateAclReq) (*pb.DDIResponse, error) {
	if err := service.handler.UpdateACL(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) DeleteAcl(context context.Context, req *pb.DeleteAclReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteACL(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) BatchCreateAcl(context context.Context, req *pb.BatchCreateAclReq) (*pb.DDIResponse, error) {
	if err := service.handler.BatchCreateACL(req); err != nil {
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

func (service *DNSService) CreateForwardZone(context context.Context, req *pb.CreateForwardZoneReq) (*pb.DDIResponse, error) {
	if err := service.handler.CreateForwardZone(req); err != nil {
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

func (service *DNSService) DeleteForwardZone(context context.Context, req *pb.DeleteForwardZoneReq) (*pb.DDIResponse, error) {
	if err := service.handler.DeleteForwardZone(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) FlushForwardZone(context context.Context, req *pb.FlushForwardZoneReq) (*pb.DDIResponse, error) {
	if err := service.handler.FlushForwardZone(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}

func (service *DNSService) UploadLog(context context.Context, req *pb.UploadLogReq) (*pb.DDIResponse, error) {
	if err := service.handler.UploadLog(*req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	}

	return &pb.DDIResponse{Succeed: true}, nil
}
