package pkg

import (
	"strings"
	"testing"

	"github.com/andreyvit/diff"
	"github.com/openshift-splat-team/haproxy-dyna-configure/data"
)

const (
	baseDomain  = "example.com"
	goodBackend = `
backend backend-1
  mode tcp
  server 192.168.1.4-443 192.168.1.4:443 check verify none
  server 192.168.1.5-443 192.168.1.5:443 check verify none
  server 192.168.1.6-443 192.168.1.6:443 check verify none
`
	goodFrontend = `
frontend frontend-1
  mode tcp
  bind 0.0.0.0:10443
  tcp-request content accept if { req_ssl_hello_type 1 }
  tcp-request inspect-delay 5000
`
	goodHttpsFrontEnd = `
frontend frontend-1
  mode tcp
  bind 0.0.0.0:10443
  tcp-request content accept if { req_ssl_hello_type 1 }
  tcp-request inspect-delay 5000
  use_backend backend-1 if { req.ssl_sni -m end .example.com }
`

	goodAPIFrontEnd = `
frontend frontend-1
  mode tcp
  bind 0.0.0.0:16443
  tcp-request content accept if { req_ssl_hello_type 1 }
  tcp-request inspect-delay 5000
  use_backend backend-1 if { req.ssl_sni -m end .example.com }
`
)

var (
	targets = []string{"192.168.1.4", "192.168.1.5", "192.168.1.6"}
	apiPort = data.MonitorPort{
		Port:      6443,
		Targets:   targets,
		PathMatch: "api",
	}
	appsPort = data.MonitorPort{
		Port:       443,
		Targets:    targets,
		PathPrefix: "*.apps",
	}
)

func expectMatch(t *testing.T, presentedStr, expectedStr string) {
	if strings.Compare(presentedStr, expectedStr) != 0 {

		t.Fatal("test output does not match expected output: ", diff.LineDiff(presentedStr, expectedStr))
		//t.Fatalf("%s(%d)\nshould match:\n%s(%d)", presentedStr, len(presentedStr), expectedStr, len(expectedStr))
	}
}

func TestCreateBackend(t *testing.T) {
	section := createBackend("backend-1", &appsPort)
	sectionBytes := section.Serialize(nil)
	sectionStr := sectionBytes.String()

	expectMatch(t, sectionStr, goodBackend)
}

func TestCreateFrontend(t *testing.T) {
	section := createFrontend("frontend-1", &appsPort)
	sectionBytes := section.Serialize(nil)
	sectionStr := sectionBytes.String()

	expectMatch(t, sectionStr, goodFrontend)
}

func TestCreateBackendRules(t *testing.T) {
	tests := []struct {
		name     string
		port     data.MonitorPort
		expected string
	}{
		{
			name:     "create backend switching rule for HTTPS",
			port:     appsPort,
			expected: goodHttpsFrontEnd,
		},
		{
			name:     "create backend switching rule for API",
			port:     apiPort,
			expected: goodAPIFrontEnd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := createBackend("backend-1", &tt.port)
			frontend := createFrontend("frontend-1", &tt.port)

			if err := createBackendSwitchingRule(baseDomain, frontend, backend, &tt.port); err != nil {
				t.Fatal(err)
			}
			expectMatch(t, frontend.Serialize(nil).String(), tt.expected)
		})
	}

}

/*func TestBuildDynamicConfiguration(t *testing.T) {
	config, err := BuildDynamicConfiguration(&goodMonitorConfig.MonitorConfig)
	if err != nil {
		t.Fatal(err)
	}
	expectMatch(t, config, goodDynamicConfig)

}

/*func TestBuildTargetHAProxyConfig(t *testing.T) {
	config, hash, err := BuildTargetHAProxyConfig(&goodMonitorConfig.MonitorConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("checking config hash")
	expectMatch(t, hash, goodTargetConfigHash)

	t.Log("checking config")
	expectMatch(t, config, goodTargetConfig)
}*/
