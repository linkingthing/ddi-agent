package grpcserver

import (
	"google.golang.org/grpc"
	"log"
	"net"

	grpcservice "github.com/linkingthing/ddi-agent/pkg/dns/grpcservice"
	"github.com/linkingthing/ddi-metric/pb"
)

type GRPCServer struct {
	dnsService *grpcservice.DNSService
	server     *grpc.Server
	listener   net.Listener
}

func NewGRPCServer(listenAddr string, DNSConfPath string, agentPath string, isDnsOpen bool, isDhcpOpen bool) (*GRPCServer, error) {
	server := grpc.NewServer()
	var dnsService *grpcservice.DNSService
	if isDnsOpen {
		dnsService = grpcservice.NewDNSService(DNSConfPath, agentPath)
	}
	log.Println("in server.go, to register")
	pb.RegisterAgentManagerServer(server, dnsService)
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Println("in server.go, error to listen")
		return nil, err
	}
	return &GRPCServer{
		dnsService: dnsService,
		server:     server,
		listener:   listener,
	}, nil
}

func (s *GRPCServer) Start() error {
	return s.server.Serve(s.listener)
}

func (s *GRPCServer) Stop() error {
	s.server.GracefulStop()
	s.dnsService.Close()
	return nil
}
