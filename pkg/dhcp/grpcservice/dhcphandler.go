package grpcservice

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/zdnscloud/cement/log"
	restdb "github.com/zdnscloud/gorest/db"

	"github.com/linkingthing/ddi-agent/config"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
)

const (
	DHCP4ConfigFileName    = "kea-dhcp4.conf"
	DHCP6ConfigFileName    = "kea-dhcp6.conf"
	StartDHCPCmd           = "keactrl start"
	StopDHCPCmd            = "keactrl stop"
	DHCP4Name              = "dhcp4"
	DHCP6Name              = "dhcp6"
	DHCPAgentName          = "ctrl-agent"
	Option4DNSServers      = "domain-name-servers"
	Option6DNSServers      = "dns-servers"
	Option4Routers         = "routers"
	DHCPCommandConfigSet   = "config-set"
	DHCPCommandConfigWrite = "config-write"
	PostgresqlConnStr      = "user=%s password=%s host=localhost port=%d database=%s sslmode=disable pool_max_conns=10"
	MethodPost             = "POST"
	TableLease4            = "lease4"
)

type DHCPHandler struct {
	cmdAddr    string
	conf       *DHCPConfig
	lock       sync.RWMutex
	db         restdb.ResourceStore
	httpClient *http.Client
}

type DHCPConfig struct {
	dhcp4Conf *DHCP4Config
	dhcp6Conf *DHCP6Config
}

func newDHCPHandler(conf *config.AgentConfig) (*DHCPHandler, error) {
	meta, err := restdb.NewResourceMeta(nil)
	if err != nil {
		return nil, err
	}

	db, err := restdb.NewRStore(fmt.Sprintf(PostgresqlConnStr, conf.DHCP.DB.User, conf.DHCP.DB.Password,
		conf.DHCP.DB.Port, conf.DHCP.DB.Name), meta)
	if err != nil {
		return nil, err
	}

	handler := &DHCPHandler{cmdAddr: conf.DHCP.CmdAddr, db: db, httpClient: &http.Client{}}
	if err := handler.loadDHCPConfig(conf.DHCP.ConfigDir); err != nil {
		return nil, err
	}

	if err := handler.startDHCP(); err != nil {
		return nil, err
	}

	go handler.monitor()
	return handler, nil
}

func (h *DHCPHandler) loadDHCPConfig(configDir string) error {
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.Mkdir(configDir, os.ModePerm); err != nil {
			return fmt.Errorf("create config dir %s failed: %s", configDir, err.Error())
		}
	}

	var dhcp4Conf DHCP4Config
	dhcp4ConfPath := path.Join(configDir, DHCP4ConfigFileName)
	if _, err := os.Stat(dhcp4ConfPath); os.IsNotExist(err) {
		dhcp4Conf = genDefaultDHCP4Config()
		if err := genDefaultDHCPConfigFile(dhcp4ConfPath, &dhcp4Conf); err != nil {
			return err
		}
	} else {
		if err := parseJsonConfig(&dhcp4Conf, dhcp4ConfPath); err != nil {
			return fmt.Errorf("load dhcp4 config failed: %s", err.Error())
		}
	}

	var dhcp6Conf DHCP6Config
	dhcp6ConfPath := path.Join(configDir, DHCP6ConfigFileName)
	if _, err := os.Stat(dhcp6ConfPath); os.IsNotExist(err) {
		dhcp6Conf = genDefaultDHCP6Config()
		if err := genDefaultDHCPConfigFile(dhcp6ConfPath, &dhcp6Conf); err != nil {
			return err
		}
	} else {
		if err := parseJsonConfig(&dhcp6Conf, dhcp6ConfPath); err != nil {
			return fmt.Errorf("load dhcp6 config failed: %s", err.Error())
		}
	}

	dhcp4Conf.Path = dhcp4ConfPath
	dhcp6Conf.Path = dhcp6ConfPath
	h.conf = &DHCPConfig{
		dhcp4Conf: &dhcp4Conf,
		dhcp6Conf: &dhcp6Conf,
	}
	return nil
}

