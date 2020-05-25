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
	StartDNS                  = "StartDNS"
	StopDNS                   = "StopDNS"
	CreateACL                 = "CreateACL"
	UpdateACL                 = "UpdateACL"
	DeleteACL                 = "DeleteACL"
	CreateView                = "CreateView"
	UpdateView                = "UpdateView"
	DeleteView                = "DeleteView"
	CreateZone                = "CreateZone"
	UpdateZone                = "UpdateZone"
	DeleteZone                = "DeleteZone"
	CreateForwardZone         = "CreateForwardZone"
	UpdateForwardZone         = "UpdateForwardZone"
	DeleteForwardZone         = "DeleteForwardZone"
	CreateRR                  = "CreateRR"
	UpdateRR                  = "UpdateRR"
	DeleteRR                  = "DeleteRR"
	CreateForward             = "CreateForward"
	UpdateForward             = "UpdateForward"
	DeleteForward             = "DeleteForward"
	CreateRedirection         = "CreateRedirection"
	UpdateRedirection         = "UpdateRedirection"
	DeleteRedirection         = "DeleteRedirection"
	CreateDNS64               = "CreateDNS64"
	UpdateDNS64               = "UpdateDNS64"
	DeleteDNS64               = "DeleteDNS64"
	CreateIPBlackHole         = "CreateIPBlackHole"
	UpdateIPBlackHole         = "UpdateIPBlackHole"
	DeleteIPBlackHole         = "DeleteIPBlackHole"
	UpdateRecursiveConcurrent = "UpdateRecursiveConcurrent"
	CreateSortList            = "CreateSortList"
	UpdateSortList            = "UpdateSortList"
	DeleteSortList            = "DeleteSortList"
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
		case StartDNS:
			var target pb.DNSStartReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.StartDNS(context.Background(), &target)
		case StopDNS:
			var target pb.DNSStopReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.StopDNS(context.Background(), &target)
		case CreateACL:
			var target pb.CreateACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateACL(context.Background(), &target)
		case UpdateACL:
			var target pb.UpdateACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateACL(context.Background(), &target)
		case DeleteACL:
			var target pb.DeleteACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteACL(context.Background(), &target)
		case CreateView:
			var target pb.CreateViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateView(context.Background(), &target)
		case UpdateView:
			var target pb.UpdateViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateView(context.Background(), &target)
		case DeleteView:
			var target pb.DeleteViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteView(context.Background(), &target)
		case CreateZone:
			var target pb.CreateZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateZone(context.Background(), &target)
		case UpdateZone:
			var target pb.UpdateZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateZone(context.Background(), &target)
		case DeleteZone:
			var target pb.DeleteZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteZone(context.Background(), &target)
		case CreateForwardZone:
			var target pb.CreateForwardZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateForwardZone(context.Background(), &target)
		case UpdateForwardZone:
			var target pb.UpdateForwardZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateForwardZone(context.Background(), &target)
		case DeleteForwardZone:
			var target pb.DeleteForwardZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteForwardZone(context.Background(), &target)
		case CreateRR:
			var target pb.CreateRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateRR(context.Background(), &target)
		case UpdateRR:
			var target pb.UpdateRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateRR(context.Background(), &target)
		case DeleteRR:
			var target pb.DeleteRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteRR(context.Background(), &target)
		case CreateForward:
			var target pb.CreateForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateForward(context.Background(), &target)
		case UpdateForward:
			var target pb.UpdateForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateForward(context.Background(), &target)
		case DeleteForward:
			var target pb.DeleteForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteForward(context.Background(), &target)
		case CreateRedirection:
			var target pb.CreateRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateRedirection(context.Background(), &target)
		case UpdateRedirection:
			var target pb.UpdateRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateRedirection(context.Background(), &target)
		case DeleteRedirection:
			var target pb.DeleteRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteRedirection(context.Background(), &target)
		case CreateDNS64:
			var target pb.CreateDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateDNS64(context.Background(), &target)
		case UpdateDNS64:
			var target pb.UpdateDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateDNS64(context.Background(), &target)
		case DeleteDNS64:
			var target pb.DeleteDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteDNS64(context.Background(), &target)
		case CreateIPBlackHole:
			var target pb.CreateIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateIPBlackHole(context.Background(), &target)
		case UpdateIPBlackHole:
			var target pb.UpdateIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateIPBlackHole(context.Background(), &target)
		case DeleteIPBlackHole:
			var target pb.DeleteIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteIPBlackHole(context.Background(), &target)
		case UpdateRecursiveConcurrent:
			var target pb.UpdateRecurConcuReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateRecursiveConcurrent(context.Background(), &target)
		case CreateSortList:
			var target pb.CreateSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.CreateSortList(context.Background(), &target)
		case UpdateSortList:
			var target pb.UpdateSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.UpdateSortList(context.Background(), &target)
		case DeleteSortList:
			var target pb.DeleteSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			cli.DeleteSortList(context.Background(), &target)

		}
	}
}
