package grpcclient

import (
	"google.golang.org/grpc"

	monitorpb "github.com/linkingthing/ddi-monitor/pkg/proto"
)

type GrpcClient struct {
	monitorClient monitorpb.DDIMonitorClient
}

var grpcClient *GrpcClient

func New(conn *grpc.ClientConn) {
	grpcClient = &GrpcClient{monitorClient: monitorpb.NewDDIMonitorClient(conn)}
}

func GetDDIMonitorGrpcClient() monitorpb.DDIMonitorClient {
	return grpcClient.monitorClient
}
