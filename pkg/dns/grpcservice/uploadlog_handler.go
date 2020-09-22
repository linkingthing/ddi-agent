package grpcservice

import (
	"fmt"
	"os"
	"path"
	"time"

	ftpconn "github.com/jlaffaye/ftp"
	"github.com/zdnscloud/cement/log"

	"github.com/linkingthing/ddi-agent/pkg/kafkaproducer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

var queryLogName = "query.log"

func (handler *DNSHandler) UploadLog(req pb.UploadLogReq) error {
	if req.MasterNodeIp != handler.localip {
		return nil
	}

	conn, err := ftpconn.Dial(req.Address, ftpconn.DialWithTimeout(5*time.Second))
	if err != nil {
		if err = handler.sendUploadKafkaMsg(
			req.Id, "", pb.UploadLogResponse_STATUS_CONN_FAILED, err); err != nil {
			return err
		}
		return err
	}

	if err = conn.Login(req.User, req.Password); err != nil {
		if err = handler.sendUploadKafkaMsg(
			req.Id, "", pb.UploadLogResponse_STATUS_CONN_FAILED, err); err != nil {
			return err
		}
		return err
	}

	go handler.doUploadLog(conn, req.Id)
	return nil
}

func (handler *DNSHandler) doUploadLog(conn *ftpconn.ServerConn, id string) {
	if err := handler.sendUploadKafkaMsg(
		id, "", pb.UploadLogResponse_STATUS_TRANSPORTING, nil); err != nil {
		return
	}

	file, err := os.Open(path.Join(handler.dnsConfPath, queryLogName))
	defer file.Close()
	if err != nil {
		if err = handler.sendUploadKafkaMsg(
			id, "", pb.UploadLogResponse_STATUS_TRANSPORT_FAILED, err); err != nil {
			log.Errorf("doUploadLog sendUploadKafkaMsg id:%s failed:%s", id, err.Error())
		}
		return
	}

	fileName := fmt.Sprintf("%s-%s", queryLogName, time.Now().Format("2006-01-02_15:04:05"))
	if err = conn.Stor(fileName, file); err != nil {
		if err = handler.sendUploadKafkaMsg(
			id, fileName, pb.UploadLogResponse_STATUS_TRANSPORT_FAILED, err); err != nil {
			log.Errorf("doUploadLog sendUploadKafkaMsg id:%s failed:%s", id, err.Error())
		}
		return
	}

	if err = handler.sendUploadKafkaMsg(
		id, fileName, pb.UploadLogResponse_STATUS_TRANSPORT_DONE, nil); err != nil {
		log.Errorf("doUploadLog sendUploadKafkaMsg id:%s failed:%s", id, err.Error())
	}
	conn.Quit()
}

func (handler *DNSHandler) sendUploadKafkaMsg(
	id string, fileName string, status pb.UploadLogResponse_UploadStatus, err error) error {
	if err == nil {
		var finishTime string
		if status == pb.UploadLogResponse_STATUS_TRANSPORT_DONE {
			finishTime = time.Now().Format("2006-01-02 15:04:05")
		}
		return kafkaproducer.GetKafkaProducer().SendUploadMessage(&pb.UploadLogResponse{
			Id:         id,
			Status:     status,
			FileName:   fileName,
			FinishTime: finishTime,
		})
	}

	return kafkaproducer.GetKafkaProducer().SendUploadMessage(&pb.UploadLogResponse{
		Id:      id,
		Status:  status,
		Message: err.Error(),
	})
}
