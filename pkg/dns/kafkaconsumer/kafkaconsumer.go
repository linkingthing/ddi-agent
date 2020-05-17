package kafkaconsumer

import (
	"context"

	"github.com/golang/protobuf/proto"
	kg "github.com/segmentio/kafka-go"
	"google.golang.org/grpc"

	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-agent/config"
	"github.com/linkingthing/ddi-agent/pkg/pb"
	"github.com/linkingthing/ddi-metric/register"
)

const (
	STARTDNS                  = "StartDNS"
	STOPDNS                   = "StopDNS"
	CREATEACL                 = "CreateACL"
	UPDATEACL                 = "UpdateACL"
	DELETEACL                 = "DeleteACL"
	CREATEVIEW                = "CreateView"
	UPDATEVIEW                = "UpdateView"
	DELETEVIEW                = "DeleteView"
	CREATEZONE                = "CreateZone"
	DELETEZONE                = "DeleteZone"
	CREATERR                  = "CreateRR"
	UPDATERR                  = "UpdateRR"
	DELETERR                  = "DeleteRR"
	UPDATEDEFAULTFORWARD      = "UpdateDefaultForward"
	DELETEDEFAULTFORWARD      = "DeleteDefaultForward"
	UPDATEFORWARD             = "UpdateForward"
	DELETEFORWARD             = "DeleteForward"
	CREATEREDIRECTION         = "CreateRedirection"
	UPDATEREDIRECTION         = "UpdateRedirection"
	DELETEREDIRECTION         = "DeleteRedirection"
	CREATEDEFAULTDNS64        = "CreateDefaultDNS64"
	UPDATEDEFAULTDNS64        = "UpdateDefaultDNS64"
	DELETEDEFAULTDNS64        = "DeleteDefaultDNS64"
	CREATEDNS64               = "CreateDNS64"
	UPDATEDNS64               = "UpdateDNS64"
	DELETEDNS64               = "DeleteDNS64"
	CREATEIPBLACKHOLE         = "CreateIPBlackHole"
	UPDATEIPBLACKHOLE         = "UpdateIPBlackHole"
	DELETEIPBLACKHOLE         = "DeleteIPBlackHole"
	UPDATERECURSIVECONCURRENT = "UpdateRecursiveConcurrent"
	CREATESORTLIST            = "CreateSortList"
	UPDATESORTLIST            = "UpdateSortList"
	DELETESORTLIST            = "DeleteSortList"
)

var (
	DNSTopic = "dns"
)

func New(conn *grpc.ClientConn, conf *config.AgentConfig) {
	if conf.Server.DNSEnabled == false {
		return
	}

	register.RegisterNode(conf.Server.Hostname, conf.Prometheus.IP, conf.Prometheus.Port,
		conf.Server.IP, conf.Server.ParentIP, register.DNSRole, conf.Kafka.Addr)

	cli := pb.NewAgentManagerClient(conn)
	kafkaReader := kg.NewReader(kg.ReaderConfig{
		Brokers: []string{conf.Kafka.Addr},
		Topic:   DNSTopic,
	})

	for {
		message, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Errorf("read message from kafka failed: %s", err.Error())
			return
		}

		switch string(message.Key) {
		case STARTDNS:
			var target pb.DNSStartReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.StartDNS(context.Background(), &target)
		case STOPDNS:
			var target pb.DNSStopReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.StopDNS(context.Background(), &target)
		case CREATEACL:
			var target pb.CreateACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateACL(context.Background(), &target)
		case UPDATEACL:
			var target pb.UpdateACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateACL(context.Background(), &target)
		case DELETEACL:
			var target pb.DeleteACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteACL(context.Background(), &target)
		case CREATEVIEW:
			var target pb.CreateViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateView(context.Background(), &target)
		case UPDATEVIEW:
			var target pb.UpdateViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateView(context.Background(), &target)
		case DELETEVIEW:
			var target pb.DeleteViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteView(context.Background(), &target)
		case CREATEZONE:
			var target pb.CreateZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateZone(context.Background(), &target)
		case DELETEZONE:
			var target pb.DeleteZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteZone(context.Background(), &target)
		case CREATERR:
			var target pb.CreateRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateRR(context.Background(), &target)
		case UPDATERR:
			var target pb.UpdateRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateRR(context.Background(), &target)
		case DELETERR:
			var target pb.DeleteRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteRR(context.Background(), &target)
		case UPDATEDEFAULTFORWARD:
			var target pb.UpdateDefaultForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateDefaultForward(context.Background(), &target)
		case DELETEDEFAULTFORWARD:
			var target pb.DeleteDefaultForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteDefaultForward(context.Background(), &target)
		case UPDATEFORWARD:
			var target pb.UpdateForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateForward(context.Background(), &target)
		case DELETEFORWARD:
			var target pb.DeleteForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteForward(context.Background(), &target)
		case CREATEREDIRECTION:
			var target pb.CreateRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateRedirection(context.Background(), &target)
		case UPDATEREDIRECTION:
			var target pb.UpdateRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateRedirection(context.Background(), &target)
		case DELETEREDIRECTION:
			var target pb.DeleteRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteRedirection(context.Background(), &target)
		case CREATEDEFAULTDNS64:
			var target pb.CreateDefaultDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateDefaultDNS64(context.Background(), &target)
		case UPDATEDEFAULTDNS64:
			var target pb.UpdateDefaultDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateDefaultDNS64(context.Background(), &target)
		case DELETEDEFAULTDNS64:
			var target pb.DeleteDefaultDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteDefaultDNS64(context.Background(), &target)
		case CREATEDNS64:
			var target pb.CreateDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateDNS64(context.Background(), &target)
		case UPDATEDNS64:
			var target pb.UpdateDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateDNS64(context.Background(), &target)
		case DELETEDNS64:
			var target pb.DeleteDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteDNS64(context.Background(), &target)
		case CREATEIPBLACKHOLE:
			var target pb.CreateIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateIPBlackHole(context.Background(), &target)
		case UPDATEIPBLACKHOLE:
			var target pb.UpdateIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateIPBlackHole(context.Background(), &target)
		case DELETEIPBLACKHOLE:
			var target pb.DeleteIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteIPBlackHole(context.Background(), &target)
		case UPDATERECURSIVECONCURRENT:
			var target pb.UpdateRecurConcuReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateRecursiveConcurrent(context.Background(), &target)
		case CREATESORTLIST:
			var target pb.CreateSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateSortList(context.Background(), &target)
		case UPDATESORTLIST:
			var target pb.UpdateSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateSortList(context.Background(), &target)
		case DELETESORTLIST:
			var target pb.DeleteSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteSortList(context.Background(), &target)

		}
	}
}
