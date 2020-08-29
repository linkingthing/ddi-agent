package grpcservice

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/zdnscloud/cement/log"
	"google.golang.org/grpc"

	"github.com/linkingthing/ddi-agent/config"
	"github.com/linkingthing/ddi-agent/pkg/dhcp/util"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	monitorpb "github.com/linkingthing/ddi-monitor/pkg/proto"
)

const (
	DHCP4ConfigFileName    = "kea-dhcp4.conf"
	DHCP6ConfigFileName    = "kea-dhcp6.conf"
	DHCP4Name              = "dhcp4"
	DHCP6Name              = "dhcp6"
	DHCPAgentName          = "ctrl-agent"
	Option4DNSServers      = "domain-name-servers"
	Option6DNSServers      = "dns-servers"
	Option4Routers         = "routers"
	DHCPCommandConfigSet   = "config-set"
	DHCPCommandConfigWrite = "config-write"
	PostgresqlConnStr      = "user=%s password=%s host=localhost port=%d database=%s sslmode=disable pool_max_conns=10"
	TableLease4            = "lease4"
	TableLease6            = "lease6"
	HttpClientTimeout      = 10
	HttpScheme             = "http://"
)

type DHCPHandler struct {
	monitorClient monitorpb.DDIMonitorClient
	cmdUrl        string
	conf          *DHCPConfig
	lock          sync.RWMutex
	db            *pgxpool.Pool
	httpClient    *http.Client
}

type DHCPConfig struct {
	dhcp4Conf *DHCP4Config
	dhcp6Conf *DHCP6Config
}

func newDHCPHandler(conn *grpc.ClientConn, conf *config.AgentConfig) (*DHCPHandler, error) {
	cmdUrl, err := url.Parse(HttpScheme + conf.DHCP.CmdAddr)
	if err != nil {
		return nil, fmt.Errorf("parse dhcp cmd url %s failed: %s", HttpScheme+conf.DHCP.CmdAddr, err.Error())
	}

	db, err := pgxpool.Connect(context.Background(), fmt.Sprintf(PostgresqlConnStr, conf.DHCP.DB.User, conf.DHCP.DB.Password,
		conf.DHCP.DB.Port, conf.DHCP.DB.Name))
	if err != nil {
		return nil, err
	}

	handler := &DHCPHandler{
		monitorClient: monitorpb.NewDDIMonitorClient(conn),
		cmdUrl:        cmdUrl.String(),
		db:            db,
		httpClient:    &http.Client{Timeout: HttpClientTimeout * time.Second}}
	if err := handler.loadDHCPConfig(conf); err != nil {
		return nil, err
	}

	if err := handler.startDHCP(); err != nil {
		return nil, err
	}

	go handler.monitor()
	return handler, nil
}

