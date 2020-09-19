package kafkaconsumer

import (
	"context"

	"github.com/golang/protobuf/proto"
	kg "github.com/segmentio/kafka-go"
	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

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
	UpdateForwardZone         = "UpdateForwardZone"
	CreateRR                  = "CreateRR"
	UpdateRR                  = "UpdateRR"
	DeleteRR                  = "DeleteRR"
	UpdateRRsByZone           = "UpdateRRsByZone"
	UpdateForward             = "UpdateForward"
	CreateRedirection         = "CreateRedirection"
	UpdateRedirection         = "UpdateRedirection"
	DeleteRedirection         = "DeleteRedirection"
	CreateIPBlackHole         = "CreateIPBlackHole"
	UpdateIPBlackHole         = "UpdateIPBlackHole"
	DeleteIPBlackHole         = "DeleteIPBlackHole"
	UpdateRecursiveConcurrent = "UpdateRecursiveConcurrent"
	CreateUrlRedirect         = "CreateUrlRedirect"
	UpdateUrlRedirect         = "UpdateUrlRedirect"
	DeleteUrlRedirect         = "DeleteUrlRedirect"
	UpdateGlobalConfig        = "UpdateGlobalConfig"
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
		Brokers:  []string{conf.Kafka.Addr},
		Topic:    DNSTopic,
		GroupID:  conf.DNS.GroupID,
		MinBytes: 10,
		MaxBytes: 10e6,
	})

	for {
		message, err := kafkaReader.ReadMessage(context.Background())
		if err != nil {
			log.Warnf("read dns message from kafka failed: %s", err.Error())
			continue
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
		case UpdateForwardZone:
			var target pb.UpdateForwardZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateForwardZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateForwardZone failed: %s", err.Error())
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
		case UpdateRRsByZone:
			var target pb.UpdateRRsByZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateRRsByZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateRRsByZone failed: %s", err.Error())
			}
		case UpdateForward:
			var target pb.UpdateForwardReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateForward(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateForward failed: %s", err.Error())
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
		case CreateUrlRedirect:
			var target pb.CreateUrlRedirectReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.CreateUrlRedirect(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec CreateUrlRedirect failed: %s", err.Error())
			}
		case UpdateUrlRedirect:
			var target pb.UpdateUrlRedirectReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateUrlRedirect(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateUrlRedirect failed: %s", err.Error())
			}
		case DeleteUrlRedirect:
			var target pb.DeleteUrlRedirectReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.DeleteUrlRedirect(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec DeleteUrlRedirect failed: %s", err.Error())
			}
		case UpdateGlobalConfig:
			var target pb.UpdateGlobalConfigReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.UpdateGlobalConfig(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec UpdateGlobalConfig failed:%s", err.Error())
			}
		}
	}
}
