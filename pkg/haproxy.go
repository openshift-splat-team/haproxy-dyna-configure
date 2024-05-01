package pkg

import (
	"bytes"
	"fmt"

	"github.com/openshift-splat-team/haproxy-dyna-configure/data"
	haproxy "github.com/openshift-splat-team/haproxy-dyna-configure/data/haproxy"
	"github.com/openshift-splat-team/haproxy-dyna-configure/pkg/util"
	"github.com/sirupsen/logrus"
)

func createFrontend(name string, port *data.MonitorPort) *haproxy.Section {
	logrus.Infof("creating frontend %s", name)

	return &haproxy.Section{
		Type: haproxy.SectionFrontEnd,
		Name: name,
		Attributes: []string{
			"mode tcp",
			fmt.Sprintf("bind 0.0.0.0:%d", (10000 + port.Port)),
			"tcp-request content accept if { req_ssl_hello_type 1 }",
			"tcp-request inspect-delay 5000",
		},
	}
}

func createBackendSwitchingRule(baseDomain string, frontend *haproxy.Section, backend *haproxy.Section, port *data.MonitorPort) error {
	logrus.Infof("creating backend switching rule %s", backend.Name)

	var rule string
	if len(port.PathMatch) > 0 || len(port.PathPrefix) > 0 {
		rule = fmt.Sprintf("if { req.ssl_sni -m end .%s }", baseDomain)
	}

	frontend.AppendAttribute(fmt.Sprintf("use_backend %s %s", backend.Name, rule))
	return nil
}

func createBackend(name string, port *data.MonitorPort) *haproxy.Section {
	logrus.Infof("creating backend %s", name)

	backend := haproxy.Section{
		Type: haproxy.SectionBackEnd,
		Name: name,
		Attributes: []string{
			"mode tcp",
		},
	}

	for _, target := range port.Targets {
		port := port.Port
		serverName := fmt.Sprintf("%s-%d", target, port)
		server := fmt.Sprintf("server %s %s:%d check verify none", serverName, target, port)
		backend.AppendAttribute(server)
	}
	return &backend
}

func BuildDynamicConfiguration(monitorConfig *data.MonitorConfig) (string, error) {
	sections := []*haproxy.Section{}

	frontEnds := map[string]*haproxy.Section{}

	var frontEnd *haproxy.Section

	for _, monitorRange := range monitorConfig.MonitorRanges {
		for _, monitorPort := range monitorRange.MonitorPorts {
			if len(monitorPort.Targets) == 0 || len(monitorRange.BaseDomain) == 0 {
				continue
			}
			name := fmt.Sprintf("%s-%d", monitorRange.BaseDomain, monitorPort.Port)
			frontendName := fmt.Sprintf("dynaconfig-fe-%d", monitorPort.Port)

			var exists bool

			if frontEnd, exists = frontEnds[frontendName]; !exists {
				frontEnd = createFrontend(frontendName, &monitorPort)
				frontEnds[frontendName] = frontEnd
			}

			backEnd := createBackend(name, &monitorPort)

			err := createBackendSwitchingRule(monitorRange.BaseDomain, frontEnd, backEnd, &monitorPort)
			if err != nil {
				return "", fmt.Errorf("unable to create backend switching rules: %w", err)
			}

			sections = append(sections, backEnd)
		}
	}
	for _, frontEnd := range frontEnds {
		sections = append(sections, frontEnd)
	}

	buf := &bytes.Buffer{}
	for _, section := range sections {
		buf = section.Serialize(buf)
	}

	return buf.String(), nil
}

func BuildTargetHAProxyConfig(monitorConfig *data.MonitorConfig) (string, string, error) {
	buffer := bytes.Buffer{}
	buffer.WriteString(monitorConfig.HaproxyHeader)

	dynamicConfig, err := BuildDynamicConfiguration(monitorConfig)
	if err != nil {
		return "", "", fmt.Errorf("unable to build the dynamic configuration: %v", err)
	}

	buffer.WriteString(dynamicConfig)

	outStr := buffer.String()
	hash := util.GenerateSHA512Hash(buffer.Bytes())
	return outStr, hash, nil
}
