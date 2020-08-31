package grpcserver

import (
	"fmt"
	"net"

	"google.golang.org/grpc"

	"github.com/linkingthing/ddi-agent/config"
	dhcpsrv "github.com/linkingthing/ddi-agent/pkg/dhcp/grpcservice"
	dnssrv "github.com/linkingthing/ddi-agent/pkg/dns/grpcservice"
	"github.com/linkingthing/ddi-agent/pkg/proto"
)

type GRPCServer struct {
	server   *grpc.Server
	listener net.Listener
}

func New(conf *config.AgentConfig) (*GRPCServer, error) {
	listener, err := net.Listen("tcp", conf.Server.GrpcAddr)
	if err != nil {
		return nil, fmt.Errorf("create listener with addr %s failed: %s", conf.Server.GrpcAddr, err.Error())
	}

	grpcServer := &GRPCServer{
		server:   grpc.NewServer(),
		listener: listener,
	}

	if conf.DNS.Enabled {
		dnsService, err := dnssrv.New(conf)
		if err != nil {
			return nil, fmt.Errorf("create dns grpc service failed: %s", err.Error())
		}
		proto.RegisterAgentManagerServer(grpcServer.server, dnsService)
	}

	if conf.DHCP.Enabled {
		dhcpService, err := dhcpsrv.New(conf)
		if err != nil {
			return nil, fmt.Errorf("create dhcp grpc service failed: %s", err.Error())
		}
		proto.RegisterDHCPManagerServer(grpcServer.server, dhcpService)
	}

	return grpcServer, nil
}

func (s *GRPCServer) Run() error {
	return s.server.Serve(s.listener)
}

func (s *GRPCServer) Stop() error {
	s.server.GracefulStop()
	return nil
}
