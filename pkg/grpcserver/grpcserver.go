package grpcserver

import (
	"log"
	"net"
	"os"
	"path/filepath"
	"strings"

	"google.golang.org/grpc"

	"github.com/linkingthing/ddi-agent/config"
	dns "github.com/linkingthing/ddi-agent/pkg/dns/grpcservice"
	"github.com/linkingthing/ddi-metric/pb"
)

type GRPCServer struct {
	dnsService *grpcservice.DNSService
	server     *grpc.Server
	listener   net.Listener
}

func New(conf *config.AgentConfig) (*GRPCServer, error) {
	agentExecDir, err := getAgentExecDirectory()
	if err != nil {
		return nil, fmt.Errorf("get agent exec directory failed:%s", err.Error())
	}

	listener, err := net.Listen("tcp", conf.Grpc.Addr)
	if err != nil {
		return nil, fmt.Errorf("create listener with addr %s failed: %s", conf.Grpc.Addr, err.Error())
	}

	grpcServer := &GRPCServer{
		server:   grpc.NewServer(),
		listener: listener,
	}

	if conf.Server.DnsEnabled {
		grpcServer.dnsService = dns.NewDNSService(conf.Dns.ConfDir, agentExecDir)
		pb.RegisterAgentManagerServer(grpcServer.server, grpcServer.dnsService)
		//TODO add DHCP service and register dhcp service to grpc server
	}

	return grpcServer, nil
}

func getAgentExecDirectory() (string, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return "", err
	}
	return strings.Replace(dir, "\\", "/", -1), nil
}

func (s *GRPCServer) Run() error {
	return s.server.Serve(s.listener)
}

func (s *GRPCServer) Stop() error {
	s.server.GracefulStop()
	s.dnsService.Close()
	return nil
}
