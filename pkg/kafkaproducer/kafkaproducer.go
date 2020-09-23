package kafkaproducer

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/linkingthing/ddi-agent/pkg/proto"

	kg "github.com/segmentio/kafka-go"

	"github.com/linkingthing/ddi-agent/config"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

const (
	AgentEventTopic = "AgentEventTopic"
	UploadLogTopic  = "UploadLogTopic"
	AgentEvent      = "AgentEvent"
	UploadLogEvent  = "UploadLogEvent"
)

type KafkaProducer struct {
	agentWriter  *kg.Writer
	uploadWriter *kg.Writer
}

var globalKafkaProducer *KafkaProducer

func GetKafkaProducer() *KafkaProducer {
	return globalKafkaProducer
}

func Init(conf *config.AgentConfig) {
	globalKafkaProducer = &KafkaProducer{
		agentWriter: kg.NewWriter(kg.WriterConfig{
			Brokers:   conf.Kafka.Addr,
			Topic:     AgentEventTopic,
			BatchSize: 1,
		}),
		uploadWriter: kg.NewWriter(kg.WriterConfig{
			Brokers:   conf.Kafka.Addr,
			Topic:     UploadLogTopic,
			BatchSize: 1,
		}),
	}
}

func (producer *KafkaProducer) SendAgentEventMessage(node, nodeType string, header *pb.DDIRequestHead, req interface{}, ddiResponse *pb.DDIResponse, err error) error {
	if ddiResponse == nil {
		ddiResponse = &pb.DDIResponse{}
	}
	ddiResponse.Header = header
	if !ddiResponse.Succeed && err != nil {
		s, ok := status.FromError(err)
		if !ok {
			ddiResponse.ErrorMessage = err.Error()
		} else {
			ddiResponse.ErrorMessage = s.Message()
		}
	}
	reqData, _ := json.Marshal(req)
	ddiResponse.CmdMessage = string(reqData)
	ddiResponse.Node = node
	ddiResponse.NodeType = nodeType
	ddiResponse.OperationTime = time.Now().Format("2006-01-02 15:04:05")

	data, err := proto.Marshal(ddiResponse)
	if err != nil {
		return fmt.Errorf("kafka SendAgentEventMessage Marshal failed: %s", err.Error())
	}

	return producer.agentWriter.WriteMessages(context.Background(), kg.Message{Key: []byte(AgentEvent), Value: data})
}

func (producer *KafkaProducer) SendUploadMessage(m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return fmt.Errorf("kafka SendUploadMessage Marshal failed: %s", err.Error())
	}

	return producer.uploadWriter.WriteMessages(context.Background(), kg.Message{Key: []byte(UploadLogEvent), Value: data})
}
