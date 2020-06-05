package kafkaconsumer

import (
	"context"

	"github.com/golang/protobuf/proto"
	kg "github.com/segmentio/kafka-go"
	"google.golang.org/grpc"

	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-agent/config"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
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

func Run(conn *grpc.ClientConn, conf *config.AgentConfig) {
	if conf.DNS.Enabled == false {
		return
	}

	cli := pb.NewAgentManagerClient(conn)
	kafkaReader := kg.NewReader(kg.ReaderConfig{
		Brokers: []string{conf.Kafka.Addr},
		Topic:   DNSTopic,
	})

	for {
		message, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Errorf("read dns message from kafka failed: %s", err.Error())
			return
		}

		switch string(message.Key) {
		case StartDNS:
			var target pb.DNSStartReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.StartDNS(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec StartDNS failed: %s", err.Error())
			}
		case StopDNS:
			var target pb.DNSStopReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.StopDNS(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec StopDNS failed: %s", err.Error())
			}
		case CreateACL:
			var target pb.CreateACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateACL(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateACL failed: %s", err.Error())
			}
		case UpdateACL:
			var target pb.UpdateACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateACL(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateACL failed: %s", err.Error())
			}
		case DeleteACL:
			var target pb.DeleteACLReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteACL(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteACL failed: %s", err.Error())
			}
		case CreateView:
			var target pb.CreateViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateView(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateView failed: %s", err.Error())
			}
		case UpdateView:
			var target pb.UpdateViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateView(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateView failed: %s", err.Error())
			}
		case DeleteView:
			var target pb.DeleteViewReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteView(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteView failed: %s", err.Error())
			}
		case CreateZone:
			var target pb.CreateZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateZone failed: %s", err.Error())
			}
		case UpdateZone:
			var target pb.UpdateZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateZone failed: %s", err.Error())
			}
		case DeleteZone:
			var target pb.DeleteZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteZone failed: %s", err.Error())
			}
		case CreateForwardZone:
			var target pb.CreateForwardZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateForwardZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateForwardZone failed: %s", err.Error())
			}
		case UpdateForwardZone:
			var target pb.UpdateForwardZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateForwardZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateForwardZone failed: %s", err.Error())
			}
		case DeleteForwardZone:
			var target pb.DeleteForwardZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteForwardZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteForwardZone failed: %s", err.Error())
			}
		case CreateRR:
			var target pb.CreateRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateRR(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateRR failed: %s", err.Error())
			}
		case UpdateRR:
			var target pb.UpdateRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateRR(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateRR failed: %s", err.Error())
			}
		case DeleteRR:
			var target pb.DeleteRRReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteRR(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteRR failed: %s", err.Error())
			}
		case CreateForward:
			var target pb.CreateForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateForward(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateForward failed: %s", err.Error())
			}
		case UpdateForward:
			var target pb.UpdateForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateForward(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateForward failed: %s", err.Error())
			}
		case DeleteForward:
			var target pb.DeleteForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteForward(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteForward failed: %s", err.Error())
			}
		case CreateRedirection:
			var target pb.CreateRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateRedirection(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateRedirection failed: %s", err.Error())
			}
		case UpdateRedirection:
			var target pb.UpdateRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateRedirection(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateRedirection failed: %s", err.Error())
			}
		case DeleteRedirection:
			var target pb.DeleteRedirectionReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteRedirection(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteRedirection failed: %s", err.Error())
			}
		case CreateDNS64:
			var target pb.CreateDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateDNS64(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateDNS64 failed: %s", err.Error())
			}
		case UpdateDNS64:
			var target pb.UpdateDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateDNS64(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateDNS64 failed: %s", err.Error())
			}
		case DeleteDNS64:
			var target pb.DeleteDNS64Req
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteDNS64(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteDNS64 failed: %s", err.Error())
			}
		case CreateIPBlackHole:
			var target pb.CreateIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateIPBlackHole(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateIPBlackHole failed: %s", err.Error())
			}
		case UpdateIPBlackHole:
			var target pb.UpdateIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateIPBlackHole(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateIPBlackHole failed: %s", err.Error())
			}
		case DeleteIPBlackHole:
			var target pb.DeleteIPBlackHoleReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteIPBlackHole(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteIPBlackHole failed: %s", err.Error())
			}
		case UpdateRecursiveConcurrent:
			var target pb.UpdateRecurConcuReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateRecursiveConcurrent(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateRecursiveConcurrent failed: %s", err.Error())
			}
		case CreateSortList:
			var target pb.CreateSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateSortList(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateSortList failed: %s", err.Error())
			}
		case UpdateSortList:
			var target pb.UpdateSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateSortList(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateSortList failed: %s", err.Error())
			}
		case DeleteSortList:
			var target pb.DeleteSortListReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteSortList(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteSortList failed: %s", err.Error())
			}
		}
	}
}