func (h *DHCPHandler) loadDHCPConfig(conf *config.AgentConfig) error {
	if _, err := os.Stat(conf.DHCP.ConfigDir); os.IsNotExist(err) {
		if err := os.Mkdir(conf.DHCP.ConfigDir, os.ModePerm); err != nil {
			return fmt.Errorf("create config dir %s failed: %s", conf.DHCP.ConfigDir, err.Error())
		}
	}

	genDHCP4ConfFile := false
	var dhcp4Conf DHCP4Config
	dhcp4ConfPath := path.Join(conf.DHCP.ConfigDir, DHCP4ConfigFileName)
	if _, err := os.Stat(dhcp4ConfPath); os.IsNotExist(err) {
		dhcp4Conf = genDefaultDHCP4Config(conf.DHCP.ConfigDir, conf)
		genDHCP4ConfFile = true
	} else {
		if err := parseJsonConfig(&dhcp4Conf, dhcp4ConfPath); err != nil {
			return fmt.Errorf("load dhcp4 config failed: %s", err.Error())
		} else {
			if interfaces := getInterfaces(true); isDiffStrSlice(dhcp4Conf.DHCP4.InterfacesConfig.Interfaces, interfaces) {
				dhcp4Conf.DHCP4.InterfacesConfig.Interfaces = interfaces
				genDHCP4ConfFile = true
			}
		}
	}

	if genDHCP4ConfFile {
		if err := genDefaultDHCPConfigFile(dhcp4ConfPath, &dhcp4Conf); err != nil {
			return err
		}
	}

	genDHCP6ConfFile := false
	var dhcp6Conf DHCP6Config
	dhcp6ConfPath := path.Join(conf.DHCP.ConfigDir, DHCP6ConfigFileName)
	if _, err := os.Stat(dhcp6ConfPath); os.IsNotExist(err) {
		dhcp6Conf = genDefaultDHCP6Config(conf.DHCP.ConfigDir, conf)
		genDHCP6ConfFile = true
	} else {
		if err := parseJsonConfig(&dhcp6Conf, dhcp6ConfPath); err != nil {
			return fmt.Errorf("load dhcp6 config failed: %s", err.Error())
		} else {
			if interfaces := getInterfaces(false); isDiffStrSlice(dhcp6Conf.DHCP6.InterfacesConfig.Interfaces, interfaces) {
				dhcp6Conf.DHCP6.InterfacesConfig.Interfaces = interfaces
				genDHCP6ConfFile = true
			}
		}
	}

	if genDHCP6ConfFile {
		if err := genDefaultDHCPConfigFile(dhcp6ConfPath, &dhcp6Conf); err != nil {
			return err
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

func isDiffStrSlice(s1s, s2s []string) bool {
	if len(s1s) != len(s2s) {
		return true
	}

	intersection := getStrSliceIntersection(s1s, s2s)
	return len(intersection) != len(s1s) || len(intersection) != len(s2s)
}

func getStrSliceIntersection(s1s, s2s []string) []string {
	var intersection []string
	for _, s1 := range s1s {
		for _, s2 := range s2s {
			if s1 == s2 {
				intersection = append(intersection, s1)
				break
			}
		}
	}
	return intersection
}

func (h *DHCPHandler) startDHCP() error {
	_, err := h.monitorClient.StartDHCP(context.Background(), &monitorpb.StartDHCPRequest{})
	return err
}

func (h *DHCPHandler) stopDHCP() error {
	_, err := h.monitorClient.StopDHCP(context.Background(), &monitorpb.StopDHCPRequest{})
	return err
}

func (h *DHCPHandler) monitor() {
	for {
		resp, err := h.monitorClient.GetDHCPState(context.Background(), &monitorpb.GetDHCPStateRequest{})
		if err == nil && resp.GetIsRunning() == false {
			h.stopDHCP()
			if err := h.startDHCP(); err != nil {
				log.Warnf("start dhcp failed: %s", err.Error())
			}
		}

		time.Sleep(10 * time.Second)
	}
}

func (h *DHCPHandler) CreateSubnet4(req *pb.CreateSubnet4Request) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	dhcp4Conf.DHCP4.Subnet4s = append(dhcp4Conf.DHCP4.Subnet4s, Subnet4{
		ID:               req.GetId(),
		Subent:           req.GetIpnet(),
		ClientClass:      req.GetClientClass(),
		ValidLifetime:    req.GetValidLifetime(),
		MaxValidLifetime: req.GetMaxValidLifetime(),
		MinValidLifetime: req.GetMinValidLifetime(),
		OptionDatas:      genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), req.GetRouters()),
		Relay:            genRelayAgent(req.GetRelayAgentAddresses()),
	})

	return h.reconfig4(dhcp4Conf)
}

