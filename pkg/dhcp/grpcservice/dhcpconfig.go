package grpcservice

type DHCP4Config struct {
	Path  string `json:"-"`
	DHCP4 DHCP4  `json:"Dhcp4"`
}

type DHCP4 struct {
	InterfacesConfig InterfacesConfig `json:"interfaces-config,omitempty"`
	LeaseDatabase    LeaseDatabase    `json:"lease-database,omitempty"`
	ValidLifetime    uint32           `json:"valid-lifetime,omitempty"`
	MaxValidLifetime uint32           `json:"max-valid-lifetime,omitempty"`
	MinValidLifetime uint32           `json:"min-valid-lifetime,omitempty"`
	ClientClasses    []ClientClass    `json:"client-classes,omitempty"`
	OptionDatas      []OptionData     `json:"option-data,omitempty"`
	Loggers          []Logger         `json:"loggers,omitempty"`
	Subnet4s         []Subnet4        `json:"subnet4,omitempty"`
}

type InterfacesConfig struct {
	Interfaces []string `json:"interfaces,omitempty"`
}

type LeaseDatabase struct {
	Type     string `json:"type,omitempty"`
	Name     string `json:"name,omitempty"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
	Port     uint32 `json:"port,omitempty"`
	Host     string `json:"host,omitempty"`
}

type ClientClass struct {
	Name string `json:"name,omitempty"`
	Test string `json:"test,omitempty"`
}

type OptionData struct {
	Name string `json:"name,omitempty"`
	Data string `json:"data,omitempty"`
}

type Logger struct {
	Name          string         `json:"name,omitempty"`
	Severity      string         `json:"severity,omitempty"`
	DebugLevel    uint32         `json:"debuglevel,omitempty"`
	OutputOptions []OutputOption `json:"output_options,omitempty"`
}

type OutputOption struct {
	Output  string `json:"output,omitempty"`
	Flush   bool   `json:"flush,omitempty"`
	Maxsize uint32 `json:"maxsize,omitempty"`
	MaxVer  uint32 `json:"maxver,omitempty"`
	Pattern string `json:"pattern,omitempty"`
}

type Subnet4 struct {
	ID               uint32         `json:"id,omitempty"`
	Subent           string         `json:"subnet,omitempty"`
	ClientClass      string         `json:"client-class,omitempty"`
	Pools            []Pool         `json:"pools,omitempty"`
	Reservations     []Reservation4 `json:"reservations,omitempty"`
	OptionDatas      []OptionData   `json:"option-data,omitempty"`
	Relay            RelayAgent     `json:"relay,omitempty"`
	ValidLifetime    uint32         `json:"valid-lifetime,omitempty"`
	MaxValidLifetime uint32         `json:"max-valid-lifetime,omitempty"`
	MinValidLifetime uint32         `json:"min-valid-lifetime,omitempty"`
}

type Pool struct {
	Pool        string       `json:"pool,omitempty"`
	ClientClass string       `json:"client-class,omitempty"`
	OptionDatas []OptionData `json:"option-data,omitempty"`
}

type Reservation4 struct {
	HWAddress   string       `json:"hw-address,omitempty"`
	IPAddress   string       `json:"ip-address,omitempty"`
	OptionDatas []OptionData `json:"option-data,omitempty"`
}

type RelayAgent struct {
	IPAddresses []string `json:"ip-addresses"`
}

type DHCP6Config struct {
	Path  string `json:"-"`
	DHCP6 DHCP6  `json:"Dhcp6"`
}

type DHCP6 struct {
	InterfacesConfig InterfacesConfig `json:"interfaces-config,omitempty"`
	LeaseDatabase    LeaseDatabase    `json:"lease-database,omitempty"`
	ValidLifetime    uint32           `json:"valid-lifetime,omitempty"`
	MaxValidLifetime uint32           `json:"max-valid-lifetime,omitempty"`
	MinValidLifetime uint32           `json:"min-valid-lifetime,omitempty"`
	ClientClasses    []ClientClass    `json:"client-classes,omitempty"`
	OptionDatas      []OptionData     `json:"option-data,omitempty"`
	Loggers          []Logger         `json:"loggers,omitempty"`
	Subnet6s         []Subnet6        `json:"subnet6,omitempty"`
}

type Subnet6 struct {
	ID               uint32         `json:"id,omitempty"`
	Subent           string         `json:"subnet,omitempty"`
	Pools            []Pool         `json:"pools,omitempty"`
	PDPools          []PDPool       `json:"pd-pools,omitempty"`
	Reservations     []Reservation6 `json:"reservations,omitempty"`
	ClientClass      string         `json:"client-class,omitempty"`
	OptionDatas      []OptionData   `json:"option-data,omitempty"`
	Relay            RelayAgent     `json:"relay,omitempty"`
	ValidLifetime    uint32         `json:"valid-lifetime,omitempty"`
	MaxValidLifetime uint32         `json:"max-valid-lifetime,omitempty"`
	MinValidLifetime uint32         `json:"min-valid-lifetime,omitempty"`
}

type PDPool struct {
	Prefix       string       `json:"prefix,omitempty"`
	PrefixLen    uint32       `json:"prefix-len,omitempty"`
	DelegatedLen uint32       `json:"delegated-len,omitempty"`
	ClientClass  string       `json:"client-class,omitempty"`
	OptionDatas  []OptionData `json:"option-data,omitempty"`
}

type Reservation6 struct {
	HWAddress   string       `json:"hw-address,omitempty"`
	IPAddresses []string     `json:"ip-addresses,omitempty"`
	OptionDatas []OptionData `json:"option-data,omitempty"`
}

type DHCPCmdRequest struct {
	Command   string      `json:"command"`
	Services  []string    `json:"service"`
	Arguments interface{} `json:"arguments"`
}

type DHCPCmdResponse struct {
	Result int    `json:"result"`
	Text   string `json:"text"`
}
