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
			var req pb.CreateAclReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateACL request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateAcl(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateACL failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case BatchCreateACL:
			var req pb.BatchCreateAclReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal BatchCreateAclReq request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.BatchCreateAcl(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec BatchCreateAclReq failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateACL:
			var req pb.UpdateAclReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateACL request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateAcl(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateACL failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteACL:
			var req pb.DeleteAclReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteACL request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteAcl(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteACL failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateAuthZone:
			var req pb.CreateAuthZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateAuthZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateAuthZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateAuthZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateAuthZone:
			var req pb.UpdateAuthZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateAuthZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteAuthZone:
			var req pb.DeleteAuthZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteZone request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteAuthZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteZone failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateAuthZoneAuthRRs:
			var req pb.CreateAuthZoneAuthRRsReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateAuthZoneAuthRRs request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateAuthZoneAuthRRs(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateAuthZoneAuthRRs failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateAuthZoneAXFR:
			var req pb.UpdateAuthZoneAXFRReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateAuthZoneAXFR request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateAuthZoneAXFR(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateAuthZoneAXFR failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateAuthZoneIXFR:
			var req pb.UpdateAuthZoneIXFRReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateAuthZoneIXFR request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateAuthZoneIXFR(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateAuthZoneIXFR failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateAuthRR:
			var req pb.CreateAuthRRReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateAuthRR request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateAuthRR(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateAuthRR failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateAuthRR:
			var req pb.UpdateAuthRRReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateAuthRR request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateAuthRR(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateAuthRR failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteAuthRR:
			var req pb.DeleteAuthRRReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteAuthRR request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteAuthRR(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteAuthRR failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case BatchCreateAuthRRs:
			var req pb.BatchCreateAuthRRsReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal BatchCreateAuthRRs request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.BatchCreateAuthRRs(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec BatchCreateAuthRRs failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case CreateNginxProxy:
			var req pb.CreateNginxProxyReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal CreateNginxProxy request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.CreateNginxProxy(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec CreateNginxProxy failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case UpdateNginxProxy:
			var req pb.UpdateNginxProxyReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal UpdateNginxProxy request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.UpdateNginxProxy(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec UpdateNginxProxy failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		case DeleteNginxProxy:
			var req pb.DeleteNginxProxyReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal DeleteNginxProxy request failed: %s", err.Error())
			} else {
				ddiResponse, err := cli.DeleteNginxProxy(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec DeleteNginxProxy failed: %s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
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
		case FlushForwardZone:
			var req pb.FlushForwardZoneReq
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Errorf("unmarshal FlushForwardZone failed:%s", err.Error())
			} else {
				ddiResponse, err := cli.FlushForwardZone(context.Background(), &req)
				if err != nil {
					log.Errorf("grpc service exec FlushForwardZone failed:%s", err.Error())
				}
				if err := kafkaproducer.GetKafkaProducer().SendAgentEventMessage(
					conf.Server.IP, "dns", message.Key, &req, ddiResponse, err); err != nil {
					log.Errorf("SendAgentEventMessage ddiResponse key:%s failed:%s", message.Key, err.Error())
				}
			}
		}
	}
}