func genDhcp4ConfFromDeepCopy(src *DHCP4Config) *DHCP4Config {
	dst := &DHCP4Config{}
	*dst = *src
	dst.DHCP4.Subnet4s = make([]Subnet4, len(src.DHCP4.Subnet4s))
	dst.DHCP4.ClientClasses = make([]ClientClass, len(src.DHCP4.ClientClasses))
	copy(dst.DHCP4.ClientClasses, src.DHCP4.ClientClasses)
	copy(dst.DHCP4.Subnet4s, src.DHCP4.Subnet4s)
	return dst
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

func genRelayAgent(relayAgentAddrs []string) RelayAgent {
	if len(relayAgentAddrs) == 0 {
		return RelayAgent{IPAddresses: make([]string, 0)}
	}

	return RelayAgent{IPAddresses: relayAgentAddrs}
}

func (h *DHCPHandler) reconfig4(dhcp4Conf *DHCP4Config) error {
	if err := h.reconfig(DHCP4Name, dhcp4Conf.Path, dhcp4Conf); err != nil {
		if err := h.reconfig(DHCP4Name, h.conf.dhcp4Conf.Path, h.conf.dhcp4Conf); err != nil {
			log.Errorf("rollback dhcp4 to old config failed: %s", err.Error())
		}
		return err
	}

	h.conf.dhcp4Conf = dhcp4Conf
	return nil
}

func (h *DHCPHandler) reconfig(service string, configPath string, conf interface{}) error {
	if err := h.setDHCPConfigToMemory(service, conf); err != nil {
		return err
	}

	return h.writeDHCPConfigToFile(service, configPath)
}

func (h *DHCPHandler) setDHCPConfigToMemory(service string, conf interface{}) error {
	_, err := SendHttpRequestToDHCP(h.httpClient, h.cmdUrl, &DHCPCmdRequest{
		Command:   DHCPCommandConfigSet,
		Services:  []string{service},
		Arguments: conf,
	})

	return err
}

func (h *DHCPHandler) writeDHCPConfigToFile(service string, configPath string) error {
	_, err := SendHttpRequestToDHCP(h.httpClient, h.cmdUrl, &DHCPCmdRequest{
		Command:  DHCPCommandConfigWrite,
		Services: []string{service},
		Arguments: map[string]interface{}{
			"filename": configPath,
		},
	})

	return err
}

func (h *DHCPHandler) UpdateSubnet4(req *pb.UpdateSubnet4Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, subnet := range dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetId() {
			dhcp4Conf.DHCP4.Subnet4s[i].ClientClass = req.GetClientClass()
			dhcp4Conf.DHCP4.Subnet4s[i].ValidLifetime = req.GetValidLifetime()
			dhcp4Conf.DHCP4.Subnet4s[i].MaxValidLifetime = req.GetMaxValidLifetime()
			dhcp4Conf.DHCP4.Subnet4s[i].MinValidLifetime = req.GetMinValidLifetime()
			dhcp4Conf.DHCP4.Subnet4s[i].OptionDatas = genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), req.GetRouters())
			dhcp4Conf.DHCP4.Subnet4s[i].Relay = genRelayAgent(req.GetRelayAgentAddresses())
			exists = true
			break
		}
	}

	if exists {
		return h.reconfig4(dhcp4Conf)
	} else {
		return fmt.Errorf("no found subnet4 %d", req.GetId())
	}
}

func (h *DHCPHandler) DeleteSubnet4(req *pb.DeleteSubnet4Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, subnet := range dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetId() {
			dhcp4Conf.DHCP4.Subnet4s = append(dhcp4Conf.DHCP4.Subnet4s[:i], dhcp4Conf.DHCP4.Subnet4s[i+1:]...)
			exists = true
			break
		}
	}

	if exists {
		return h.reconfig4(dhcp4Conf)
	} else {
		return fmt.Errorf("no found subnet4 %d", req.GetId())
	}
}

func (h *DHCPHandler) CreateSubnet6(req *pb.CreateSubnet6Request) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	dhcp6Conf.DHCP6.Subnet6s = append(dhcp6Conf.DHCP6.Subnet6s, Subnet6{
		ID:               req.GetId(),
		Subent:           req.GetIpnet(),
		ClientClass:      req.GetClientClass(),
		ValidLifetime:    req.GetValidLifetime(),
		MaxValidLifetime: req.GetMaxValidLifetime(),
		MinValidLifetime: req.GetMinValidLifetime(),
		OptionDatas:      genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil),
		Relay:            genRelayAgent(req.GetRelayAgentAddresses()),
	})

	return h.reconfig6(dhcp6Conf)
}