func genDefaultDHCPConfigFile(filePath string, fileContent interface{}) error {
	content, err := json.MarshalIndent(fileContent, "", "\t")
	if err != nil {
		return fmt.Errorf("marshal file %s content failed: %s", filePath, err.Error())
	}

	return ioutil.WriteFile(filePath, content, 0644)
}

func parseJsonConfig(conf interface{}, filepath string) error {
	data, err := ioutil.ReadFile(filepath)
	if err != nil {
		return err
	}

	return json.Unmarshal(data, conf)
}

func (h *DHCPHandler) startDHCP() error {
	return runCommand(StartDHCPCmd)
}

func (h *DHCPHandler) stopDHCP() error {
	return runCommand(StopDHCPCmd)
}

func runCommand(cmdline string) error {
	cmd := exec.Command("bash", "-c", cmdline)
	return cmd.Run()
}

func (h *DHCPHandler) monitor() {
	for {
		if checkProcessExists(DHCP4Name) == false ||
			checkProcessExists(DHCP6Name) == false ||
			checkProcessExists(DHCPAgentName) == false {
			if err := h.startDHCP(); err != nil {
				log.Warnf("start dhcp failed: %s", err.Error())
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func checkProcessExists(processName string) bool {
	out, _ := exec.Command("bash", "-c", "ps -ef | grep "+processName+" | grep -v grep").Output()
	return len(out) > 0
}

func (h *DHCPHandler) CreateSubnet4(req *pb.CreateSubnet4Request) error {
	h.lock.Lock()
	h.conf.dhcp4Conf.DHCP4.Subnet4s = append(h.conf.dhcp4Conf.DHCP4.Subnet4s, Subnet4{
		ID:               req.GetId(),
		Subent:           req.GetIpnet(),
		ClientClass:      req.GetClientClass(),
		ValidLifetime:    req.GetValidLifetime(),
		MaxValidLifetime: req.GetMaxValidLifetime(),
		MinValidLifetime: req.GetMinValidLifetime(),
		OptionDatas:      genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), req.GetRouters()),
		Relay:            RelayAgent{IPAddresses: req.GetRelayAgentAddresses()},
	})
	h.lock.Unlock()

	return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
}

func genDHCPOptionDatas(optionNameDNS string, domainServers []string, routers []string) []OptionData {
	var options []OptionData
	if len(domainServers) != 0 {
		options = append(options, OptionData{
			Name: optionNameDNS,
			Data: strings.Join(domainServers, ", "),
		})
	}

	if len(routers) != 0 {
		options = append(options, OptionData{
			Name: Option4Routers,
			Data: strings.Join(routers, ", "),
		})
	}

	return options
}

func (h *DHCPHandler) reconfig(services []string, configPath string, conf interface{}) error {
	if err := h.setDHCPConfigToMemory(services, conf); err != nil {
		return err
	}

	return h.writeDHCPConfigToFile(services, configPath)
}

func (h *DHCPHandler) setDHCPConfigToMemory(services []string, conf interface{}) error {
	var resp []DHCPCmdResponse
	if err := h.sendHttpRequest(&DHCPCmdRequest{
		Command:   DHCPCommandConfigSet,
		Services:  services,
		Arguments: conf,
	}, &resp); err != nil {
		return err
	}

	if len(resp) != 0 && resp[0].Result != 0 {
		return fmt.Errorf("set %v config failed: %s", services, resp[0].Text)
	}

	return nil
}

func (h *DHCPHandler) writeDHCPConfigToFile(services []string, configPath string) error {
	var resp []DHCPCmdResponse
	if err := h.sendHttpRequest(&DHCPCmdRequest{
		Command:  DHCPCommandConfigWrite,
		Services: services,
		Arguments: map[string]interface{}{
			"filename": configPath,
		},
	}, &resp); err != nil {
		return err
	}

	if len(resp) != 0 && resp[0].Result != 0 {
		return fmt.Errorf("write %v config failed: %s", services, resp[0].Text)
	}

	return nil
}

func (h *DHCPHandler) sendHttpRequest(req *DHCPCmdRequest, resp interface{}) error {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("marshal command %s failed: %s", req.Command, err.Error())
	}

	httpReq, err := http.NewRequest(MethodPost, "http://"+h.cmdAddr, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("new http request with command %s failed: %s", req.Command, err.Error())
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpResp, err := h.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("send http request with command %s failed: %s", req.Command, err.Error())
	}

	defer httpResp.Body.Close()
	body, err := ioutil.ReadAll(httpResp.Body)
	if err != nil {
		return fmt.Errorf("read http response body with command %s failed: %s", req.Command, err.Error())
	}

	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("unmarshal http response with command %s failed: %s", req.Command, err.Error())
	}

	return nil
}

