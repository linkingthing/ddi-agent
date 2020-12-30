package kafkaconsumer

import (
	"context"

	"github.com/golang/protobuf/proto"
	kg "github.com/segmentio/kafka-go"
	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	"github.com/linkingthing/ddi-agent/config"
	"github.com/linkingthing/ddi-agent/pkg/kafkaproducer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

const (
	StartDNS               = "StartDNS"
	StopDNS                = "StopDNS"
	CreateACL              = "CreateACL"
	UpdateACL              = "UpdateACL"
	DeleteACL              = "DeleteACL"
	CreateView             = "CreateView"
	UpdateView             = "UpdateView"
	DeleteView             = "DeleteView"
	CreateZone             = "CreateZone"
	UpdateZone             = "UpdateZone"
	DeleteZone             = "DeleteZone"
	CreateForwardZone      = "CreateForwardZone"
	UpdateForwardZone      = "UpdateForwardZone"
	DeleteForwardZone      = "DeleteForwardZone"
	FlushForwardZone       = "FlushForwardZone"
	BatchUpdateForwardZone = "BatchUpdateForwardZone"
	CreateRR               = "CreateRR"
	UpdateRR               = "UpdateRR"
	DeleteRR               = "DeleteRR"
	UpdateForward          = "UpdateForward"
	CreateRedirection      = "CreateRedirection"
	UpdateRedirection      = "UpdateRedirection"
	DeleteRedirection      = "DeleteRedirection"
	CreateUrlRedirect      = "CreateUrlRedirect"
	UpdateUrlRedirect      = "UpdateUrlRedirect"
	DeleteUrlRedirect      = "DeleteUrlRedirect"
	UpdateGlobalConfig     = "UpdateGlobalConfig"
	UploadLog              = "UploadLog"
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
		Brokers:  conf.Kafka.Addr,
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
			var req pb.DNSStartReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal StartDNS request failed: %s", err.Error())
			} else {
				if _, err := cli.StartDNS(context.Background(), &req); err != nil {
					log.Errorf("grpc service exec StartDNS failed: %s", err.Error())
				}
			}
		case StopDNS:
			var req pb.DNSStopReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal StopDNS request failed: %s", err.Error())
			} else {
				if _, err := cli.StopDNS(context.Background(), &req); err != nil {
					log.Errorf("grpc service exec StopDNS failed: %s", err.Error())
				}
			}
		case CreateACL:
			var req pb.CreateACLReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateACL request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateACL(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateACL failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateACL:
			var req pb.UpdateACLReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateACL request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateACL(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateACL failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteACL:
			var req pb.DeleteACLReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteACL request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteACL(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteACL failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateView:
			var req pb.CreateViewReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateView request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateView(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateView failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateView:
			var req pb.UpdateViewReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateView request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateView(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateView failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteView:
			var req pb.DeleteViewReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteView request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteView(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteView failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateZone:
			var req pb.CreateZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateZone:
			var req pb.UpdateZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteZone:
			var req pb.DeleteZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateForwardZone:
			var req pb.CreateForwardZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateForwardZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateForwardZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateForwardZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateForwardZone:
			var req pb.UpdateForwardZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateForwardZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateForwardZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateForwardZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteForwardZone:
			var req pb.DeleteForwardZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteForwardZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteForwardZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteForwardZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case FlushForwardZone:
			var target pb.FlushForwardZoneReq
			if err := proto.Unmarshal(message.Value, &target); err != nil {
			}
			if _, err := cli.FlushForwardZone(context.Background(), &target); err != nil {
				log.Errorf("grpc service exec FlushForwardZone failed: %s", err.Error())
			}
		case CreateRR:
			var req pb.CreateRRReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateRR request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateRR(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateRR failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateRR:
			var req pb.UpdateRRReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateRR request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateRR(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateRR failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteRR:
			var req pb.DeleteRRReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteRR request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteRR(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteRR failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateForward:
			var req pb.UpdateForwardReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateForward request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateForward(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateForward failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateRedirection:
			var req pb.CreateRedirectionReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateRedirection request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateRedirection(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateRedirection failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateRedirection:
			var req pb.UpdateRedirectionReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateRedirection request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateRedirection(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateRedirection failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteRedirection:
			var req pb.DeleteRedirectionReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteRedirection request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteRedirection(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteRedirection failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateUrlRedirect:
			var req pb.CreateUrlRedirectReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateUrlRedirect request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateUrlRedirect(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateUrlRedirect failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateUrlRedirect:
			var req pb.UpdateUrlRedirectReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateUrlRedirect request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateUrlRedirect(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateUrlRedirect failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteUrlRedirect:
			var req pb.DeleteUrlRedirectReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteUrlRedirect request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteUrlRedirect(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteUrlRedirect failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateGlobalConfig:
			var req pb.UpdateGlobalConfigReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateGlobalConfig request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateGlobalConfig(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateGlobalConfig failed:%s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", req.Header, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UploadLog:
			var req pb.UploadLogReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("message %s Unmarshal failed:%s", message.Key, err.Error())
			} else {
				if _, err := cli.UploadLog(context.Background(), &req); err != nil {
					log.Errorf("grpc service exec FtpTransport failed:%s", err.Error())
				}
			}
		}
	}
}
