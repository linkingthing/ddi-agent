package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

const MethodPost = "POST"

type DHCPCmdRequest struct {
	Command   string      `json:"command"`
	Services  []string    `json:"service"`
	Arguments interface{} `json:"arguments"`
}

type DHCPCmdResponse struct {
	Result    int         `json:"result"`
	Text      string      `json:"text"`
	Arguments interface{} `json:"arguments"`
}

func SendHttpRequestToDHCP(cli *http.Client, dhcpCmdAddr string, req *DHCPCmdRequest) ([]DHCPCmdResponse, error) {
	var resp []DHCPCmdResponse
	err := sendHttpRequest(cli, MethodPost, "http://"+dhcpCmdAddr, req, &resp)
	return resp, err
}

func sendHttpRequest(cli *http.Client, httpMethod, url string, req, resp interface{}) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal request failed: %s", err.Error())
	}

	httpReq, err := http.NewRequest(httpMethod, url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("new http request failed: %s", err.Error())
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := cli.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send http request failed: %s", err.Error())
	}

	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read http response body failed: %s", err.Error())
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("unmarshal http response failed: %s", err.Error())
	}

	return nil
}