func (h *DHCPHandler) UpdateSubnet4(req *pb.UpdateSubnet4Request) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetId() {
			h.conf.dhcp4Conf.DHCP4.Subnet4s[i].ClientClass = req.GetClientClass()
			h.conf.dhcp4Conf.DHCP4.Subnet4s[i].ValidLifetime = req.GetValidLifetime()
			h.conf.dhcp4Conf.DHCP4.Subnet4s[i].MaxValidLifetime = req.GetMaxValidLifetime()
			h.conf.dhcp4Conf.DHCP4.Subnet4s[i].MinValidLifetime = req.GetMinValidLifetime()
			h.conf.dhcp4Conf.DHCP4.Subnet4s[i].OptionDatas = genDHCPOptionDatas(
				Option4DNSServers, req.GetDomainServers(), req.GetRouters())
			h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Relay = RelayAgent{IPAddresses: req.GetRelayAgentAddresses()}
			exists = true
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
	} else {
		return fmt.Errorf("no found subnet4 %s", req.GetId())
	}
}

func (h *DHCPHandler) DeleteSubnet4(req *pb.DeleteSubnet4Request) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetId() {
			h.conf.dhcp4Conf.DHCP4.Subnet4s = append(h.conf.dhcp4Conf.DHCP4.Subnet4s[:i], h.conf.dhcp4Conf.DHCP4.Subnet4s[i+1:]...)
			exists = true
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
	} else {
		return fmt.Errorf("no found subnet4 %s", req.GetId())
	}
}

func (h *DHCPHandler) CreateSubnet6(req *pb.CreateSubnet6Request) error {
	h.lock.Lock()
	h.conf.dhcp6Conf.DHCP6.Subnet6s = append(h.conf.dhcp6Conf.DHCP6.Subnet6s, Subnet6{
		ID:               req.GetId(),
		Subent:           req.GetIpnet(),
		ClientClass:      req.GetClientClass(),
		ValidLifetime:    req.GetValidLifetime(),
		MaxValidLifetime: req.GetMaxValidLifetime(),
		MinValidLifetime: req.GetMinValidLifetime(),
		OptionDatas:      genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil),
		Relay:            RelayAgent{IPAddresses: req.GetRelayAgentAddresses()},
	})

	h.lock.Unlock()

	return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
}

func (h *DHCPHandler) UpdateSubnet6(req *pb.UpdateSubnet6Request) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetId() {
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].ClientClass = req.GetClientClass()
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].ValidLifetime = req.GetValidLifetime()
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].MaxValidLifetime = req.GetMaxValidLifetime()
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].MinValidLifetime = req.GetMinValidLifetime()
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].OptionDatas = genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil)
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Relay = RelayAgent{IPAddresses: req.GetRelayAgentAddresses()}
			exists = true
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
	} else {
		return fmt.Errorf("no found subnet6 %s", req.GetId())
	}
}

func (h *DHCPHandler) DeleteSubnet6(req *pb.DeleteSubnet6Request) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetId() {
			h.conf.dhcp6Conf.DHCP6.Subnet6s = append(h.conf.dhcp6Conf.DHCP6.Subnet6s[:i], h.conf.dhcp6Conf.DHCP6.Subnet6s[i+1:]...)
			exists = true
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
	} else {
		return fmt.Errorf("no found subnet6 %s", req.GetId())
	}
}

func (h *DHCPHandler) CreatePool4(req *pb.CreatePool4Request) error {
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Pools = append(h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Pools, Pool{
				Pool:        genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress()),
				ClientClass: req.GetClientClass(),
				OptionDatas: genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), req.GetRouters()),
			})
			break
		}
	}
	h.lock.Unlock()
	return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
}

