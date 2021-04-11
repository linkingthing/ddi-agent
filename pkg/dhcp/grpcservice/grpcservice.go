package grpcservice

import (
	"context"

	"github.com/linkingthing/ddi-agent/config"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

type DHCPService struct {
	handler *DHCPHandler
}

func New(conf *config.AgentConfig) (*DHCPService, error) {
	handler, err := newDHCPHandler(conf)
	if err != nil {
		return nil, err
	}

	return &DHCPService{handler: handler}, nil
}

func (s *DHCPService) CreateSubnet4(ctx context.Context, req *pb.CreateSubnet4Request) (*pb.DDIResponse, error) {
	if err := s.handler.CreateSubnet4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdateSubnet4(ctx context.Context, req *pb.UpdateSubnet4Request) (*pb.DDIResponse, error) {
	if err := s.handler.UpdateSubnet4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) DeleteSubnet4(ctx context.Context, req *pb.DeleteSubnet4Request) (*pb.DDIResponse, error) {
	if err := s.handler.DeleteSubnet4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) CreateSubnet6(ctx context.Context, req *pb.CreateSubnet6Request) (*pb.DDIResponse, error) {
	if err := s.handler.CreateSubnet6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) DeleteSubnet6(ctx context.Context, req *pb.DeleteSubnet6Request) (*pb.DDIResponse, error) {
	if err := s.handler.DeleteSubnet6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdateSubnet6(ctx context.Context, req *pb.UpdateSubnet6Request) (*pb.DDIResponse, error) {
	if err := s.handler.UpdateSubnet6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) CreatePool4(ctx context.Context, req *pb.CreatePool4Request) (*pb.DDIResponse, error) {
	if err := s.handler.CreatePool4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) DeletePool4(ctx context.Context, req *pb.DeletePool4Request) (*pb.DDIResponse, error) {
	if err := s.handler.DeletePool4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdatePool4(ctx context.Context, req *pb.UpdatePool4Request) (*pb.DDIResponse, error) {
	if err := s.handler.UpdatePool4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) CreatePool6(ctx context.Context, req *pb.CreatePool6Request) (*pb.DDIResponse, error) {
	if err := s.handler.CreatePool6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) DeletePool6(ctx context.Context, req *pb.DeletePool6Request) (*pb.DDIResponse, error) {
	if err := s.handler.DeletePool6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdatePool6(ctx context.Context, req *pb.UpdatePool6Request) (*pb.DDIResponse, error) {
	if err := s.handler.UpdatePool6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) CreatePDPool(ctx context.Context, req *pb.CreatePDPoolRequest) (*pb.DDIResponse, error) {
	if err := s.handler.CreatePDPool(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) DeletePDPool(ctx context.Context, req *pb.DeletePDPoolRequest) (*pb.DDIResponse, error) {
	if err := s.handler.DeletePDPool(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdatePDPool(ctx context.Context, req *pb.UpdatePDPoolRequest) (*pb.DDIResponse, error) {
	if err := s.handler.UpdatePDPool(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) CreateReservation4(ctx context.Context, req *pb.CreateReservation4Request) (*pb.DDIResponse, error) {
	if err := s.handler.CreateReservation4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) DeleteReservation4(ctx context.Context, req *pb.DeleteReservation4Request) (*pb.DDIResponse, error) {
	if err := s.handler.DeleteReservation4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdateReservation4(ctx context.Context, req *pb.UpdateReservation4Request) (*pb.DDIResponse, error) {
	if err := s.handler.UpdateReservation4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) CreateReservation6(ctx context.Context, req *pb.CreateReservation6Request) (*pb.DDIResponse, error) {
	if err := s.handler.CreateReservation6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) DeleteReservation6(ctx context.Context, req *pb.DeleteReservation6Request) (*pb.DDIResponse, error) {
	if err := s.handler.DeleteReservation6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdateReservation6(ctx context.Context, req *pb.UpdateReservation6Request) (*pb.DDIResponse, error) {
	if err := s.handler.UpdateReservation6(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) CreateClientClass4(ctx context.Context, req *pb.CreateClientClass4Request) (*pb.DDIResponse, error) {
	if err := s.handler.CreateClientClass4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) DeleteClientClass4(ctx context.Context, req *pb.DeleteClientClass4Request) (*pb.DDIResponse, error) {
	if err := s.handler.DeleteClientClass4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdateClientClass4(ctx context.Context, req *pb.UpdateClientClass4Request) (*pb.DDIResponse, error) {
	if err := s.handler.UpdateClientClass4(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) UpdateGlobalConfig(ctx context.Context, req *pb.UpdateGlobalConfigRequest) (*pb.DDIResponse, error) {
	if err := s.handler.UpdateGlobalConfig(req); err != nil {
		return &pb.DDIResponse{Succeed: false}, err
	} else {
		return &pb.DDIResponse{Succeed: true}, nil
	}
}

func (s *DHCPService) GetSubnetsLeasesCount(ctx context.Context, req *pb.GetSubnetsLeasesCountRequest) (*pb.GetSubnetsLeasesCountResponse, error) {
	if subnetsLeasesCount, err := s.handler.GetSubnetsLeasesCount(req); err != nil {
		return &pb.GetSubnetsLeasesCountResponse{Succeed: false}, err
	} else {
		return &pb.GetSubnetsLeasesCountResponse{Succeed: true, SubnetsLeasesCount: subnetsLeasesCount}, nil
	}
}

func (s *DHCPService) GetSubnet4Leases(ctx context.Context, req *pb.GetSubnet4LeasesRequest) (*pb.GetLeasesResponse, error) {
	if leases, err := s.handler.GetSubnet4Leases(req); err != nil {
		return &pb.GetLeasesResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesResponse{Succeed: true, Leases: leases}, nil
	}
}

func (s *DHCPService) GetSubnet4LeasesCount(ctx context.Context, req *pb.GetSubnet4LeasesCountRequest) (*pb.GetLeasesCountResponse, error) {
	if count, err := s.handler.GetSubnet4LeasesCount(req); err != nil {
		return &pb.GetLeasesCountResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesCountResponse{Succeed: true, LeasesCount: count}, nil
	}
}

func (s *DHCPService) GetPool4Leases(ctx context.Context, req *pb.GetPool4LeasesRequest) (*pb.GetLeasesResponse, error) {
	if leases, err := s.handler.GetPool4Leases(req); err != nil {
		return &pb.GetLeasesResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesResponse{Succeed: true, Leases: leases}, nil
	}
}

func (s *DHCPService) GetPool4LeasesCount(ctx context.Context, req *pb.GetPool4LeasesCountRequest) (*pb.GetLeasesCountResponse, error) {
	if count, err := s.handler.GetPool4LeasesCount(req); err != nil {
		return &pb.GetLeasesCountResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesCountResponse{Succeed: true, LeasesCount: count}, nil
	}
}

func (s *DHCPService) GetReservation4LeasesCount(ctx context.Context, req *pb.GetReservation4LeasesCountRequest) (*pb.GetLeasesCountResponse, error) {
	if count, err := s.handler.GetReservation4LeasesCount(req); err != nil {
		return &pb.GetLeasesCountResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesCountResponse{Succeed: true, LeasesCount: count}, nil
	}
}

func (s *DHCPService) GetSubnet6Leases(ctx context.Context, req *pb.GetSubnet6LeasesRequest) (*pb.GetLeasesResponse, error) {
	if leases, err := s.handler.GetSubnet6Leases(req); err != nil {
		return &pb.GetLeasesResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesResponse{Succeed: true, Leases: leases}, nil
	}
}

func (s *DHCPService) GetSubnet6LeasesCount(ctx context.Context, req *pb.GetSubnet6LeasesCountRequest) (*pb.GetLeasesCountResponse, error) {
	if count, err := s.handler.GetSubnet6LeasesCount(req); err != nil {
		return &pb.GetLeasesCountResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesCountResponse{Succeed: true, LeasesCount: count}, nil
	}
}

func (s *DHCPService) GetPool6Leases(ctx context.Context, req *pb.GetPool6LeasesRequest) (*pb.GetLeasesResponse, error) {
	if leases, err := s.handler.GetPool6Leases(req); err != nil {
		return &pb.GetLeasesResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesResponse{Succeed: true, Leases: leases}, nil
	}
}

func (s *DHCPService) GetPool6LeasesCount(ctx context.Context, req *pb.GetPool6LeasesCountRequest) (*pb.GetLeasesCountResponse, error) {
	if count, err := s.handler.GetPool6LeasesCount(req); err != nil {
		return &pb.GetLeasesCountResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesCountResponse{Succeed: true, LeasesCount: count}, nil
	}
}

func (s *DHCPService) GetReservation6LeasesCount(ctx context.Context, req *pb.GetReservation6LeasesCountRequest) (*pb.GetLeasesCountResponse, error) {
	if count, err := s.handler.GetReservation6LeasesCount(req); err != nil {
		return &pb.GetLeasesCountResponse{Succeed: false}, err
	} else {
		return &pb.GetLeasesCountResponse{Succeed: true, LeasesCount: count}, nil
	}
}
