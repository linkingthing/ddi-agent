package kafkaproducer

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	kg "github.com/segmentio/kafka-go"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/linkingthing/ddi-agent/config"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

const (
	AgentEventTopic = "AgentEventTopic"
	UploadLogTopic  = "UploadLogTopic"
	AgentEvent      = "AgentEvent"
	UploadLogEvent  = "UploadLogEvent"
)

const (
	CmdSplitSymbol = "_"
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

func formatCmd(cmd []byte) (string, string) {
	cmds := strings.Split(string(cmd), CmdSplitSymbol)
	if len(cmds) > 1 {
		return cmds[0], cmds[1]
	}

	return "", ""
}

func (producer *KafkaProducer) SendAgentEventMessage(
	node, nodeType string, cmd []byte, req interface{}, ddiResponse *pb.DDIResponse, err error) error {
	if ddiResponse == nil {
		ddiResponse = &pb.DDIResponse{}
	}
	if !ddiResponse.Succeed && err != nil {
		s, ok := status.FromError(err)
		if !ok {
			ddiResponse.ErrorMessage = err.Error()
		} else {
			ddiResponse.ErrorMessage = s.Message()
		}
	}
	method, resource := formatCmd(cmd)
	ddiResponse.Method = method
	ddiResponse.Resource = resource
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