func genDhcp6ConfFromDeepCopy(src *DHCP6Config) *DHCP6Config {
	dst := &DHCP6Config{}
	*dst = *src
	dst.DHCP6.Subnet6s = make([]Subnet6, len(dst.DHCP6.Subnet6s))
	copy(dst.DHCP6.Subnet6s, src.DHCP6.Subnet6s)
	return dst
}

func (h *DHCPHandler) reconfig6(dhcp6Conf *DHCP6Config) error {
	if err := h.reconfig(DHCP6Name, dhcp6Conf.Path, dhcp6Conf); err != nil {
		if err := h.reconfig(DHCP6Name, h.conf.dhcp6Conf.Path, h.conf.dhcp6Conf); err != nil {
			log.Errorf("rollback dhcp6 to old config failed: %s", err.Error())
		}
		return err
	}

	h.conf.dhcp6Conf = dhcp6Conf
	return nil
}

func (h *DHCPHandler) UpdateSubnet6(req *pb.UpdateSubnet6Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetId() {
			dhcp6Conf.DHCP6.Subnet6s[i].ClientClass = req.GetClientClass()
			dhcp6Conf.DHCP6.Subnet6s[i].ValidLifetime = req.GetValidLifetime()
			dhcp6Conf.DHCP6.Subnet6s[i].MaxValidLifetime = req.GetMaxValidLifetime()
			dhcp6Conf.DHCP6.Subnet6s[i].MinValidLifetime = req.GetMinValidLifetime()
			dhcp6Conf.DHCP6.Subnet6s[i].OptionDatas = genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil)
			dhcp6Conf.DHCP6.Subnet6s[i].Relay = genRelayAgent(req.GetRelayAgentAddresses())
			exists = true
			break
		}
	}

	if exists {
		return h.reconfig6(dhcp6Conf)
	} else {
		return fmt.Errorf("no found subnet6 %d", req.GetId())
	}
}

func (h *DHCPHandler) DeleteSubnet6(req *pb.DeleteSubnet6Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetId() {
			dhcp6Conf.DHCP6.Subnet6s = append(dhcp6Conf.DHCP6.Subnet6s[:i], dhcp6Conf.DHCP6.Subnet6s[i+1:]...)
			exists = true
			break
		}
	}

	if exists {
		return h.reconfig6(dhcp6Conf)
	} else {
		return fmt.Errorf("no found subnet6 %d", req.GetId())
	}
}

func (h *DHCPHandler) CreatePool4(req *pb.CreatePool4Request) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, subnet := range dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			dhcp4Conf.DHCP4.Subnet4s[i].Pools = append(dhcp4Conf.DHCP4.Subnet4s[i].Pools, Pool{
				Pool:        genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress()),
				ClientClass: req.GetClientClass(),
				OptionDatas: genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), req.GetRouters()),
			})
			break
		}
	}
	return h.reconfig4(dhcp4Conf)
}

func genPoolByBeginAndEnd(begin, end string) string {
	return begin + " - " + end
}