func genPoolByBeginAndEnd(begin, end string) string {
	return begin + " - " + end
}

func (h *DHCPHandler) UpdatePool4(req *pb.UpdatePool4Request) error {
	exists := false
	updatePool := genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress())
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			for j, pool := range subnet.Pools {
				if pool.Pool == updatePool {
					h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Pools[j].ClientClass = req.GetClientClass()
					h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Pools[j].OptionDatas = genDHCPOptionDatas(
						Option4DNSServers, req.GetDomainServers(), req.GetRouters())
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
	} else {
		return fmt.Errorf("no found pool4 %s-%s in subnet4 %s", req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeletePool4(req *pb.DeletePool4Request) error {
	exists := false
	deletePool := genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress())
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			for j, pool := range subnet.Pools {
				if pool.Pool == deletePool {
					h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Pools = append(h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Pools[:j],
						h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Pools[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
	} else {
		return fmt.Errorf("no found pool4 %s-%s in subnet4 %s", req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreatePool6(req *pb.CreatePool6Request) error {
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Pools = append(h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Pools, Pool{
				Pool:        genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress()),
				ClientClass: req.GetClientClass(),
				OptionDatas: genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil),
			})
			break
		}
	}
	h.lock.Unlock()
	return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
}

func (h *DHCPHandler) UpdatePool6(req *pb.UpdatePool6Request) error {
	exists := false
	updatePool := genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress())
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, pool := range subnet.Pools {
				if pool.Pool == updatePool {
					h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Pools[j].ClientClass = req.GetClientClass()
					h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Pools[j].OptionDatas = genDHCPOptionDatas(
						Option6DNSServers, req.GetDnsServers(), nil)
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
	} else {
		return fmt.Errorf("no found pool6 %s-%s in subnet6 %s", req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeletePool6(req *pb.DeletePool6Request) error {
	exists := false
	deletePool := genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress())
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, pool := range subnet.Pools {
				if pool.Pool == deletePool {
					h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Pools = append(h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Pools[:j],
						h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Pools[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
	} else {
		return fmt.Errorf("no found pool6 %s-%s in subnet6 %s", req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreatePDPool(req *pb.CreatePDPoolRequest) error {
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].PDPools = append(h.conf.dhcp6Conf.DHCP6.Subnet6s[i].PDPools, PDPool{
				Prefix:       req.GetPrefix(),
				PrefixLen:    req.GetPrefixLen(),
				DelegatedLen: req.GetDelegatedLen(),
				ClientClass:  req.GetClientClass(),
				OptionDatas:  genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil),
			})
			break
		}
	}
	h.lock.Unlock()
	return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
}

func (h *DHCPHandler) UpdatePDPool(req *pb.UpdatePDPoolRequest) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, pdpool := range subnet.PDPools {
				if pdpool.Prefix == req.GetPrefix() {
					h.conf.dhcp6Conf.DHCP6.Subnet6s[i].PDPools[j].ClientClass = req.GetClientClass()
					h.conf.dhcp6Conf.DHCP6.Subnet6s[i].PDPools[j].OptionDatas = genDHCPOptionDatas(
						Option6DNSServers, req.GetDnsServers(), nil)
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
	} else {
		return fmt.Errorf("no found pd-pool %s in subnet %s", req.GetPrefix(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeletePDPool(req *pb.DeletePDPoolRequest) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, pdpool := range subnet.PDPools {
				if pdpool.Prefix == req.GetPrefix() {
					h.conf.dhcp6Conf.DHCP6.Subnet6s[i].PDPools = append(h.conf.dhcp6Conf.DHCP6.Subnet6s[i].PDPools[:j],
						h.conf.dhcp6Conf.DHCP6.Subnet6s[i].PDPools[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
	} else {
		return fmt.Errorf("no found pd-pool %s in subnet %s", req.GetPrefix(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreateReservation4(req *pb.CreateReservation4Request) error {
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Reservations = append(h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Reservations, Reservation4{
				HWAddress:   req.GetHwAddress(),
				IPAddress:   req.GetIpAddress(),
				OptionDatas: genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), req.GetRouters()),
			})
			break
		}
	}
	h.lock.Unlock()
	return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
}

func (h *DHCPHandler) UpdateReservation4(req *pb.UpdateReservation4Request) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			for j, reservation := range subnet.Reservations {
				if reservation.HWAddress == req.GetHwAddress() {
					h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Reservations[j].OptionDatas = genDHCPOptionDatas(
						Option4DNSServers, req.GetDomainServers(), req.GetRouters())
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
	} else {
		return fmt.Errorf("no found reservation4 %s in subnet4 %s", req.GetHwAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeleteReservation4(req *pb.DeleteReservation4Request) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			for j, reservation := range subnet.Reservations {
				if reservation.HWAddress == req.GetHwAddress() {
					h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Reservations = append(h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Reservations[:j],
						h.conf.dhcp4Conf.DHCP4.Subnet4s[i].Reservations[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
	} else {
		return fmt.Errorf("no found reservation4 %s in subnet4 %s", req.GetHwAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreateReservation6(req *pb.CreateReservation6Request) error {
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Reservations = append(h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Reservations, Reservation6{
				HWAddress:   req.GetHwAddress(),
				IPAddresses: req.GetIpAddresses(),
				OptionDatas: genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil),
			})
			break
		}
	}
	h.lock.Unlock()
	return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
}

func (h *DHCPHandler) UpdateReservation6(req *pb.UpdateReservation6Request) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, reservation := range subnet.Reservations {
				if reservation.HWAddress == req.GetHwAddress() {
					h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Reservations[j].OptionDatas = genDHCPOptionDatas(
						Option6DNSServers, req.GetDnsServers(), nil)
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
	} else {
		return fmt.Errorf("no found reservation6 %s in subnet6 %s", req.GetHwAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeleteReservation6(req *pb.DeleteReservation6Request) error {
	exists := false
	h.lock.Lock()
	for i, subnet := range h.conf.dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, reservation := range subnet.Reservations {
				if reservation.HWAddress == req.GetHwAddress() {
					h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Reservations = append(h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Reservations[:j],
						h.conf.dhcp6Conf.DHCP6.Subnet6s[i].Reservations[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
	} else {
		return fmt.Errorf("no found reservation6 %s in subnet6 %s", req.GetHwAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreateClientClass4(req *pb.CreateClientClass4Request) error {
	h.lock.Lock()
	h.conf.dhcp4Conf.DHCP4.ClientClasses = append(h.conf.dhcp4Conf.DHCP4.ClientClasses, ClientClass{
		Name: req.GetName(),
		Test: req.GetRegexp(),
	})
	h.lock.Unlock()

	return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
}

func (h *DHCPHandler) UpdateClientClass4(req *pb.UpdateClientClass4Request) error {
	exists := false
	h.lock.Lock()
	for i, clientclass := range h.conf.dhcp4Conf.DHCP4.ClientClasses {
		if clientclass.Name == req.GetName() {
			h.conf.dhcp4Conf.DHCP4.ClientClasses[i].Test = req.GetRegexp()
			exists = true
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
	} else {
		return fmt.Errorf("no found clientclass4 %s", req.GetName())
	}
}

func (h *DHCPHandler) DeleteClientClass4(req *pb.DeleteClientClass4Request) error {
	exists := false
	h.lock.Lock()
	for i, clientclass := range h.conf.dhcp4Conf.DHCP4.ClientClasses {
		if clientclass.Name == req.GetName() {
			h.conf.dhcp4Conf.DHCP4.ClientClasses = append(h.conf.dhcp4Conf.DHCP4.ClientClasses[:i],
				h.conf.dhcp4Conf.DHCP4.ClientClasses[i+1:]...)
			exists = true
			break
		}
	}
	h.lock.Unlock()

	if exists {
		return h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf)
	} else {
		return fmt.Errorf("no found clientclass4 %s", req.GetName())
	}
}

func (h *DHCPHandler) UpdateGlobalConfig(req *pb.UpdateGlobalConfigRequest) error {
	h.lock.Lock()
	h.conf.dhcp4Conf.DHCP4.ValidLifetime = req.GetValidLifetime()
	h.conf.dhcp4Conf.DHCP4.MinValidLifetime = req.GetMinValidLifetime()
	h.conf.dhcp4Conf.DHCP4.MaxValidLifetime = req.GetMaxValidLifetime()
	h.conf.dhcp4Conf.DHCP4.OptionDatas = genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), nil)
	h.conf.dhcp6Conf.DHCP6.ValidLifetime = req.GetValidLifetime()
	h.conf.dhcp6Conf.DHCP6.MinValidLifetime = req.GetMinValidLifetime()
	h.conf.dhcp6Conf.DHCP6.MaxValidLifetime = req.GetMaxValidLifetime()
	h.conf.dhcp6Conf.DHCP6.OptionDatas = genDHCPOptionDatas(Option6DNSServers, req.GetDomainServers(), nil)
	h.lock.Unlock()

	if err := h.reconfig([]string{DHCP4Name}, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf); err != nil {
		return err
	}

	return h.reconfig([]string{DHCP6Name}, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf)
}

type Lease4 struct {
	Address       uint32
	Hwaddr        []byte
	ClientId      []byte
	ValidLifetime uint32
	Expire        time.Time
	SubnetId      uint32
	FqdnFwd       bool
	FqdnRev       bool
	Hostname      string
	State         uint32
	UserContext   string
}

func (h *DHCPHandler) GetSubnet4Leases(req *pb.GetSubnet4LeasesRequest) ([]*pb.DHCPLease4, error) {
	var lease4s []*Lease4
	if err := h.getLeasesFromDB(map[string]interface{}{"subnet_id": req.GetId()}, &lease4s); err != nil {
		return nil, fmt.Errorf("get subnet4 %s leases from db failed: %s", req.GetId(), err.Error())
	}

	return lease4sToPbDHCPLease4s(lease4s), nil
}

func (h *DHCPHandler) getLeasesFromDB(conds map[string]interface{}, resources interface{}) error {
	return restdb.WithTx(h.db, func(tx restdb.Transaction) error {
		return tx.Fill(conds, resources)
	})
}

func lease4sToPbDHCPLease4s(lease4s []*Lease4) []*pb.DHCPLease4 {
	var pbleases []*pb.DHCPLease4
	now := time.Now()
	for _, lease := range lease4s {
		if lease.Expire.After(now) {
			pbleases = append(pbleases, lease4ToPbDHCPLease4(lease))
		}
	}

	return pbleases
}

func lease4ToPbDHCPLease4(lease *Lease4) *pb.DHCPLease4 {
	return &pb.DHCPLease4{
		Address:       ipv4FromUint32(lease.Address).String(),
		HwAddress:     string(lease.Hwaddr),
		ClientId:      string(lease.ClientId),
		SubnetId:      lease.SubnetId,
		ValidLifetime: lease.ValidLifetime,
		Expire:        lease.Expire.Unix(),
		Hostname:      lease.Hostname,
		State:         lease.State,
	}
}

func ipv4FromUint32(ipInt uint32) net.IP {
	return net.IP{
		uint8((ipInt & 0xff000000) >> 24),
		uint8((ipInt & 0x00ff0000) >> 16),
		uint8((ipInt & 0x0000ff00) >> 8),
		uint8(ipInt & 0x000000ff),
	}
}

func (h *DHCPHandler) GetPool4Leases(req *pb.GetPool4LeasesRequest) ([]*pb.DHCPLease4, error) {
	var lease4s []*Lease4
	if err := restdb.WithTx(h.db, func(tx restdb.Transaction) error {
		return tx.FillEx(&lease4s, "select * from lease4 where subnet_id = $1 and address between $2 and $3",
			req.GetSubnetId(), ipv4StrToUint32(req.GetBeginAddress()), ipv4StrToUint32(req.GetEndAddress()))
	}); err != nil {
		return nil, fmt.Errorf("get pool4 %s-%s with subnet %s leases from db failed: %s",
			req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId(), err.Error())
	}

	return lease4sToPbDHCPLease4s(lease4s), nil
}

func ipv4StrToUint32(ipv4 string) uint32 {
	return binary.BigEndian.Uint32(net.ParseIP(ipv4).To4())
}

func (h *DHCPHandler) GetReservation4Leases(req *pb.GetReservation4LeasesRequest) ([]*pb.DHCPLease4, error) {
	var lease4s []*Lease4
	if err := h.getLeasesFromDB(map[string]interface{}{
		"subnet_id": req.GetSubnetId(), "hwaddr": req.GetHwAddress()}, &lease4s); err != nil {
		return nil, fmt.Errorf("get reservation4 %s with subnet %s leases from db failed: %s",
			req.GetHwAddress(), req.GetSubnetId(), err.Error())
	}

	return lease4sToPbDHCPLease4s(lease4s), nil
}

func (h *DHCPHandler) GetSubnet4LeasesCount(req *pb.GetSubnet4LeasesRequest) (uint64, error) {
	var count uint64
	err := restdb.WithTx(h.db, func(tx restdb.Transaction) error {
		c, err := tx.Count(TableLease4, map[string]interface{}{"subnet_id": req.GetId()})
		count = uint64(c)
		return err
	})

	return count, err
}

func (h *DHCPHandler) GetPool4LeasesCount(req *pb.GetPool4LeasesRequest) (uint64, error) {
	var count uint64
	err := restdb.WithTx(h.db, func(tx restdb.Transaction) error {
		c, err := tx.CountEx(TableLease4, "select count(*) from lease4 where subnet_id = $1 and address between $2 and $3",
			req.GetSubnetId(), ipv4StrToUint32(req.GetBeginAddress()), ipv4StrToUint32(req.GetEndAddress()))
		count = uint64(c)
		return err
	})

	return count, err
}

func (h *DHCPHandler) GetReservation4LeasesCount(req *pb.GetReservation4LeasesRequest) (uint64, error) {
	var count uint64
	err := restdb.WithTx(h.db, func(tx restdb.Transaction) error {
		c, err := tx.Count(TableLease4, map[string]interface{}{"subnet_id": req.GetSubnetId(), "hwaddr": req.GetHwAddress()})
		count = uint64(c)
		return err
	})

	return count, err
}

type Lease6 struct {
	Address       string
	Duid          []byte
	ValidLifetime uint32
	Expire        time.Time
	SubnetId      uint32
	PrefLifetime  uint32
	LeaseType     uint32
	Iaid          uint32
	PrefixLen     uint32
	FqdnFwd       bool
	FqdnRev       bool
	Hostname      string
	State         uint32
	Hwaddr        []byte
	Hwtype        uint32
	HwaddrSource  uint32
	UserContext   string
}

func (h *DHCPHandler) GetSubnet6Leases(req *pb.GetSubnet6LeasesRequest) ([]*pb.DHCPLease6, error) {
	var lease6s []*Lease6
	if err := h.getLeasesFromDB(map[string]interface{}{"subnet_id": req.GetId()}, &lease6s); err != nil {
		return nil, fmt.Errorf("get subnet6 %s leases from db failed: %s", req.GetId(), err.Error())
	}

	return lease6sToPbDHCPLease6s(lease6s), nil
}

func lease6sToPbDHCPLease6s(lease6s []*Lease6) []*pb.DHCPLease6 {
	var pbleases []*pb.DHCPLease6
	now := time.Now()
	for _, lease := range lease6s {
		if lease.Expire.After(now) {
			pbleases = append(pbleases, lease6ToPbDHCPLease6(lease))
		}
	}

	return pbleases
}

func lease6ToPbDHCPLease6(lease *Lease6) *pb.DHCPLease6 {
	return &pb.DHCPLease6{
		Address:         lease.Address,
		SubnetId:        lease.SubnetId,
		HwAddress:       string(lease.Hwaddr),
		HwAddressSource: lease.HwaddrSource,
		Duid:            string(lease.Duid),
		ValidLifetime:   lease.ValidLifetime,
		Expire:          lease.Expire.Unix(),
		PrefixLen:       lease.PrefixLen,
		LeaseType:       pb.DHCPLease6_LeaseType(lease.LeaseType),
		State:           lease.State,
	}
}
