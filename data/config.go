package data

type MonitorPort struct {
	Port       int64  `yaml:"port"`
	Name       string `yaml:"name"`
	Targets    []string
	PathPrefix string `yaml:"path-prefix"`
	PathMatch  string `yaml:"path-match"`
	Protocol   string `yaml:"protocol"`
}

type MonitorRange struct {
	IpAddressStart string        `yaml:"ip-address-start"`
	IpAddressEnd   string        `yaml:"ip-address-end"`
	MonitorPorts   []MonitorPort `yaml:"monitor-ports"`
	BaseDomain     string
}

type MonitorConfig struct {
	MonitorRanges []MonitorRange `yaml:"monitor-ranges"`
	HaproxyHeader string         `yaml:"haproxy-header"`
	CheckTimeout  int            `yaml:"check-timeout"`
	SubnetsJson   string         `yaml:"subnets-json-path"`
	BaseDomain    string         `yaml:"base-domain"`
}

type MonitorConfigSpec struct {
	MonitorConfig MonitorConfig `yaml:"monitor-config"`
}
