package kafkaconsumer

import (
	"context"

	"github.com/golang/protobuf/proto"
	"github.com/segmentio/kafka-go"
	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	"github.com/linkingthing/ddi-agent/config"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

func Run(conn *grpc.ClientConn, conf *config.AgentConfig) {
	if conf.DHCP.Enabled == false {
		return
	}

	run(pb.NewDHCPManagerClient(conn), kafka.NewReader(kafka.ReaderConfig{
		Brokers: []string{conf.Kafka.Addr},
		Topic:   Topic,
	}))
}

func run(cli pb.DHCPManagerClient, kafkaConsumer *kafka.Reader) {
	for {
		message, err := kafkaConsumer.ReadMessage(context.Background())
		if err != nil {
			log.Warnf("read dhcp message from kafka failed: %s", err.Error())
			continue
		}

		switch string(message.Key) {
		case CreateSubnet4:
			var req pb.CreateSubnet4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal create subnet4 request failed: %s", err.Error())
			} else {
				if _, err := cli.CreateSubnet4(context.Background(), &req); err != nil {
					log.Warnf("create subnet4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdateSubnet4:
			var req pb.UpdateSubnet4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update subnet4 request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdateSubnet4(context.Background(), &req); err != nil {
					log.Warnf("update subnet4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case DeleteSubnet4:
			var req pb.DeleteSubnet4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal delete subnet4 request failed: %s", err.Error())
			} else {
				if _, err := cli.DeleteSubnet4(context.Background(), &req); err != nil {
					log.Warnf("delete subnet4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case CreateSubnet6:
			var req pb.CreateSubnet6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal create subnet6 request failed: %s", err.Error())
			} else {
				if _, err := cli.CreateSubnet6(context.Background(), &req); err != nil {
					log.Warnf("create subnet6 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdateSubnet6:
			var req pb.UpdateSubnet6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update subnet6 request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdateSubnet6(context.Background(), &req); err != nil {
					log.Warnf("update subnet6 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case DeleteSubnet6:
			var req pb.DeleteSubnet6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal delete subnet6 request failed: %s", err.Error())
			} else {
				if _, err := cli.DeleteSubnet6(context.Background(), &req); err != nil {
					log.Warnf("delete subnet6 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case CreatePool4:
			var req pb.CreatePool4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal create pool4 request failed: %s", err.Error())
			} else {
				if _, err := cli.CreatePool4(context.Background(), &req); err != nil {
					log.Warnf("create pool4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdatePool4:
			var req pb.UpdatePool4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update pool4 request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdatePool4(context.Background(), &req); err != nil {
					log.Warnf("update pool4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case DeletePool4:
			var req pb.DeletePool4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal delete pool4 request failed: %s", err.Error())
			} else {
				if _, err := cli.DeletePool4(context.Background(), &req); err != nil {
					log.Warnf("delete pool4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case CreatePool6:
			var req pb.CreatePool6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal create pool6 request failed: %s", err.Error())
			} else {
				if _, err := cli.CreatePool6(context.Background(), &req); err != nil {
					log.Warnf("create pool6 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdatePool6:
			var req pb.UpdatePool6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update pool6 request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdatePool6(context.Background(), &req); err != nil {
					log.Warnf("update pool6 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case DeletePool6:
			var req pb.DeletePool6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal delete pool6 request failed: %s", err.Error())
			} else {
				if _, err := cli.DeletePool6(context.Background(), &req); err != nil {
					log.Warnf("delete pool6 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case CreatePDPool:
			var req pb.CreatePDPoolRequest
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal create pd-pool request failed: %s", err.Error())
			} else {
				if _, err := cli.CreatePDPool(context.Background(), &req); err != nil {
					log.Warnf("create pd-pool with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdatePDPool:
			var req pb.UpdatePDPoolRequest
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update pd-pool request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdatePDPool(context.Background(), &req); err != nil {
					log.Warnf("update pd-pool with req %s failed: %s", req.String(), err.Error())
				}
			}
		case DeletePDPool:
			var req pb.DeletePDPoolRequest
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal delete pd-pool request failed: %s", err.Error())
			} else {
				if _, err := cli.DeletePDPool(context.Background(), &req); err != nil {
					log.Warnf("delete pd-pool with req %s failed: %s", req.String(), err.Error())
				}
			}
		case CreateReservation4:
			var req pb.CreateReservation4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal create reservation4 request failed: %s", err.Error())
			} else {
				if _, err := cli.CreateReservation4(context.Background(), &req); err != nil {
					log.Warnf("create reservation4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdateReservation4:
			var req pb.UpdateReservation4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update reservation4 request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdateReservation4(context.Background(), &req); err != nil {
					log.Warnf("update reservation4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case DeleteReservation4:
			var req pb.DeleteReservation4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal delete reservation4 request failed: %s", err.Error())
			} else {
				if _, err := cli.DeleteReservation4(context.Background(), &req); err != nil {
					log.Warnf("delete reservation4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case CreateReservation6:
			var req pb.CreateReservation6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal create reservation6 request failed: %s", err.Error())
			} else {
				if _, err := cli.CreateReservation6(context.Background(), &req); err != nil {
					log.Warnf("create reservation6 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdateReservation6:
			var req pb.UpdateReservation6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update reservation6 request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdateReservation6(context.Background(), &req); err != nil {
					log.Warnf("update reservation4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case DeleteReservation6:
			var req pb.DeleteReservation6Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal delete reservation4 request failed: %s", err.Error())
			} else {
				if _, err := cli.DeleteReservation6(context.Background(), &req); err != nil {
					log.Warnf("delete reservation4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case CreateClientClass4:
			var req pb.CreateClientClass4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal create clientclass4 request failed: %s", err.Error())
			} else {
				if _, err := cli.CreateClientClass4(context.Background(), &req); err != nil {
					log.Warnf("create clientclass4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdateClientClass4:
			var req pb.UpdateClientClass4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update clientclass4 request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdateClientClass4(context.Background(), &req); err != nil {
					log.Warnf("update clientclass4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case DeleteClientClass4:
			var req pb.DeleteClientClass4Request
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal delete clientclass4 request failed: %s", err.Error())
			} else {
				if _, err := cli.DeleteClientClass4(context.Background(), &req); err != nil {
					log.Warnf("delete clientclass4 with req %s failed: %s", req.String(), err.Error())
				}
			}
		case UpdateGlobalConfig:
			var req pb.UpdateGlobalConfigRequest
			if err := proto.Unmarshal(message.Value, &req); err != nil {
				log.Warnf("unmarshal update dhcp global config request failed: %s", err.Error())
			} else {
				if _, err := cli.UpdateGlobalConfig(context.Background(), &req); err != nil {
					log.Warnf("update dhcp global config with req %s failed: %s", req.String(), err.Error())
				}
			}
		}
	}
}
