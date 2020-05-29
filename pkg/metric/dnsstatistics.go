package metric

import (
	"time"
)

type DNSStatistics struct {
	Server Server `xml:"server"`
	Views  []View `xml:"views>view"`
}

type Server struct {
	BootTime    time.Time  `xml:"boot-time"`
	ConfigTime  time.Time  `xml:"config-time"`
	CurrentTime time.Time  `xml:"current-time"`
	Counters    []Counters `xml:"counters"`
}

type View struct {
	Name     string     `xml:"name,attr"`
	Cache    []Gauge    `xml:"cache>rrset"`
	Counters []Counters `xml:"counters"`
}

type Counters struct {
	Type     string    `xml:"type,attr"`
	Counters []Counter `xml:"counter"`
}

type Counter struct {
	Name    string `xml:"name,attr"`
	Counter uint64 `xml:",chardata"`
}

type Gauge struct {
	Name  string `xml:"name"`
	Gauge int64  `xml:"counter"`
}