func (h *DHCPHandler) UpdatePool4(req *pb.UpdatePool4Request) error {
	exists := false
	updatePool := genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress())
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, subnet := range dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			for j, pool := range subnet.Pools {
				if pool.Pool == updatePool {
					dhcp4Conf.DHCP4.Subnet4s[i].Pools[j].ClientClass = req.GetClientClass()
					dhcp4Conf.DHCP4.Subnet4s[i].Pools[j].OptionDatas = genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), req.GetRouters())
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig4(dhcp4Conf)
	} else {
		return fmt.Errorf("no found pool4 %s-%s in subnet4 %d", req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeletePool4(req *pb.DeletePool4Request) error {
	exists := false
	deletePool := genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress())
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, subnet := range dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			for j, pool := range subnet.Pools {
				if pool.Pool == deletePool {
					dhcp4Conf.DHCP4.Subnet4s[i].Pools = append(dhcp4Conf.DHCP4.Subnet4s[i].Pools[:j], dhcp4Conf.DHCP4.Subnet4s[i].Pools[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig4(dhcp4Conf)
	} else {
		return fmt.Errorf("no found pool4 %s-%s in subnet4 %d", req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreatePool6(req *pb.CreatePool6Request) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			dhcp6Conf.DHCP6.Subnet6s[i].Pools = append(dhcp6Conf.DHCP6.Subnet6s[i].Pools, Pool{
				Pool:        genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress()),
				ClientClass: req.GetClientClass(),
				OptionDatas: genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil),
			})
			break
		}
	}
	return h.reconfig6(dhcp6Conf)
}

func (h *DHCPHandler) UpdatePool6(req *pb.UpdatePool6Request) error {
	exists := false
	updatePool := genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress())
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, pool := range subnet.Pools {
				if pool.Pool == updatePool {
					dhcp6Conf.DHCP6.Subnet6s[i].Pools[j].ClientClass = req.GetClientClass()
					dhcp6Conf.DHCP6.Subnet6s[i].Pools[j].OptionDatas = genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil)
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig6(dhcp6Conf)
	} else {
		return fmt.Errorf("no found pool6 %s-%s in subnet6 %d", req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeletePool6(req *pb.DeletePool6Request) error {
	exists := false
	deletePool := genPoolByBeginAndEnd(req.GetBeginAddress(), req.GetEndAddress())
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, pool := range subnet.Pools {
				if pool.Pool == deletePool {
					dhcp6Conf.DHCP6.Subnet6s[i].Pools = append(dhcp6Conf.DHCP6.Subnet6s[i].Pools[:j], dhcp6Conf.DHCP6.Subnet6s[i].Pools[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig6(dhcp6Conf)
	} else {
		return fmt.Errorf("no found pool6 %s-%s in subnet6 %d", req.GetBeginAddress(), req.GetEndAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreatePDPool(req *pb.CreatePDPoolRequest) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			dhcp6Conf.DHCP6.Subnet6s[i].PDPools = append(dhcp6Conf.DHCP6.Subnet6s[i].PDPools, PDPool{
				Prefix:       req.GetPrefix(),
				PrefixLen:    req.GetPrefixLen(),
				DelegatedLen: req.GetDelegatedLen(),
				ClientClass:  req.GetClientClass(),
				OptionDatas:  genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil),
			})
			break
		}
	}
	return h.reconfig6(dhcp6Conf)
}

func (h *DHCPHandler) UpdatePDPool(req *pb.UpdatePDPoolRequest) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, pdpool := range subnet.PDPools {
				if pdpool.Prefix == req.GetPrefix() {
					dhcp6Conf.DHCP6.Subnet6s[i].PDPools[j].ClientClass = req.GetClientClass()
					dhcp6Conf.DHCP6.Subnet6s[i].PDPools[j].OptionDatas = genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil)
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig6(dhcp6Conf)
	} else {
		return fmt.Errorf("no found pd-pool %s in subnet %d", req.GetPrefix(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeletePDPool(req *pb.DeletePDPoolRequest) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, pdpool := range subnet.PDPools {
				if pdpool.Prefix == req.GetPrefix() {
					dhcp6Conf.DHCP6.Subnet6s[i].PDPools = append(dhcp6Conf.DHCP6.Subnet6s[i].PDPools[:j], dhcp6Conf.DHCP6.Subnet6s[i].PDPools[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig6(dhcp6Conf)
	} else {
		return fmt.Errorf("no found pd-pool %s in subnet %d", req.GetPrefix(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreateReservation4(req *pb.CreateReservation4Request) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, subnet := range dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			dhcp4Conf.DHCP4.Subnet4s[i].Reservations = append(dhcp4Conf.DHCP4.Subnet4s[i].Reservations, Reservation4{
				HWAddress:   req.GetHwAddress(),
				IPAddress:   req.GetIpAddress(),
				OptionDatas: genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), req.GetRouters()),
			})
			break
		}
	}
	return h.reconfig4(dhcp4Conf)
}

func (h *DHCPHandler) UpdateReservation4(req *pb.UpdateReservation4Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, subnet := range dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			for j, reservation := range subnet.Reservations {
				if reservation.HWAddress == req.GetHwAddress() {
					dhcp4Conf.DHCP4.Subnet4s[i].Reservations[j].OptionDatas = genDHCPOptionDatas(
						Option4DNSServers, req.GetDomainServers(), req.GetRouters())
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig4(dhcp4Conf)
	} else {
		return fmt.Errorf("no found reservation4 %s in subnet4 %d", req.GetHwAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeleteReservation4(req *pb.DeleteReservation4Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, subnet := range dhcp4Conf.DHCP4.Subnet4s {
		if subnet.ID == req.GetSubnetId() {
			for j, reservation := range subnet.Reservations {
				if reservation.HWAddress == req.GetHwAddress() {
					dhcp4Conf.DHCP4.Subnet4s[i].Reservations = append(dhcp4Conf.DHCP4.Subnet4s[i].Reservations[:j],
						dhcp4Conf.DHCP4.Subnet4s[i].Reservations[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig4(dhcp4Conf)
	} else {
		return fmt.Errorf("no found reservation4 %s in subnet4 %d", req.GetHwAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreateReservation6(req *pb.CreateReservation6Request) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			dhcp6Conf.DHCP6.Subnet6s[i].Reservations = append(dhcp6Conf.DHCP6.Subnet6s[i].Reservations, Reservation6{
				HWAddress:   req.GetHwAddress(),
				IPAddresses: req.GetIpAddresses(),
				OptionDatas: genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil),
			})
			break
		}
	}
	return h.reconfig6(dhcp6Conf)
}

func (h *DHCPHandler) UpdateReservation6(req *pb.UpdateReservation6Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, reservation := range subnet.Reservations {
				if reservation.HWAddress == req.GetHwAddress() {
					dhcp6Conf.DHCP6.Subnet6s[i].Reservations[j].OptionDatas = genDHCPOptionDatas(Option6DNSServers, req.GetDnsServers(), nil)
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig6(dhcp6Conf)
	} else {
		return fmt.Errorf("no found reservation6 %s in subnet6 %d", req.GetHwAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) DeleteReservation6(req *pb.DeleteReservation6Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	for i, subnet := range dhcp6Conf.DHCP6.Subnet6s {
		if subnet.ID == req.GetSubnetId() {
			for j, reservation := range subnet.Reservations {
				if reservation.HWAddress == req.GetHwAddress() {
					dhcp6Conf.DHCP6.Subnet6s[i].Reservations = append(dhcp6Conf.DHCP6.Subnet6s[i].Reservations[:j],
						dhcp6Conf.DHCP6.Subnet6s[i].Reservations[j+1:]...)
					exists = true
					break
				}
			}
			break
		}
	}

	if exists {
		return h.reconfig6(dhcp6Conf)
	} else {
		return fmt.Errorf("no found reservation6 %s in subnet6 %d", req.GetHwAddress(), req.GetSubnetId())
	}
}

func (h *DHCPHandler) CreateClientClass4(req *pb.CreateClientClass4Request) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	dhcp4Conf.DHCP4.ClientClasses = append(dhcp4Conf.DHCP4.ClientClasses, ClientClass{
		Name: req.GetName(),
		Test: req.GetRegexp(),
	})

	return h.reconfig4(dhcp4Conf)
}

func (h *DHCPHandler) UpdateClientClass4(req *pb.UpdateClientClass4Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, clientclass := range dhcp4Conf.DHCP4.ClientClasses {
		if clientclass.Name == req.GetName() {
			dhcp4Conf.DHCP4.ClientClasses[i].Test = req.GetRegexp()
			exists = true
			break
		}
	}

	if exists {
		return h.reconfig4(dhcp4Conf)
	} else {
		return fmt.Errorf("no found clientclass4 %s", req.GetName())
	}
}

func (h *DHCPHandler) DeleteClientClass4(req *pb.DeleteClientClass4Request) error {
	exists := false
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	for i, clientclass := range dhcp4Conf.DHCP4.ClientClasses {
		if clientclass.Name == req.GetName() {
			dhcp4Conf.DHCP4.ClientClasses = append(dhcp4Conf.DHCP4.ClientClasses[:i], dhcp4Conf.DHCP4.ClientClasses[i+1:]...)
			exists = true
			break
		}
	}

	if exists {
		return h.reconfig4(dhcp4Conf)
	} else {
		return fmt.Errorf("no found clientclass4 %s", req.GetName())
	}
}

func (h *DHCPHandler) UpdateGlobalConfig(req *pb.UpdateGlobalConfigRequest) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	dhcp4Conf := genDhcp4ConfFromDeepCopy(h.conf.dhcp4Conf)
	dhcp4Conf.DHCP4.ValidLifetime = req.GetValidLifetime()
	dhcp4Conf.DHCP4.MinValidLifetime = req.GetMinValidLifetime()
	dhcp4Conf.DHCP4.MaxValidLifetime = req.GetMaxValidLifetime()
	dhcp4Conf.DHCP4.OptionDatas = genDHCPOptionDatas(Option4DNSServers, req.GetDomainServers(), nil)

	dhcp6Conf := genDhcp6ConfFromDeepCopy(h.conf.dhcp6Conf)
	dhcp6Conf.DHCP6.ValidLifetime = req.GetValidLifetime()
	dhcp6Conf.DHCP6.MinValidLifetime = req.GetMinValidLifetime()
	dhcp6Conf.DHCP6.MaxValidLifetime = req.GetMaxValidLifetime()
	dhcp6Conf.DHCP6.OptionDatas = genDHCPOptionDatas(Option6DNSServers, req.GetDomainServers(), nil)

	if err := h.reconfig4(dhcp4Conf); err != nil {
		return err
	}

	return h.reconfig6(dhcp6Conf)
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

func (h *DHCPHandler) GetSubnet4Leases(req *pb.GetSubnet4LeasesRequest) ([]*pb.DHCPLease, error) {
	rows, err := h.db.Query(context.Background(), "select * from lease4 where subnet_id=$1 and state = 0 and expire > now()",
		req.GetId())
	if err != nil {
		return nil, err
	}

	var pbleases []*pb.DHCPLease
	for rows.Next() {
		var lease4 Lease4
		if err := rows.Scan(&lease4.Address, &lease4.Hwaddr, &lease4.ClientId, &lease4.ValidLifetime, &lease4.Expire,
			&lease4.SubnetId, &lease4.FqdnFwd, &lease4.FqdnRev, &lease4.Hostname, &lease4.State, &lease4.UserContext,
		); err != nil {
			return nil, err
		}

		pbleases = append(pbleases, &pb.DHCPLease{
			Address:       ipv4FromUint32(lease4.Address).String(),
			HwAddress:     net.HardwareAddr(lease4.Hwaddr).String(),
			SubnetId:      lease4.SubnetId,
			ValidLifetime: lease4.ValidLifetime,
			Expire:        lease4.Expire.Unix(),
			Hostname:      lease4.Hostname,
		})
	}

	return pbleases, nil
}

func ipv4FromUint32(ipInt uint32) net.IP {
	return net.IP{
		uint8((ipInt & 0xff000000) >> 24),
		uint8((ipInt & 0x00ff0000) >> 16),
		uint8((ipInt & 0x0000ff00) >> 8),
		uint8(ipInt & 0x000000ff),
	}
}

func (h *DHCPHandler) GetSubnet4LeasesCount(req *pb.GetSubnet4LeasesCountRequest) (uint64, error) {
	return h.getLeasesCountFromDB("select count(*) from lease4 where subnet_id = $1 and state = 0 and expire > now()", req.GetId())
}

func (h *DHCPHandler) getLeasesCountFromDB(sql string, args ...interface{}) (uint64, error) {
	var count uint64
	if err := h.db.QueryRow(context.Background(), sql, args...).Scan(&count); err != nil {
		return 0, err
	}

	return count, nil
}

func (h *DHCPHandler) GetPool4LeasesCount(req *pb.GetPool4LeasesCountRequest) (uint64, error) {
	return h.getLeasesCountFromDB("select count(*) from lease4 where subnet_id = $1 and address between $2 and $3 and state = 0 and expire > now()",
		req.GetSubnetId(), ipv4StrToUint32(req.GetBeginAddress()), ipv4StrToUint32(req.GetEndAddress()))
}

func ipv4StrToUint32(ipv4 string) uint32 {
	return util.Ipv4ToUint32(net.ParseIP(ipv4))
}

func (h *DHCPHandler) GetReservation4LeasesCount(req *pb.GetReservation4LeasesCountRequest) (uint64, error) {
	return h.getLeasesCountFromDB("select count(*) from lease4 where subnet_id = $1 and hwaddr = $2 and state = 0 and expire > now()",
		req.GetSubnetId(), req.GetHwAddress())
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

func (h *DHCPHandler) GetSubnet6Leases(req *pb.GetSubnet6LeasesRequest) ([]*pb.DHCPLease, error) {
	return h.getSubnet6Leases(req.GetId())
}

func (h *DHCPHandler) getSubnet6Leases(subnetID uint32) ([]*pb.DHCPLease, error) {
	rows, err := h.db.Query(context.Background(), "select * from lease6 where subnet_id = $1 and state = 0 and expire > now()",
		subnetID)
	if err != nil {
		return nil, err
	}

	var pbleases []*pb.DHCPLease
	for rows.Next() {
		var lease6 Lease6
		if err := rows.Scan(&lease6.Address, &lease6.Duid, &lease6.ValidLifetime, &lease6.Expire, &lease6.SubnetId,
			&lease6.PrefLifetime, &lease6.LeaseType, &lease6.Iaid, &lease6.PrefixLen, &lease6.FqdnFwd, &lease6.FqdnRev,
			&lease6.Hostname, &lease6.State, &lease6.Hwaddr, &lease6.Hwtype, &lease6.HwaddrSource, &lease6.UserContext,
		); err != nil {
			return nil, err
		}

		pbleases = append(pbleases, &pb.DHCPLease{
			Address:       lease6.Address,
			SubnetId:      lease6.SubnetId,
			HwAddress:     net.HardwareAddr(lease6.Hwaddr).String(),
			ValidLifetime: lease6.ValidLifetime,
			Expire:        lease6.Expire.Unix(),
			Hostname:      lease6.Hostname,
		})
	}

	return pbleases, nil
}

func (h *DHCPHandler) GetSubnet6LeasesCount(req *pb.GetSubnet6LeasesCountRequest) (uint64, error) {
	return h.getLeasesCountFromDB("select count(*) from lease6 where subnet_id = $1 and state = 0 and expire > now()", req.GetId())
}

func (h *DHCPHandler) GetPool6LeasesCount(req *pb.GetPool6LeasesCountRequest) (uint64, error) {
	pblease6s, err := h.getSubnet6Leases(req.GetSubnetId())
	if err != nil {
		return 0, fmt.Errorf("get subnet6 %d leases from db failed: %s", req.GetSubnetId(), err.Error())
	}

	var count uint64
	for _, lease6 := range pblease6s {
		if ipV6InPool(lease6.Address, req.GetBeginAddress(), req.GetEndAddress()) {
			count += 1
		}
	}

	return count, nil
}

func ipV6InPool(ip, begin, end string) bool {
	if util.Ipv6ToBigInt(net.ParseIP(end)).Cmp(util.Ipv6ToBigInt(net.ParseIP(ip))) == -1 ||
		util.Ipv6ToBigInt(net.ParseIP(ip)).Cmp(util.Ipv6ToBigInt(net.ParseIP(begin))) == -1 {
		return false
	}

	return true
}

func (h *DHCPHandler) GetReservation6LeasesCount(req *pb.GetReservation6LeasesCountRequest) (uint64, error) {
	return h.getLeasesCountFromDB("select count(*) from lease6 where subnet_id = $1 and hwaddr = $2 and state = 0 and expire > now()",
		req.GetSubnetId(), req.GetHwAddress())
}
